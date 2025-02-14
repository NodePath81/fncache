package memory

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryCache_SetGet(t *testing.T) {
	cache := NewInMemoryCache[string, int](500 * time.Millisecond)
	defer cache.Stop()
	ctx := context.Background()
	key, value := "foo", 42

	if err := cache.Set(ctx, key, value); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := cache.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != value {
		t.Errorf("Expected %d, got %d", value, got)
	}
}

func TestInMemoryCache_Expiration(t *testing.T) {
	cache := NewInMemoryCache[string, int](100 * time.Millisecond)
	defer cache.Stop()
	ctx := context.Background()
	key, value := "bar", 100

	if err := cache.Set(ctx, key, value); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get before expiration.
	if got, err := cache.Get(ctx, key); err != nil || got != value {
		t.Fatalf("Expected %d before expiration, got %v (err: %v)", value, got, err)
	}

	// Wait for expiration.
	time.Sleep(150 * time.Millisecond)
	if got, err := cache.Get(ctx, key); err == nil {
		t.Errorf("Expected error after expiration, got %d", got)
	}
}

func TestInMemoryCache_Delete(t *testing.T) {
	cache := NewInMemoryCache[string, int](500 * time.Millisecond)
	defer cache.Stop()
	ctx := context.Background()
	key, value := "baz", 7

	if err := cache.Set(ctx, key, value); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if got, err := cache.Get(ctx, key); err != nil || got != value {
		t.Fatalf("Expected %d, got %v (err: %v)", value, got, err)
	}

	cache.Delete(ctx, key)
	if got, err := cache.Get(ctx, key); err == nil {
		t.Errorf("Expected error after deletion, got %d", got)
	}
}

func TestInMemoryCache_Stop(t *testing.T) {
	cache := NewInMemoryCache[string, int](500 * time.Millisecond)
	// Ensure Stop does not panic.
	cache.Stop()
}
