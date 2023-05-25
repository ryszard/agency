package cache

import "sync"

type MemoryCache struct {
	sync.RWMutex
	data map[string][]byte
}

func Memory() *MemoryCache {
	return &MemoryCache{
		data: make(map[string][]byte),
	}
}

func (c *MemoryCache) Get(key []byte) ([]byte, error) {
	c.RLock()
	defer c.RUnlock()
	if val, ok := c.data[string(key)]; ok {
		return val, nil
	}
	return nil, NotFoundError{Key: key}
}

func (c *MemoryCache) Set(key []byte, value []byte) error {
	c.Lock()
	defer c.Unlock()
	c.data[string(key)] = value
	return nil
}
