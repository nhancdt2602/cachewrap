package cache

import (
	"fmt"
	"sync"
)

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{})
	Delete(key string)
	Clear()
}

var (
	namedCaches = make(map[string]Cache)
	registryMu  sync.RWMutex
	globalCache = make(map[string]interface{})
	mu          sync.RWMutex
)

// RegisterCache registers a named cache instance
func RegisterCache(name string, c Cache) {
	registryMu.Lock()
	defer registryMu.Unlock()
	namedCaches[name] = c
}

// GetCache retrieves a named cache by name
func GetCache(name string) Cache {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return namedCaches[name]
}

func Get(key string) (interface{}, bool) {
	mu.RLock()
	defer mu.RUnlock()
	val, ok := globalCache[key]
	return val, ok
}

func Set(key string, value interface{}) {
	mu.Lock()
	defer mu.Unlock()
	globalCache[key] = value
}

// MakeCacheKey creates a cache key from prefix and parameter name-value pairs
// Format: "Prefix:paramName1:value1:paramName2:value2:..."
// Example:
// MakeCacheKey("User", "id", "123", "email", "john@example.com")
// Result:
// "User:id:123:email:john@example.com"
func MakeCacheKey(prefix string, paramPairs ...interface{}) string {
	key := prefix
	for i := 0; i < len(paramPairs); i += 2 {
		paramName := paramPairs[i].(string)
		paramValue := paramPairs[i+1]
		key += fmt.Sprintf(":%s:%v", paramName, paramValue)
	}
	return key
}

// MakeCacheKeySimple creates a cache key with just values (backward compat)
// Format: "Prefix:value1:value2:..."
func MakeCacheKeySimple(prefix string, params ...interface{}) string {
	key := prefix
	for _, p := range params {
		key += fmt.Sprintf(":%v", p)
	}
	return key
}

func Delete(key string) {
	mu.Lock()
	defer mu.Unlock()
	delete(globalCache, key)
}

func Clear() {
	mu.Lock()
	defer mu.Unlock()
	globalCache = make(map[string]interface{})
}
