package singleflight

import "sync"

// 在一瞬间有大量请求get(key)，而且key未被缓存或者未被缓存在当前节点 如果不用singleflight，那么这些请求都会发送远端节点或者从本地数据库读取，会造成远端节点或本地数据库压力猛增。
// 使用singleflight，第一个get(key)请求到来时，singleflight会记录当前key正在被处理，后续的请求只需要等待第一个请求处理完成，取返回值即可。
// 并发场景下如果 GeeCache 已经向其他节点/源获取数据了，那么就加锁阻塞其他相同的请求，等待请求结果，防止其他节点/源压力猛增被击穿。

// call 代表正在进行中，或已经结束的请求。使用sync.WaitGroup锁避免重入
type call struct {
	wg  sync.WaitGroup
	val any
	err error
}

// Group 是singleflight的主数据结构，管理不同key的请求(call)
type Group struct {
	mu sync.Mutex // protects m
	m  map[string]*call
}

// Do 方法，接收2个参数，第一个参数是key，第二个参数是一个函数fn。
// Do 的作用就是，针对相同的key，无论Do被调用多少次，函数fn都只会被调用一次，等待fn调用结束了，返回返回值或错误
// g.mu是保护Group的成员变量m不被并发读写而加上的锁
// 并发协程之间，并不需要消息传递，非常适合sync.WaitGroup
// wg.Add(1) 锁加1
// wg.Wait() 阻塞，直到锁被释放
// wg.Done() 锁减1
func (g *Group) Do(key string, fn func() (any, error)) (any, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()         // 如果请求正在进行中，则等待
		return c.val, c.err // 请求结束，返回结果
	}
	c := new(call)
	c.wg.Add(1)  // 发起请求前加锁
	g.m[key] = c // 添加到g.m表明key已经有对应的请求在处理
	g.mu.Unlock()

	c.val, c.err = fn() // 调用fn，发起请求
	c.wg.Done()         // 请求结束，释放锁

	g.mu.Lock()
	delete(g.m, key) // 更新g.m，为什么在请求后要删除g.m映射关系中的key，详细见下方
	g.mu.Unlock()

	return c.val, c.err // 返回结果
}

// 能否在请求结束后 不删除g.m映射关系中的key？
// 不能，原因如下：
// 1.不删除，如果key对应的值变化，所得的值还是旧值
// 2.占用内存，缓存值的存储都放在LRU中，其他地方不保存数据，不删除，占用内存，且不会淘汰

// 加锁会不会影响性能？
// 这样做的目的是提升性能，加锁的时间与访问数据源相比，可以忽略。

// 不同客户端访问相同的key会受到影响吗？比如10个用户同一时间访问一个key，只有第一个能收到结果？
// 所有用户都能收到结果，请求是在服务端阻塞的，等待某一个查询返回结果的，其余请求直接复用这个结果了。singleflight 是这个目的。

// 您好，问什么要使用waitgroup呢，实际上c.wg中同时最多只会有一个任务，使用group是不是太多了。
// 如果用信道，接受和发送需要一一对应， waitgroup 有 Add(1) 和 Done() 是一一对应的，但是可以有多个请求同时调用 Wait()，同时等待该任务结束， 一般锁和信道是做不到这一点的。
