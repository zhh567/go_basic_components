package go_cache_test

import (
	"fmt"
	"go_cache"
	"log/slog"
	"testing"
)

func TestGet(t *testing.T) {
	var db = map[string]string{
		"Tom":  "630",
		"Jack": "589",
		"Sam":  "567",
	}
	loadCounts := make(map[string]int, len(db))

	getterFunc := func(key string) ([]byte, error) {
		slog.Info("[SlowDB] search", "key", key)
		if v, ok := db[key]; ok {
			if _, ok := loadCounts[key]; !ok {
				loadCounts[key] = 0
			}
			loadCounts[key]++
			return []byte(v), nil
		}
		return nil, fmt.Errorf("%s not exist", key)
	}

	g := go_cache.NewGroup("score", 1024, go_cache.GetterFunc(getterFunc))

	for k, v := range db {
		// load from callback
		if view, err := g.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		}
		// cache hit
		if _, err := g.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	if view, err := g.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}
