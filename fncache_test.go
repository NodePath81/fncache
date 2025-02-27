package fncache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/NodePath81/fncache"
)

type mockCacheLayer[K comparable, V any] struct {
	store  map[K]V
	getCnt int
	setCnt int
}

func (m *mockCacheLayer[K, V]) Get(ctx context.Context, key K) (V, error) {
	m.getCnt++
	val, ok := m.store[key]
	if !ok {
		var zero V
		return zero, errors.New("not found")
	}
	return val, nil
}

func (m *mockCacheLayer[K, V]) Set(ctx context.Context, key K, value V) error {
	m.setCnt++
	if m.store == nil {
		m.store = make(map[K]V)
	}
	m.store[key] = value
	return nil
}

func TestGetCacheMiss(t *testing.T) {
	ctx := context.Background()
	mockCache := &mockCacheLayer[int, string]{}
	getCallCount := 0

	cache := fncache.NewFnCache[int, string](
		func(ctx context.Context, key int) (string, error) {
			getCallCount++
			return "value", nil
		},
		nil,
		mockCache,
		fncache.CacheConfig{
			CacheDuration: time.Minute,
		},
	)

	// First call - cache miss
	val, err := cache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if val != "value" {
		t.Errorf("Expected 'value', got %q", val)
	}
	if getCallCount != 1 {
		t.Errorf("Expected getFn to be called 1 time, got %d", getCallCount)
	}
	if mockCache.setCnt != 1 {
		t.Errorf("Expected cache.Set to be called 1 time, got %d", mockCache.setCnt)
	}

	// Second call - cache hit
	val, err = cache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if val != "value" {
		t.Errorf("Expected 'value', got %q", val)
	}
	if getCallCount != 1 {
		t.Errorf("Expected getFn to be called 1 time, got %d", getCallCount)
	}
	if mockCache.getCnt != 2 {
		t.Errorf("Expected 2 cache.Get calls, got %d", mockCache.getCnt)
	}
}

func TestGetCacheErrorPropagation(t *testing.T) {
	ctx := context.Background()
	mockCache := &mockCacheLayer[int, string]{}
	expectedErr := errors.New("data source error")

	cache := fncache.NewFnCache[int, string](
		func(ctx context.Context, key int) (string, error) {
			return "", expectedErr
		},
		nil,
		mockCache,
		fncache.CacheConfig{
			CacheDuration: time.Minute,
		},
	)

	_, err := cache.Get(ctx, 1)
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if mockCache.setCnt != 0 {
		t.Errorf("Expected no cache updates on error, got %d", mockCache.setCnt)
	}
}

func TestSetPropagation(t *testing.T) {
	ctx := context.Background()
	mockCache := &mockCacheLayer[int, string]{}
	setCallCount := 0

	cache := fncache.NewFnCache[int, string](
		nil,
		func(ctx context.Context, key int, value string) error {
			setCallCount++
			return nil
		},
		mockCache,
		fncache.CacheConfig{
			CacheDuration: time.Minute,
		},
	)

	err := cache.Set(ctx, 1, "value")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if setCallCount != 1 {
		t.Errorf("Expected setFn to be called 1 time, got %d", setCallCount)
	}
	if mockCache.setCnt != 1 {
		t.Errorf("Expected cache.Set to be called 1 time, got %d", mockCache.setCnt)
	}
}

func TestCacheLayerIntegration(t *testing.T) {
	ctx := context.Background()
	mockCache := &mockCacheLayer[string, int]{}

	cache := fncache.NewFnCache[string, int](
		func(ctx context.Context, key string) (int, error) {
			return len(key), nil
		},
		nil,
		mockCache,
		fncache.CacheConfig{
			CacheDuration: time.Minute,
		},
	)

	// First call to prime cache
	val, err := cache.Get(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	if val != 4 {
		t.Errorf("Expected 4, got %d", val)
	}

	// Directly check cache storage
	cachedVal, err := mockCache.Get(ctx, "test")
	if err != nil {
		t.Errorf("Expected value in cache: %v", err)
	}
	if cachedVal != 4 {
		t.Errorf("Cache stored incorrect value: %d", cachedVal)
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	mockCache := &mockCacheLayer[int, string]{}

	cache := fncache.NewFnCache[int, string](
		func(ctx context.Context, key int) (string, error) {
			<-ctx.Done()
			return "", ctx.Err()
		},
		nil,
		mockCache,
		fncache.CacheConfig{
			CacheDuration: time.Minute,
		},
	)

	cancel()
	_, err := cache.Get(ctx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context canceled error, got %v", err)
	}
}
