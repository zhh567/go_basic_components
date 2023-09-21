package lru

import "container/list"

// Cache is a LRU cache. It is not safe for concurrent access.
// It is a LinkedListHashMap
type Cache struct {
	maxByte int64
	nBytes  int64

	ll    *list.List               // O(1) move to font or tail
	cache map[string]*list.Element // O(1) search

	// optional and executed when entry is purged
	OnEvicted func(key string, value Value)
}

// value type in DoubleLinkedList
type entry struct {
	key   string
	value Value
}
type Value interface {
	Len() int // return memory space
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxByte:   maxBytes,
		nBytes:    0,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		if kv, ok := ele.Value.(*entry); ok {
			return kv.value, true
		} else { // assert fail
			c.ll.Remove(ele)
			return nil, false
		}
	}
	return nil, false
}

func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		kv := ele.Value.(*entry)
		c.nBytes -= int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key: key, value: value})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}

	for c.maxByte != 0 && c.maxByte < c.nBytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
