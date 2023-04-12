package lru

import "container/list"

type Cache struct {
	maxBytes int64                    // 允许使用的最大内存
	nowBytes int64                    // 当前使用的内存
	ll       *list.List               // Go 语言标准库实现的双向链表list.List
	cache    map[string]*list.Element // 键是字符串，值是双向链表中对应节点的指针，list.Element是Go语言标准库实现的双向链表节点
	// optional and executed when an entry is purged 可选，并在清除条目时执行下面这个方法(回调函数)
	OnEvicted func(Key string, value Value) // 某条记录被移除时的回调函数，可以为 nil，即可以没有
}

// 键值对 entry 是双向链表节点的数据类型，即Element中的Value存放的东西，在链表中仍保存每个值对应的 key 的好处在于，淘汰队首节点时，需要用 key 从字典中删除对应的映射。
type entry struct {
	key   string
	value Value
}

// Value 为了通用性，我们允许值是实现了 Value 接口的任意类型，该接口只包含了一个方法 Len() int，用于返回值所占用的内存大小。
// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New is the Constructor of Cache
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get 查找功能
// 查找主要有2个步骤，第一步是从字典中找到对应的双向链表的节点，第二步，将该节点移动到队尾
// Get look ups a key's value
func (c *Cache) Get(key string) (value Value, ok bool) {
	// 如果键对应的链表节点存在，则将对应节点移动到队尾，并返回查找到的值
	if ele, ok := c.cache[key]; ok { // 从缓存map拿到的ele是双向链表的一个节点的指针*list.Element
		c.ll.MoveToFront(ele)    // 将链表中的节点ele移动到队尾（双向链表作为队列，队首队尾是相对的，在这里约定front为队尾）
		kv := ele.Value.(*entry) // 类型转换的第二种，断言 x.( T )，第二个返回值是bool
		return kv.value, true
	}
	return
}

// RemoveOldest 删除功能
// 缓存淘汰，即移除最近最少访问的节点（队首）
// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // 拿到队首节点的指针
	if ele != nil {
		c.ll.Remove(ele)                                         // 将该节点从双向链表中删除
		kv := ele.Value.(*entry)                                 // 获取该节点Value存放的值
		delete(c.cache, kv.key)                                  // 从字典（map）c.cache删除该节点的映射关系
		c.nowBytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 更新当前所用的内存c.nowBytes
		if c.OnEvicted != nil {                                  // 如果回调函数OnEvicted存在的话，就调用回调函数
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add 新增/修改
// Add adds a value to the cache or edit a value
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		// 如果键存在，则更新对应节点的值，并将该节点移动到队尾
		c.ll.MoveToFront(ele) // 将该节点移动到链表队尾
		kv := ele.Value.(*entry)
		c.nowBytes += int64(value.Len()) - int64(kv.value.Len()) // 更新c.nowBytes
		kv.value = value                                         // 更新值
	} else {
		// 不存在则是新增场景
		ele := c.ll.PushFront(&entry{key, value})          // 队尾新增节点&entry{key,value}
		c.cache[key] = ele                                 // 在字典中添加key和节点的映射关系
		c.nowBytes += int64(len(key)) + int64(value.Len()) // 更新c.nowBytes
	}
	// 判断更新后的c.nowBytes是否比c.maxBytes大，如果超出就不断淘汰掉一些节点
	for c.maxBytes != 0 && c.maxBytes < c.nowBytes {
		c.RemoveOldest()
	}
}

// Length Len the number of cache entries
func (c *Cache) Length() int {
	return c.ll.Len()
}
