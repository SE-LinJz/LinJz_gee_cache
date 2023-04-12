package main

import (
	"LinJz_gee_cache/geecache"
	"fmt"
	"log"
	"net/http"
)

// 同样地，我们使用map模拟了数据源db
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// 创建一个名为scores的Group，若缓存为空，回调函数会从db中换取数据并返回
// 使用http.ListenAndServe在localhost:9999端口启动了HTTP服务
func main() {
	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		},
	))

	addr := "localhost:9999"
	peers := geecache.NewHTTPPool(addr)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
}
