package redis

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements CacheLayer using Redis with gob serialization.
// FnReturns must be gob-encodable.
type RedisCache[FnParams comparable, FnReturns any] struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// NewRedisCache creates a RedisCache with the specified Redis options, TTL, and key prefix.
func NewRedisCache[FnParams comparable, FnReturns any](opts *redis.Options, ttl time.Duration, prefix string) *RedisCache[FnParams, FnReturns] {
	client := redis.NewClient(opts)
	return &RedisCache[FnParams, FnReturns]{
		client: client,
		ttl:    ttl,
		prefix: prefix,
	}
}

// computeKey returns a Redis key with the configured prefix and a hex-encoded FNV hash of the params.
func computeKey[FnParams comparable](prefix string, params FnParams) string {
	hasher := fnv.New64a()
	keyStr := fmt.Sprint(params)
	hasher.Write([]byte(keyStr))
	return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(hasher.Sum(nil)))
}

// encode serializes a value using gob.
func encode[T any](v T) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decode deserializes data into v using gob.
func decode[T any](data []byte, v *T) error {
	return gob.NewDecoder(bytes.NewBuffer(data)).Decode(v)
}

// Get retrieves the cached value for the given key.
func (r *RedisCache[FnParams, FnReturns]) Get(ctx context.Context, params FnParams) (FnReturns, error) {
	var zero FnReturns
	key := computeKey(r.prefix, params)
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return zero, err
	}
	var result FnReturns
	if err := decode(data, &result); err != nil {
		return zero, err
	}
	return result, nil
}

// Set stores the value for the given key with the cache TTL.
func (r *RedisCache[FnParams, FnReturns]) Set(ctx context.Context, params FnParams, value FnReturns) error {
	key := computeKey(r.prefix, params)
	data, err := encode(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}
