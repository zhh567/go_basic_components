package go_cache

import (
	"fmt"
	"go_cache/consistenthash"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// ********************** server end *************************

const (
	defaultBasePath = "/_go_cache/"
	defaultReplicas = 50 // the mutiple of virtual nodes relative to real nodes
)

type HTTPPool struct {
	poolName string // used in log
	basePath string // domain name and port. e.g. "https://example.net:8000"

	// for distributed use
	mu          sync.Mutex             // guards late two variables
	peers       *consistenthash.Map    // select node by key. map[hashValueOfKey]key
	httpGetters map[string]*httpGetter // map peer's baseURL to httpGetter. keyed by e.g. "http://10.0.0.2:8008"
}

func NewHTTPPool(name string) *HTTPPool {
	return &HTTPPool{
		poolName: name,
		basePath: defaultBasePath,
	}
}

// Set updates the HTTPPool's list of peers.
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

// PickPeer picks a peer according key
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if peer := p.peers.Get(key); peer != "" && peer != p.poolName {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

func (p *HTTPPool) Log(format string, v ...interface{}) {
	slog.Info(fmt.Sprintf("[Server %s] %s", p.poolName, fmt.Sprintf(format, v...)))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		http.Error(w, fmt.Sprintf("request [%s] not match basepath [%s]", r.URL.Path, p.basePath), http.StatusBadRequest)
		return
	}
	p.Log("%s %s", r.Method, r.URL.Path)

	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "must have group name and key", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// the operation of truly get value
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

// ********************** client end *************************

type httpGetter struct {
	baseURL string
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	url := fmt.Sprintf("%v%v/%v",
		h.baseURL, url.QueryEscape(group), url.QueryEscape(key))
	res, err := http.Get(url)
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

// check if struct `httpGetter` is interface `PeerGetter`
var _ PeerGetter = (*httpGetter)(nil)
