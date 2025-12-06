package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"
)

type Customer struct {
	ID    string
	Email string
	Name  string
}

type Repository struct {
	// Simulated database
	data map[string]*Customer
}

func NewRepository() *Repository {
	return &Repository{
		data: map[string]*Customer{
			"1": {ID: "1", Email: "john@example.com", Name: "John Doe"},
			"2": {ID: "2", Email: "jane@example.com", Name: "Jane Smith"},
		},
	}
}

// GetByID retrieves customer by ID
// Use default global cache instance
// cachewrap[prefix=Customer]: id
func (r *Repository) GetByID(ctx context.Context, id string) (*Customer, error) {
	// Simulate slow database query
	_, _, line, _ := runtime.Caller(0)
	fmt.Printf("[DB QUERY at line %d] GetByID(%s) - fetching from database...\n", line, id)
	time.Sleep(100 * time.Millisecond)

	customer, ok := r.data[id]
	if !ok {
		return nil, errors.New("customer not found")
	}

	return customer, nil
}

// GetByEmailAndName retrieves customer by email and name
// Use named cache instance @userCache
// cachewrap[@userCache, prefix=Customer]: email, name
func (r *Repository) GetByEmailAndName(ctx context.Context, email, name string) (*Customer, error) {
	_, _, line, _ := runtime.Caller(0)
	fmt.Printf("[DB QUERY at line %d] GetByEmailAndName(%s, %s) - fetching from database...\n", line, email, name)
	time.Sleep(100 * time.Millisecond)

	for _, customer := range r.data {
		if customer.Email == email && customer.Name == name {
			return customer, nil
		}
	}

	return nil, errors.New("customer not found")
}

// UPDATE - evicts cache
// cacheevict[prefix=Customer]: id
func (r *Repository) Update(id string, customer *Customer) error {
	r.data[id] = customer
	r.evictEmailAndName(customer.Email, customer.Name)
	fmt.Printf("[UPDATE] Updated customer %s (cache evicted)\n", id)
	return nil
}

// evictEmailAndName
// cacheevict[@userCache, prefix=Customer]: email, name
func (r *Repository) evictEmailAndName(email, name string) error {
	// do nothing
	return nil
}

// GetByEmail retrieves by email (NO CACHING - for comparison)
func (r *Repository) GetByEmail(ctx context.Context, email string) (*Customer, error) {
	_, _, line, _ := runtime.Caller(0)
	fmt.Printf("[DB QUERY at line %d] GetByEmail(%s) - NOT CACHED\n", line, email)
	time.Sleep(100 * time.Millisecond)

	for _, customer := range r.data {
		if customer.Email == email {
			return customer, nil
		}
	}

	return nil, errors.New("customer not found")
}
