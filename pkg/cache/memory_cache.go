package cache

import (
	"strings"
	"sync"
)

type MemoryCache struct {
	items map[string]interface{}
	mu    sync.RWMutex
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		items: make(map[string]interface{}),
	}
}

func (m *MemoryCache) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.items[key]
	return val, ok
}

func (m *MemoryCache) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key] = value
}

func (m *MemoryCache) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.items, key)
}

func (m *MemoryCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]interface{})
}

func (m *MemoryCache) ClearPrefix(prefix string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	prefixWithColon := prefix + ":"
	for key := range m.items {
		if strings.HasPrefix(key, prefixWithColon) {
			delete(m.items, key)
		}
	}
}
