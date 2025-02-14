package memory

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"
	"weak"
)

type cacheEntry[FnReturns any] struct {
	value     weak.Pointer[FnReturns]
	expiresAt time.Time
}

type InMemoryCache[FnParams comparable, FnReturns any] struct {
	ttl   time.Duration
	cache sync.Map // maps FnParams to *cacheEntry[FnReturns]
	done  chan struct{}
	wg    sync.WaitGroup
}

func NewInMemoryCache[FnParams comparable, FnReturns any](ttl time.Duration) *InMemoryCache[FnParams, FnReturns] {
	c := &InMemoryCache[FnParams, FnReturns]{
		ttl:  ttl,
		done: make(chan struct{}),
	}
	c.wg.Add(1)
	go c.cleaner()
	return c
}

func (c *InMemoryCache[FnParams, FnReturns]) Get(ctx context.Context, params FnParams) (FnReturns, error) {
	var zero FnReturns
	val, ok := c.cache.Load(params)
	if !ok {
		return zero, errors.New("cache miss")
	}
	entry, ok := val.(*cacheEntry[FnReturns])
	if !ok {
		return zero, fmt.Errorf("unexpected cache entry type")
	}
	if time.Now().After(entry.expiresAt) {
		c.cache.Delete(params)
		return zero, errors.New("cache expired")
	}
	res := entry.value.Value()
	if res == nil {
		c.cache.Delete(params)
		return zero, errors.New("cache value collected")
	}
	return *res, nil
}

func (c *InMemoryCache[FnParams, FnReturns]) Set(ctx context.Context, params FnParams, value FnReturns) error {
	wp := weak.Make(&value)
	entry := &cacheEntry[FnReturns]{
		value:     wp,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.cache.Store(params, entry)
	runtime.KeepAlive(value)
	return nil
}

func (c *InMemoryCache[FnParams, FnReturns]) Delete(ctx context.Context, params FnParams) {
	c.cache.Delete(params)
}

func (c *InMemoryCache[FnParams, FnReturns]) Stop() {
	close(c.done)
	c.wg.Wait()
}

func (c *InMemoryCache[FnParams, FnReturns]) cleaner() {
	defer c.wg.Done()
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			c.cache.Range(func(key, value interface{}) bool {
				if entry, ok := value.(*cacheEntry[FnReturns]); ok && now.After(entry.expiresAt) {
					c.cache.Delete(key)
				}
				return true
			})
		case <-c.done:
			return
		}
	}
}
