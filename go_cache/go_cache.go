package go_cache

import (
	"fmt"
	"log/slog"
	"sync"
)

//                             是
// 接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
//                 |  否                         是
//                 |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
//                             |  否
//                             |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶

// 步骤 2：
//
// 使用一致性哈希选择节点        是                                    是
//     |-----> 是否是远程节点 -----> HTTP 客户端访问远程节点 --> 成功？-----> 服务端返回返回值
//                     |  否                                    ↓  否
//                     |----------------------------> 回退到本地节点处理。

var (
	// protect groups map
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

// Group is a cache namespace and associated data loaded spread over.
type Group struct {
	name      string // namespace's name
	getter    Getter // callback when miss data
	mainCache cache

	peers PeerPicker // get value from peer cache
}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}

	mu.Lock()
	defer mu.Unlock()
	groups[name] = g

	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	defer mu.RUnlock()
	return groups[name]
}

// Get value for a key from main cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		slog.Info(fmt.Sprintf("cache hit: %s", key))
		return v, nil
	}

	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err := g.getFromPeer(peer, key); err == nil {
				return value, nil
			} else {
				slog.Info("[GeeCache] Failed to get from peer", "peer", err)
			}
		}
	}

	return g.getLocally(key)
}

// ************************** get value in local other source

// use in single machie
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: bytes}
	g.populateCache(key, value)
	return value, nil
}

// use in single machine
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// ************************** get value in peer's cache

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

// ****************************************************************

// Getter get value from other source, when key is not exist. It can be defined by user.
type Getter interface {
	Get(key string) ([]byte, error)
}

// 函数类型实现某个接口，称为接口函数。调用者可以传入函数，也可传入结构体
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}
