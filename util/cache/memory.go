package cache

import "sync"

// MemoryCache is a simple, thread-safe in-memory key-value store.
type MemoryCache struct {
	sync.RWMutex
	data map[string][]byte
}

// Memory returns a MemoryCache, which is a simple, thread-safe in-memory
// key-value store.
func Memory() *MemoryCache {
	return &MemoryCache{
		data: make(map[string][]byte),
	}
}

func (c *MemoryCache) Get(key []byte) (value []byte, ok bool, err error) {
	c.RLock()
	defer c.RUnlock()
	value, ok = c.data[string(key)]
	return

}

func (c *MemoryCache) Set(key []byte, value []byte) error {
	c.Lock()
	defer c.Unlock()
	c.data[string(key)] = value
	return nil
}
