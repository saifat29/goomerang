package main

import (
	"container/list"
	"sync"
	"time"
)

type MemoryLRU struct {
	mu       sync.Mutex
	items    map[CacheKey]*list.Element
	rankList *list.List

	maxSizeBytes  int
	usedSizeBytes int
	ttl           time.Duration
}

func NewMemoryLRU(maxSizeBytes int, ttl time.Duration) *MemoryLRU {
	return &MemoryLRU{
		items:    make(map[CacheKey]*list.Element),
		rankList: list.New(),

		maxSizeBytes:  maxSizeBytes,
		usedSizeBytes: 0,
		ttl:           ttl,
	}
}

func (c *MemoryLRU) Get(key CacheKey) *Response {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	if !ok {
		return nil
	}

	res, ok := item.Value.(*Response)
	if !ok {
		return nil
	}

	if c.expired(res) {
		c.usedSizeBytes -= res.SizeInBytes()
		c.rankList.Remove(item)
		delete(c.items, key)
		return nil
	}

	res.AccessedAt = time.Now().UTC()
	c.rankList.MoveToFront(item)

	return res
}

func (c *MemoryLRU) Set(key CacheKey, res *Response) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.items[key]; ok {
		if oldRes, ok := item.Value.(*Response); ok {
			c.usedSizeBytes -= oldRes.SizeInBytes()
		}
		c.rankList.Remove(item)
	}

	resSize := res.SizeInBytes()
	if (c.usedSizeBytes + resSize) >= c.maxSizeBytes {
		c.sweep(resSize)
	}

	c.usedSizeBytes += resSize
	newItem := c.rankList.PushFront(res)
	c.items[key] = newItem
}

func (c *MemoryLRU) expired(res *Response) bool {
	effectiveTTL := res.TTL
	if c.ttl > 0 {
		effectiveTTL = min(c.ttl, res.TTL)
	}

	if time.Now().UTC().After(res.AccessedAt.Add(effectiveTTL)) {
		return true
	}

	return false
}

// sweep deletes the least recently used item (from the back)
// till the new response fits within the cache.
func (c *MemoryLRU) sweep(bytesToFree int) {
	for c.rankList.Len() > 0 {
		lruItem := c.rankList.Back()

		lruResp, ok := lruItem.Value.(*Response)
		if !ok {
			return
		}

		if (c.usedSizeBytes + bytesToFree) < c.maxSizeBytes {
			return
		}

		c.rankList.Remove(lruItem)
		delete(c.items, lruResp.Key)

		c.usedSizeBytes -= lruResp.SizeInBytes()
	}
}
