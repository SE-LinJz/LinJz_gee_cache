package geecache

import (
	"LinJz_gee_cache/geecache/consistenthash"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

// HTTPPool 作为承载节点间HTTP通信的核心数据结构（包括服务端和客户端 ）
// HTTPPool 只有两个参数，一个是self，用来记录自己的地址，包括主机名/IP和端口，另一个是basePath，作为节点间通信地址的前缀，默认是/_geecache/，那么https://example.com/_geecache/开头的请求，就用于节点间的访问。因为一个主机上还可能承载其他的服务，加一段 Path 是一个好习惯。比如，大部分网站的 API 接口，一般以 /api 作为前缀。
// 新增成员变量peers，类型是一致性哈希算法的Map，用来根据具体的key选择节点
// 新增成员变量httpGetters，映射远程节点与对应的httpGetter。每一个远程节点对应一个httpGetter，因为httpGetter与远程节点的地址baseURL有关
// HTTPPool implements PeerPicker for a pool of HTTP peers
type HTTPPool struct {
	// this peer's base URL,e.g. "https://example.net:8000"
	self        string
	basePath    string
	mu          sync.Mutex // guards peers and httpGetters
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
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

// 在GeeCache第三天，我们为HTTPPool实现了服务端功能，但通信不仅需要服务端还需要客户端，所以，接下来就要为HTTPPool实现客户端功能
// 首先创建具体的HTTP客户端类httpGetter，实现PeerGetter接口
// baseURL表示将要访问的远程节点的地址，例如http://example.com/_geecache/
type httpGetter struct {
	baseURL string
}

// Get 使用http.Get()方式获取返回值，并转换为[]bytes类型
func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key))
	res, err := http.Get(u) // 调用http.Get()会直接来到ServeHTTP()，因为在main.go中startCacheServer()的http.ListenAndServe(addr[7:], peers)传递了处理函数接口为HTTPPool，而HTTPPool中的处理函数就是ServeHTTP()
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

// Set 方法实例化了一致性哈希算法，并且添加了传入的节点，并且为每一个节点创建了一个HTTP客户端httpGetter
// Set 第三步，实现PeerPicker接口
// Set updates the pool's list of peers
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer 包装了一致性哈希算法的Get()方法，根据具体的key，选择节点，返回节点对应的HTTP客户端
// PickPeer picks a peer according to key
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

// 这两个的作用是确保这个类型实现了这个接口 如果没有实现会报错的
var _ PeerGetter = (*httpGetter)(nil)
var _ PeerPicker = (*HTTPPool)(nil)
