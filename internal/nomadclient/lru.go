package nomadclient

import (
	"container/list"
	"sync"
	"time"
)

// lruEntry is the value stored in lruCache's linked list.
type lruEntry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
}

// lruCache is a fixed-capacity, thread-safe cache. Entries are evicted
// least-recently-used first once the cache is over capacity, and are treated
// as absent once their TTL (set per-cache, applied per-entry on write) has
// passed. A zero or negative ttl means entries never expire on their own —
// eviction happens only via LRU capacity overflow.
type lruCache[K comparable, V any] struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	now      func() time.Time
	order    *list.List
	items    map[K]*list.Element
}

// newLRUCache creates a cache holding at most capacity entries, each expiring
// ttl after it was last written. Pass ttl <= 0 for entries that never expire
// on their own, relying solely on LRU capacity eviction.
func newLRUCache[K comparable, V any](capacity int, ttl time.Duration) *lruCache[K, V] {
	return &lruCache[K, V]{
		capacity: capacity,
		ttl:      ttl,
		now:      time.Now,
		order:    list.New(),
		items:    make(map[K]*list.Element),
	}
}

// Get returns the cached value for key, if present and not expired. A hit
// marks the entry as most-recently-used.
func (c *lruCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}

	entry := elem.Value.(*lruEntry[K, V])

	expired := c.ttl > 0 && c.now().After(entry.expiresAt)
	if expired {
		c.order.Remove(elem)
		delete(c.items, key)
		var zero V
		return zero, false
	}

	c.order.MoveToFront(elem)

	return entry.value, true
}

// Set stores value for key, resetting its TTL and marking it
// most-recently-used. If the cache is over capacity afterward, the
// least-recently-used entry is evicted.
func (c *lruCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := c.now().Add(c.ttl)

	elem, ok := c.items[key]
	if ok {
		entry := elem.Value.(*lruEntry[K, V])
		entry.value = value
		entry.expiresAt = expiresAt
		c.order.MoveToFront(elem)
		return
	}

	entry := &lruEntry[K, V]{key: key, value: value, expiresAt: expiresAt}
	newElem := c.order.PushFront(entry)
	c.items[key] = newElem

	overCapacity := c.order.Len() > c.capacity
	if overCapacity {
		oldest := c.order.Back()
		c.order.Remove(oldest)
		oldestEntry := oldest.Value.(*lruEntry[K, V])
		delete(c.items, oldestEntry.key)
	}
}
