package geecache

// ByteView 只有一个数据成员，b []byte，b将会存储真实的缓存值，选择byte类型是为了能够支持任意的数据类型的存储，例如字符串，图片等。
// A ByteView holds an immutable view of bytes
type ByteView struct {
	b []byte
}

// Len 实现Len() int方法，我们在lru.Cache的实现中，要求被缓存对象必须实现Value接口，即Len() int方法，返回其所占的内存大小（即Cache.cache是一个map，键是string，值是*list.Element，Element中的Value存放的是entry，entry这个结构体有个成员是Value类型的，这个也就是ByteView）
// Len returns the view's length
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice b是只读的，使用ByteSlice()方法返回一个拷贝，防止缓存值被外部程序修改
// ByteSlice returns a copy of the data as a byte slice
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// String returns the data as a string, making a copy if necessary
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
