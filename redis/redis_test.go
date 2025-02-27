package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/NodePath81/fncache/redis"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	ID   int
	Name string
	Data []byte
}

func TestRedisCache(t *testing.T) {
	ctx := context.Background()

	// Setup Redis client - skip tests if Redis is not available
	redisOpts := &goredis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}

	client := goredis.NewClient(redisOpts)
	_, err := client.Ping(ctx).Result()
	if err != nil {
		t.Skip("Skipping Redis tests: Redis server not available")
		return
	}
	defer client.Close()

	// Clean test database
	client.FlushDB(ctx)

	t.Run("Get and Set", func(t *testing.T) {
		cache := redis.NewRedisCache[string, string](redisOpts, time.Minute, "test")

		// Set a value
		err := cache.Set(ctx, "key1", "value1")
		require.NoError(t, err)

		// Get the value back
		val, err := cache.Get(ctx, "key1")
		require.NoError(t, err)
		assert.Equal(t, "value1", val)
	})

	t.Run("Cache Miss", func(t *testing.T) {
		cache := redis.NewRedisCache[string, string](redisOpts, time.Minute, "test")

		_, err := cache.Get(ctx, "nonexistent")
		require.Error(t, err)
		assert.True(t, errors.Is(err, goredis.Nil))
	})

	t.Run("TTL Expiration", func(t *testing.T) {
		// Use a very short TTL
		cache := redis.NewRedisCache[string, string](redisOpts, 500*time.Millisecond, "test")

		err := cache.Set(ctx, "expiring", "value")
		require.NoError(t, err)

		// Verify it exists
		val, err := cache.Get(ctx, "expiring")
		require.NoError(t, err)
		assert.Equal(t, "value", val)

		// Wait for expiration
		time.Sleep(600 * time.Millisecond)

		// Should be gone now
		_, err = cache.Get(ctx, "expiring")
		require.Error(t, err)
		assert.True(t, errors.Is(err, goredis.Nil))
	})

	t.Run("Complex Types", func(t *testing.T) {
		cache := redis.NewRedisCache[int, TestStruct](redisOpts, time.Minute, "test")

		testData := TestStruct{
			ID:   123,
			Name: "Test Item",
			Data: []byte("binary data"),
		}

		err := cache.Set(ctx, 42, testData)
		require.NoError(t, err)

		retrieved, err := cache.Get(ctx, 42)
		require.NoError(t, err)
		assert.Equal(t, testData.ID, retrieved.ID)
		assert.Equal(t, testData.Name, retrieved.Name)
		assert.Equal(t, testData.Data, retrieved.Data)
	})

	t.Run("Different Prefixes", func(t *testing.T) {
		cache1 := redis.NewRedisCache[string, string](redisOpts, time.Minute, "prefix1")
		cache2 := redis.NewRedisCache[string, string](redisOpts, time.Minute, "prefix2")

		// Set same key in both caches
		err := cache1.Set(ctx, "same-key", "value1")
		require.NoError(t, err)

		err = cache2.Set(ctx, "same-key", "value2")
		require.NoError(t, err)

		// Values should be different due to different prefixes
		val1, err := cache1.Get(ctx, "same-key")
		require.NoError(t, err)
		assert.Equal(t, "value1", val1)

		val2, err := cache2.Get(ctx, "same-key")
		require.NoError(t, err)
		assert.Equal(t, "value2", val2)
	})

	t.Run("Encoding Error", func(t *testing.T) {
		// Create a type that can't be gob encoded
		type UnencodableType struct {
			Ch chan int
		}

		cache := redis.NewRedisCache[string, UnencodableType](redisOpts, time.Minute, "test")

		err := cache.Set(ctx, "bad-value", UnencodableType{Ch: make(chan int)})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "gob")
	})
}

// TestRedisIntegration tests the Redis cache with the actual Redis server
// This is more of an integration test than a unit test
func TestRedisIntegration(t *testing.T) {
	ctx := context.Background()

	redisOpts := &goredis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	}

	client := goredis.NewClient(redisOpts)
	_, err := client.Ping(ctx).Result()
	if err != nil {
		t.Skip("Skipping Redis integration tests: Redis server not available")
		return
	}
	defer client.Close()

	// Clean test database
	client.FlushDB(ctx)

	cache := redis.NewRedisCache[string, int](redisOpts, time.Minute, "integration")

	// Test multiple operations
	for i := 0; i < 10; i++ {
		key := "key" + string(rune('0'+i))
		err := cache.Set(ctx, key, i*100)
		require.NoError(t, err)
	}

	// Verify all values
	for i := 0; i < 10; i++ {
		key := "key" + string(rune('0'+i))
		val, err := cache.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, i*100, val)
	}
}
