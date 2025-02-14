package fncache

import (
	"context"
	"time"
)

type FnGetType[FnParams comparable, FnReturns any] func(context.Context, FnParams) (FnReturns, error)
type FnSetType[FnParams comparable, FnReturns any] func(context.Context, FnParams, FnReturns) error

// concurrent safe cache layer
type CacheLayer[FnParams comparable, FnReturns any] interface {
	Get(ctx context.Context, params FnParams) (FnReturns, error)
	Set(ctx context.Context, params FnParams, value FnReturns) error
}

type FnCache[FnParams comparable, FnReturns any] struct {
	CacheConfig

	cache CacheLayer[FnParams, FnReturns]

	getFn FnGetType[FnParams, FnReturns]
	setFn FnSetType[FnParams, FnReturns]
}

func NewFnCache[FnParams comparable, FnReturns any](getFn FnGetType[FnParams, FnReturns], setFn FnSetType[FnParams, FnReturns], cacheLayer CacheLayer[FnParams, FnReturns], config CacheConfig) *FnCache[FnParams, FnReturns] {
	return &FnCache[FnParams, FnReturns]{
		cache:       cacheLayer,
		CacheConfig: config,
		getFn:       getFn,
		setFn:       setFn,
	}
}

type CacheConfig struct {
	CacheDuration      time.Duration
	CacheCheckInterval time.Duration
}

func (c *FnCache[FnParams, FnReturns]) Get(ctx context.Context, params FnParams) (FnReturns, error) {
	result, err := c.cache.Get(ctx, params)
	if err == nil {
		return result, nil
	}

	result, err = c.getFn(ctx, params)

	if err == nil {
		err = c.cache.Set(ctx, params, result)
	}

	return result, err
}

func (c *FnCache[FnParams, FnReturns]) Set(ctx context.Context, params FnParams, value FnReturns) error {
	if c.setFn == nil {
		return nil
	}

	err := c.setFn(ctx, params, value)
	if err != nil {
		return err
	}

	return c.cache.Set(ctx, params, value)
}
