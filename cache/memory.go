package cache

import (
	"container/list"
	"sync"
	"time"
)

// MemoryLRU is an in-memory cache that supports eviction based on LRU and TTL.
type MemoryLRU struct {
	mu       sync.Mutex
	items    map[CacheKey]*list.Element
	rankList *list.List

	maxSizeBytes  int
	usedSizeBytes int
	ttl           time.Duration
}

// NewMemoryLRU initialises the LRU cache with the given max-size and ttl.
func NewMemoryLRU(maxSizeBytes int, ttl time.Duration) *MemoryLRU {
	return &MemoryLRU{
		items:    make(map[CacheKey]*list.Element),
		rankList: list.New(),

		maxSizeBytes:  maxSizeBytes,
		usedSizeBytes: 0,
		ttl:           ttl,
	}
}

// Get retrieves the item from the cache and updates it's access time (TTL).
// It also performes passive TTL-eviction and promotes the accessed item to the
// front of the rank.
func (c *MemoryLRU) Get(key CacheKey) *Entry {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	if !ok {
		return nil
	}

	et, ok := item.Value.(*Entry)
	if !ok {
		return nil
	}

	if c.expired(et) {
		c.usedSizeBytes -= et.SizeInBytes()
		c.rankList.Remove(item)
		delete(c.items, key)
		return nil
	}

	et.AccessedAt = time.Now().UTC()
	c.rankList.MoveToFront(item)

	return et
}

// Set inserts/updates an entry into the cache. If an entry is already found,
// rather than updating it's content it is deleted and inserted.
// In case of insufficient space in the cache, first sweep is performed, and
// then entry is inserted.
func (c *MemoryLRU) Set(key CacheKey, et *Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.items[key]; ok {
		if oldRes, ok := item.Value.(*Entry); ok {
			c.usedSizeBytes -= oldRes.SizeInBytes()
		}
		c.rankList.Remove(item)
	}

	etSize := et.SizeInBytes()
	if (c.usedSizeBytes + etSize) >= c.maxSizeBytes {
		c.sweep(etSize)
	}

	c.usedSizeBytes += etSize
	newItem := c.rankList.PushFront(et)
	c.items[key] = newItem
}

// expired returns true if the entry has expired as per TTL.
// Since, we store TTL at two levels, global and entry level,
// we consider the minimum of both to check for expiry.
func (c *MemoryLRU) expired(et *Entry) bool {
	effectiveTTL := et.TTL
	if c.ttl > 0 {
		effectiveTTL = min(c.ttl, et.TTL)
	}

	if time.Now().UTC().After(et.AccessedAt.Add(effectiveTTL)) {
		return true
	}

	return false
}

// sweep deletes the least recently used item (from the back)
// till the new response fits within the cache.
func (c *MemoryLRU) sweep(bytesToFree int) {
	for c.rankList.Len() > 0 {
		lruItem := c.rankList.Back()

		lruEntry, ok := lruItem.Value.(*Entry)
		if !ok {
			return
		}

		if (c.usedSizeBytes + bytesToFree) < c.maxSizeBytes {
			return
		}

		c.rankList.Remove(lruItem)
		delete(c.items, lruEntry.Key)

		c.usedSizeBytes -= lruEntry.SizeInBytes()
	}
}
