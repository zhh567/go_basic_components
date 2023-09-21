package main

import (
	"fmt"
	"go_cache"
	"log"
	"log/slog"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	go_cache.NewGroup("scores", 2<<10, go_cache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	addr := "localhost:8080"
	peers := go_cache.NewHTTPPool(addr)
	slog.Info(fmt.Sprintf("http server listen in: %s", addr))
	log.Fatal(http.ListenAndServe(addr, peers))
}
