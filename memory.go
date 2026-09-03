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

	maxSize  int
	usedSize int
	ttl      time.Duration
}

func NewMemoryLRU(maxSize int, ttl time.Duration) *MemoryLRU {
	return &MemoryLRU{
		items:    make(map[CacheKey]*list.Element),
		rankList: list.New(),
		maxSize:  maxSize,
		usedSize: 0,
		ttl:      ttl,
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
		c.rankList.Remove(item)
	}

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
