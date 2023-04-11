package LinJz_gee_cache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGetter(t *testing.T) {
	// 这里的GetterFunc(func(key string) ([]byte, error) {
	//		return []byte(key), nil
	//	})这一段是类型转换，将后面的函数转换成GetterFunc类型
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	// reflect.DeepEqual(v, expect)，利用反射判断两个数组是否相同
	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

// 在这个测试用例中，我们主要测试了2种情况
// （1）在缓存为空的情况下，能够通过回调函数获取源数据
// （2）在缓存已经存在的情况下，是否直接从缓存中获取，为了实现这一点，使用了loadCounts同济某个键调用回调函数的次数，如果次数大于1，则表示调用了多次回调函数，没有缓存。
func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))  //使用了loadCounts同济某个键调用回调函数的次数，如果次数大于1，则表示调用了多次回调函数，没有缓存。
	gee := NewGroup("scores", 2<<10, GetterFunc( // 2<<10表示2乘以2的10次方相当于2的11次方，GetterFunc回调函数并进行了类型转换
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value")
		} // load from callback function，第一次从缓存中换取不到，会自动调用回调函数获取源数据并且缓存
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 { // loadCounts[k] > 1说明同一个键调用回调函数多次，注意，我们目前还没有进行任何缓存过期删除操作，所以目前是缓存永久的，所以只会调用一次回调函数，之后都不需要调用回调函数了，可直接从缓存中换取
			t.Fatalf("cache %s miss", k)
		} // cache hit
	}
	if view, err := gee.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}
