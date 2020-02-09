// Package goredisstore offers Redis-based store implementation for miser using go-redis.
package goredisstore // import "github.com/pantianying/miser/store/goredisstore"

import (
	"strings"
	"time"

	"github.com/go-redis/redis"
)

const (
	redisCASMissingKey = "key does not exist"
	redisCASScript     = `
local v = redis.call('get', KEYS[1])
if v == false then
  return redis.error_reply("key does not exist")
end
if v ~= ARGV[1] then
  return 0
end
redis.call('setex', KEYS[1], ARGV[3], ARGV[2])
return 1
`
)

// GoRedisStore implements a Redis-based store using go-redis.
type GoRedisStore struct {
	client *redis.Client
	prefix string
}

// Depends on Redis 2.6+ for EVAL support.
func New(client *redis.Client, keyPrefix string) (*GoRedisStore, error) {
	return &GoRedisStore{
		client: client,
		prefix: keyPrefix,
	}, nil
}

func (r *GoRedisStore) GetWithTime(key string) (int64, time.Time, error) {
	key = r.prefix + key

	pipe := r.client.Pipeline()
	timeCmd := pipe.Time()
	getKeyCmd := pipe.Get(key)
	_, err := pipe.Exec()

	now, err := timeCmd.Result()
	if err != nil {
		return 0, now, err
	}

	v, err := getKeyCmd.Int64()
	if err == redis.Nil {
		return -1, now, nil
	} else if err != nil {
		return 0, now, err
	}

	return v, now, nil
}

// SetIfNotExistsWithTTL sets the value of key only if it is not
// already set in the store it returns whether a new value was set.
// If a new value was set, the ttl in the key is also set, though this
// operation is not performed atomically.
func (r *GoRedisStore) SetIfNotExistsWithTTL(key string, value int64, ttl time.Duration) (bool, error) {
	key = r.prefix + key

	updated, err := r.client.SetNX(key, value, 0).Result()
	if err != nil {
		return false, err
	}

	ttlSeconds := time.Duration(ttl.Seconds())

	// An `EXPIRE 0` will delete the key immediately, so make sure that we set
	// expiry for a minimum of one second out so that our results stay in the
	// store.
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	err = r.client.Expire(key, ttlSeconds*time.Second).Err()
	return updated, err
}

// match return true,not return false,atomically
func (r *GoRedisStore) CompareAndSwapWithTTL(key string, old, new int64, ttl time.Duration) (bool, error) {
	key = r.prefix + key

	ttlSeconds := int(ttl.Seconds())

	// An `EXPIRE 0` will delete the key immediately, so make sure that we set
	// expiry for a minimum of one second out so that our results stay in the
	// store.
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	// result will be 0 or 1
	result, err := r.client.Eval(redisCASScript, []string{key}, old, new, ttlSeconds).Result()

	var swapped bool
	if s, ok := result.(int64); ok {
		swapped = s == 1
	} // if not ok, zero value of swapped is false.

	if err != nil {
		if strings.Contains(err.Error(), redisCASMissingKey) {
			return false, nil
		}
		return false, err
	}

	return swapped, nil
}
