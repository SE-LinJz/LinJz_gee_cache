package lru

import (
	"fmt"
	"reflect"
	"testing"
)

type String string

// String实现了Value的全部方法，即实现了Value接口
func (d String) Len() int {
	return len(d)
}

func TestCache_Get(t *testing.T) {
	// 这里需要注意一下，Add方法会判断maxBytes和nowBytes的大小，当maxBytes不为0且小于nowBytes的时候，会进行缓存删除，但这里maxBytes为0，所以也就会进行缓存删除，但是如果换成其他不为0的数，就需要进行判断是否需要缓存删除了，我们这里举的例子，key是4个字节，value是4个字节，所以nowBytes是8个字节，所以maxBytes至少得是8个字节，才能保证有一个数据缓存
	lru := New(int64(0), nil)
	lru.Add("key1", String("1234"))
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatal("cache hit key1=1234 failed")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatal("cache miss key2 failed")
	}
}

// 测试，当使用内存超过设定值时，是否会触发"无用"节点的移除
func TestCache_RemoveOldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := "value1", "value2", "value3"
	ca := len(k1 + k2 + v1 + v2)
	lru := New(int64(ca), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))
	// k1因为内存问题被删掉了，所以找不到k1的
	if _, ok := lru.Get("key1"); ok || lru.Length() != 2 {
		t.Fatalf("RemoveOldest key1 falied")
	}
}

// 测试回调函数是否能被调用

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value Value) {
		keys = append(keys, key)
	}

	lru := New(int64(10), callback)
	lru.Add("key1", String("123456"))
	lru.Add("k2", String("v2"))
	lru.Add("k3", String("v3"))
	lru.Add("k4", String("v4"))

	expect := []string{"key1", "k2"}

	if !reflect.DeepEqual(expect, keys) {
		fmt.Println(expect, keys)
		t.Fatalf("Call OnEvicted failed,expect keys equals to %s", expect)
	}
}
