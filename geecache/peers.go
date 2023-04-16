package geecache

// 在这里，抽象出两个接口

// PeerPicker 的PickPeer()方法用于根据传入的key选择相应节点的PeerGetter，PeerGetter就对应于上述流程中的HTTP客户端。
// PeerPicker is the interface that must be implemented to locate the peer that owns a specific key
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter 的Get()方法用于从对应的group查找缓存值。
// PeerGetter is the interface that must be implemented by a peer
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
