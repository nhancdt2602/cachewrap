# CacheWrap

Golang annotation-based & compile-time automatic caching wrapper, inspired by Python's `@lru_cache`, Spring Boot's cache annotations and Alibaba `loongsuite`

## Features

- **Zero Runtime Overhead**: Cache logic injected at compile time
- **DST-based**: Preserves formatting and comments using Decorator Syntax Tree
- **Line-preserving**: Uses `//line` directives to maintain original line numbers for debugging
- **Flexibility**: Supports global cache, named cache instances, and custom cache implementations
- **Annotation-based**: Three annotations: `cachewrap`, `cacheevict`, and `cacheclear`
- **Cache Eviction**: Automatic cache invalidation on updates and deletes
- **Prefix Support**: Namespace isolation for different entities
- **Named Caches**: Use different cache backends for different data types

## Quick Start

### 1. Add Cache Annotations

```go
// GetByID retrieves customer by ID
// cachewrap[prefix=Customer]: id
func (r *Repository) GetByID(ctx context.Context, id string) (*Customer, error) {
    // Expensive database query
    return r.db.Query(id)
}

// Update invalidates the cache
// cacheevict[prefix=Customer]: id
func (r *Repository) Update(id string, customer *Customer) error {
    return r.db.Update(id, customer)
}

// ClearAll clears all customer caches
// cacheclear[prefix=Customer]
func (r *Repository) ClearAll() {
    // Cache will be cleared automatically
}
```

### 2. Build with Automatic Instrumentation

**Using -toolexec**

```bash
# Build the cachewrap tool
make build

# Build your app with automatic cache instrumentation
go build -toolexec="./cachewrap remix" -a -o myapp main.go
```

**Debug generated code**

```bash
# View instrumented code
./cachewrap-debug your_file.go

# Instrumented code saved to .cache-build/instrument/
```

## Annotation Syntax

### 1. Cache Read (cachewrap)

```go
// Basic - global cache, function name as prefix
// cachewrap: param1, param2

// With prefix - consistent keys across CRUD
// cachewrap[prefix=Entity]: param1, param2

// With named cache instance
// cachewrap[@cacheName]: param1, param2

// Combined - named cache + prefix ✅ RECOMMENDED
// cachewrap[@cacheName, prefix=Entity]: param1, param2
```

### 2. Cache Eviction (cacheevict)

```go
// Basic eviction
// cacheevict: param1

// With prefix (matches cachewrap prefix)
// cacheevict[prefix=Entity]: param1

// With named cache (MUST match the cachewrap annotation)
// cacheevict[@cacheName, prefix=Entity]: param1
```

### 3. Cache Clear (cacheclear)

```go
// Clear all keys with prefix
// cacheclear[prefix=Entity]

// Clear from named cache
// cacheclear[@cacheName, prefix=Entity]
```

## Complete CRUD Example

```go
type CustomerRepo struct {
    db *sql.DB
}

// CREATE - no caching
func (r *CustomerRepo) Create(customer *Customer) error {
    return r.db.Insert(customer)
}

// READ - cache with prefix
// cachewrap[prefix=Customer]: id
func (r *CustomerRepo) GetByID(ctx context.Context, id string) (*Customer, error) {
    // Generated cache key: "Customer:id:123"
    return r.db.Query(id)
}

// READ - another method, same prefix
// cachewrap[prefix=Customer]: email
func (r *CustomerRepo) GetByEmail(ctx context.Context, email string) (*Customer, error) {
    // Generated cache key: "Customer:email:john@example.com"
    return r.db.QueryByEmail(email)
}

// UPDATE - evict cache (same prefix as GetByID!)
// cacheevict[prefix=Customer]: id
func (r *CustomerRepo) Update(id string, customer *Customer) error {
    // Evicts cache key: "Customer:id:123" (same as GetByID)
    return r.db.Update(id, customer)
}

// DELETE - evict cache
// cacheevict[prefix=Customer]: id
func (r *CustomerRepo) Delete(id string) error {
    // Evicts cache key: "Customer:id:123"
    return r.db.Delete(id)
}

// CLEAR - clear all customer caches
// cacheclear[prefix=Customer]
func (r *CustomerRepo) ClearAll() {
    // Clears all "Customer:*" keys (both by ID and by email)
}
```

## Named Cache Instances

Register caches at application startup:

```go
func main() {
    // Register different cache backends for different entities
    cache.RegisterCache("userCache", cache.NewMemoryCache())
    cache.RegisterCache("productCache", cache.NewMemoryCache())
    
    // Now use in annotations:
    // cachewrap[@userCache, prefix=User]: id
    // cachewrap[@productCache, prefix=Product]: id
}
```

**Example with named caches:**

```go
// User repository uses @userCache
// cachewrap[@userCache, prefix=User]: id
func (r *UserRepo) GetByID(id string) (*User, error) {
    // Stores in @userCache with key "User:id:123"
}

// cacheevict[@userCache, prefix=User]: id
func (r *UserRepo) Update(id string, user *User) error {
    // Evicts from @userCache with key "User:id:123"
}

// Product repository uses @productCache (completely separate)
// cachewrap[@productCache, prefix=Product]: id
func (r *ProductRepo) GetByID(id string) (*Product, error) {
    // Stores in @productCache with key "Product:id:123"
}
```

## Cache Key Format

```
MakeCacheKey(prefix, paramName1, value1, paramName2, value2, ...)
```

**Examples:**
- `MakeCacheKey("Customer", "id", "123")` → `"Customer:id:123"`
- `MakeCacheKey("Customer", "email", "john@ex.com")` → `"Customer:email:john@ex.com"`
- `MakeCacheKey("Product", "id", "p1", "locale", "en")` → `"Product:id:p1:locale:en"`

**Why this format?**
- **Prefix first**: Groups related cache entries (`Customer:*`)
- **Param names included**: Self-documenting keys
- **Clear delimiters**: Colon separators prevent collisions

## Benefits

✅ **Consistent Keys**: Same prefix ensures GET and EVICT use identical keys  
✅ **Namespace Isolation**: `Product:id:1` != `Customer:id:1`  
✅ **Cache Isolation**: Different entities can use different cache backends  
✅ **No Boilerplate**: Write clean business logic, caching injected automatically  
✅ **Type Safe**: Compile-time code generation  
✅ **Debugger Friendly**: `//line` directives preserve original line numbers  
✅ **CRUD Complete**: Read caching + update eviction + bulk clear  