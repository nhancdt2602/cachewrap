package main

import (
	"context"
	"fmt"

	"github.com/nhancdt2602/cachewrap/pkg/cache"
)

func main() {

	fmt.Println("=== Initializing Named Caches ===")
	cache.RegisterCache("userCache", cache.NewMemoryCache())
	fmt.Println("Registered: @userCache -> MemoryCache")
	fmt.Println()

	repo := NewRepository()
	ctx := context.Background()

	fmt.Println("=== CacheWrap Demo ===")

	// First call - hits database
	fmt.Println("1. First GetByID(\"1\"):")

	customer, _ := repo.GetByID(ctx, "1")
	fmt.Printf("   Result: %+v\n\n", customer)

	// Second call - should use cache
	fmt.Println("2. Second GetByID(\"1\") [expect cache hit]:")

	customer, _ = repo.GetByID(ctx, "1")
	fmt.Printf("   Result: %+v\n\n", customer)

	// Different parameter - hits database
	fmt.Println("3. GetByID(\"2\") [different param, expect DB hit]:")
	customer, _ = repo.GetByID(ctx, "2")
	fmt.Printf("   Result: %+v\n\n", customer)

	// Multi-parameter cache
	fmt.Println("4. GetByEmailAndName [first call]:")
	customer, _ = repo.GetByEmailAndName(ctx, "john@example.com", "John Doe")
	fmt.Printf("   Result: %+v\n\n", customer)

	// Should hit cache
	fmt.Println("5. GetByEmailAndName [second call, expect cache hit]:")
	customer, _ = repo.GetByEmailAndName(ctx, "john@example.com", "John Doe")
	fmt.Printf("   Result: %+v\n\n", customer)

	// Non-cached function
	fmt.Println("6. GetByEmail (no cachewrap annotation):")
	customer, _ = repo.GetByEmail(ctx, "john@example.com")
	fmt.Printf("   Result: %+v\n\n", customer)

	// UPDATE - evicts cache
	fmt.Println("7. Update(\"1\", ...):")
	repo.Update("1", &Customer{ID: "1", Email: "john.new@example.com", Name: "John Updated"})
	fmt.Println()

	// READ by ID - after update (should hit DB)
	fmt.Println("8. GetByID(\"1\") after update - expect DB hit:")
	customer, _ = repo.GetByID(ctx, "1")
	fmt.Printf("   Result: %+v\n\n", customer)

	// READ by email and name - after update (should hit DB)
	fmt.Println("9. GetByEmailAndName(\"john.new@example.com\", \"John Updated\") after update - expect DB hit:")
	customer, _ = repo.GetByEmailAndName(ctx, "john.new@example.com", "John Updated")
	fmt.Printf("   Result: %+v\n\n", customer)
}
