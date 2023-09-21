package go_cache

// ByteView holds an immutable view of bytes.
// It is encapsulation of lru
type ByteView struct {
	b []byte // caching arbitary format data
}

func (v ByteView) Len() int {
	return len(v.b)
}

func (v ByteView) ByteSlice() []byte {
	c := make([]byte, len(v.b))
	copy(c, v.b)
	return c

}
func (v ByteView) String() string {
	return string(v.b)
}
