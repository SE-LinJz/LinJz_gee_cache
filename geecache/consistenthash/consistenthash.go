package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 定义了函数类型Hash,采取依赖注入的方式，允许用于代替成自定义的Hash函数，也方便测试时替换，默认为crc32.ChecksumIEEE算法
// Hash maps bytes to uint32
type Hash func(data []byte) uint32

// Map 是一致性哈希算法的主数据结构，包含4个成员变量：Hash函数hash;虚拟节点倍数replicas；哈希环keys；虚拟节点与真实节点的映射表hashMap，键是虚拟节点的哈希值，值是真实节点的名称
// Map contains all hashed keys
type Map struct {
	hash     Hash
	replicas int
	keys     []int // Sorted
	hashMap  map[int]string
}

// New 构造函数New()允许自定义虚拟节点倍数和Hash函数
// New creates a Map instance
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add 函数允许传入0或多个真实节点的名称，注意，传入的是节点（机子），而不是缓存
// 对每一个真实节点key，对应创建m.replicas个虚拟节点，虚拟节点的名称是strconv.Itoa(i) + key，即通过添加编号的方式区分不同虚拟节点
// 使用m.hash()计算虚拟节点的哈希值，使用append(m.keys，hash)添加到环上，在hashMap中增加虚拟节点和真实节点的映射关系
// 最后一步，环上的哈希值排序
// Add adds some keys to the hash
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get 第一步，计算key的哈希值
// 第二步，顺时针找到第一个匹配的虚拟节点的下标idx，从m.keys中获取到对应的哈希值。如果idx==len(m.keys)，说明应选择m.keys[0]，因为m.keys是一个环状结构，所以用取余数的方式来处理这种情况。
// 第三步，通过hashMap映射得到真实的节点
// Get gets the closest item in the hash to the provided key
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	// Binary search for appropriate replica 二分查找满足条件的下标
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashMap[m.keys[idx%len(m.keys)]] // m.keys是一个[]int，idx是一个int类型的数组下标，m.keys[idx%len(m.keys)]是一个int类型的hash值，然后m.hashMap[]是一个map[int]string，返回一个string类型的节点key
}
