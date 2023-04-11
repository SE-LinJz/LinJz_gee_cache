package LinJz_gee_cache

import (
	"fmt"
	"log"
	"sync"
)

// Getter 定义接口Getter和回调函数Get(key string)([]byte,error),参数为key，返回值为[]byte。
// A Getter loads data for a key
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 定义接口GetterFunc，并实现Getter接口的Get方法
// A GetterFunc implements Getter with a function
type GetterFunc func(key string) ([]byte, error)

// Get 函数类型实现某一个接口，称之为接口型函数，方便使用者再调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数
// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 一个Group可以认为是一个缓存的命名空间，每个Group拥有一个唯一的名称name，比如可以创建三个Group，缓存学生的成绩命名为scores，换成学生信息的命名为info，缓存学生课程的命名为courses
// 第二个属性是getter Getter，即缓存未命中时获取源数据的回调（callback）
// 第三个属性是mainCache cache，即一开始实现的并发缓存
// A Group is a cache namespace and associated data loaded spread over
type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// NewGroup 构建函数NewGroup用来实例化Group，并且将group存储在全局变量groups中
// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:   name,
		getter: getter,
		mainCache: cache{
			cacheBytes: cacheBytes,
		},
	}
	groups[name] = g
	return g
}

// GetGroup 用来特定名称的Group，这里使用了只读锁RLock()，因为不涉及任何冲突变量的写操作。
// GetGroup returns the named group previously created with NewGroup, or nil if there's no such group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// Get 接下来是 GeeCache 最为核心的方法Get
// Get方法实现了上述所说的流程（1）和（3），即检查是否有缓存，有则返回（1），无则调用`回调函数`，获取值并添加到缓存然后再返回缓存值（3），（2）是从远程节点获取，由于第二天是实现单机，暂时还没有远程节点
// 流程（1）：从mainCache中查找缓存，如果存在则返回缓存值
// 流程（3）：缓存不存在，则调用load方法，load调用getLocally（分布式场景下会调用getFromPeer从其他节点获取），getLocally调用用户回调函数g.getter.Get()获取源数据，并且将源数据添加到缓存mainCache中（通过populateCache方法）
// Get value for a key from cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	// 没有击中缓存
	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	return g.getLocally(key)
}

// 调用用户回调函数g.getter.Get()获取源数据，并且将源数据添加到缓存mainCache中（通过populateCache方法）
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	value := ByteView{b: cloneBytes(bytes)} // b是只读的，使用ByteSlice()方法返回一个拷贝，防止缓存值被外部程序修改
	g.populateCache(key, value)
	return value, nil
}

// 将源数据添加到缓存mainCache中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
