package geecache

import (
	"LinJz_gee_cache/geecache/lru"
	"sync"
)

// cache.go的实现非常简单，实例化lru，封装get和add方法，并添加互斥锁mu
type cache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

// 在add方法中，判断了c.lru是否为nil，如果等于nil再创建实例，这种方法称之为延迟初始化（Lazy Initialization），也叫做懒汉式，一个对象的延迟初始化意味着该对象的创建将会延迟至第一次使用该对象时，主要用于提高性能，并减少程序内存要求
func (c *cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}

	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok // v.(ByteView)是类型转换
	}
	return
}
