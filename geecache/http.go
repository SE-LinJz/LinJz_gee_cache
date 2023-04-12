package geecache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const defaultBasePath = "/_geecache/"

// HTTPPool 作为承载节点间HTTP通信的核心数据结构（包括服务端和客户端 ）
// HTTPPool 只有两个参数，一个是self，用来记录自己的地址，包括主机名/IP和端口，另一个是basePath，作为节点间通信地址的前缀，默认是/_geecache/，那么https://example.com/_geecache/开头的请求，就用于节点间的访问。因为一个主机上还可能承载其他的服务，加一段 Path 是一个好习惯。比如，大部分网站的 API 接口，一般以 /api 作为前缀。
// HTTPPool implements PeerPicker for a pool of HTTP peers
type HTTPPool struct {
	// this peer's base URL,e.g. "https://example.net:8000"
	self     string
	basePath string
}

// NewHTTPPool initializes an HTTP pool of peers
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...any) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP 的实现逻辑比较简单，首先判断访问路径的前缀是否是basePath，不是返回错误，注意，r.URL.Path是端口后面的那一段，r.URL还有一个字段是Host，保存的是host or host:port，所以只需要拿HTTPPool的basePath去比较就可以，不用拿self去比较
// 我们约定访问路径格式为/<basepath>/<groupname>/<key>，通过groupname得到group实例，再使用group.Get(key)获取缓存数据，最后使用w.Write()将缓存值作为httpResponse的body返回
// ServeHTTP handle all http requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) { // strings.HasPrefix判断r.URL.Path的前缀是否是p.basePath
		panic("HTTPPool serving unexpected path:" + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2) // r.URL.Path[len(p.basePath):]返回r.URL.Path从len(p.basePath)开始到len(r.URL.Path.)减1的位置，然后再切成两份变成数组
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = w.Write(view.ByteSlice()) // view是ByteView类型的，而w.Write需要byte[]类型的，所以给它转换成byte[]类型
	if err != nil {
		http.Error(w, "response write error", http.StatusInternalServerError)
		return
	}
}
