package nomadclient

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLRUCacheGetMiss(t *testing.T) {
	cache := newLRUCache[string, int](2, time.Minute)

	_, ok := cache.Get("missing")
	assert.False(t, ok)
}

func TestLRUCacheSetAndGet(t *testing.T) {
	cache := newLRUCache[string, int](2, time.Minute)

	cache.Set("a", 1)

	got, ok := cache.Get("a")
	require.True(t, ok)
	assert.Equal(t, 1, got)
}

func TestLRUCacheOverwriteExistingKey(t *testing.T) {
	cache := newLRUCache[string, int](2, time.Minute)

	cache.Set("a", 1)
	cache.Set("a", 2)

	got, ok := cache.Get("a")
	require.True(t, ok)
	assert.Equal(t, 2, got)
}

func TestLRUCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache := newLRUCache[string, int](2, time.Minute)

	cache.Set("a", 1)
	cache.Set("b", 2)

	// Touch "a" so "b" becomes the least-recently-used entry.
	cache.Get("a")

	cache.Set("c", 3)

	_, ok := cache.Get("b")
	assert.False(t, ok, "\"b\" should have been evicted")

	_, ok = cache.Get("a")
	assert.True(t, ok, "\"a\" should not have been evicted")

	_, ok = cache.Get("c")
	assert.True(t, ok)
}

func TestLRUCacheExpiresEntries(t *testing.T) {
	cache := newLRUCache[string, int](2, time.Minute)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }

	cache.Set("a", 1)

	now = now.Add(time.Minute + time.Second)

	_, ok := cache.Get("a")
	assert.False(t, ok, "entry should have expired")
}

func TestLRUCacheDoesNotExpireEarly(t *testing.T) {
	cache := newLRUCache[string, int](2, time.Minute)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }

	cache.Set("a", 1)

	now = now.Add(30 * time.Second)

	got, ok := cache.Get("a")
	require.True(t, ok, "entry should not have expired yet")
	assert.Equal(t, 1, got)
}

func TestLRUCacheSetResetsTTL(t *testing.T) {
	cache := newLRUCache[string, int](2, time.Minute)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }

	cache.Set("a", 1)

	now = now.Add(45 * time.Second)
	cache.Set("a", 2)

	now = now.Add(45 * time.Second)

	got, ok := cache.Get("a")
	require.True(t, ok, "re-Set should have extended the TTL")
	assert.Equal(t, 2, got)
}
