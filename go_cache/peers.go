// implement distrubuted nodes interact
package go_cache

// PeerPicker must be implemented to locate
// select a `PeerGetter` by key
type PeerPicker interface {
	PickPeer(key string) (PeerGetter, bool)
}

// PeerGetter must be implemented by a peer
// PeerGetter map a node. The `Get()` search cached value from group
type PeerGetter interface {
	Get(group string, key string) ([]byte, error)
}
