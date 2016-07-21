// Package cache provides various caches used in horizon.
package cache

import (
	"github.com/golang/groupcache/lru"
	"time"
)

type Cache struct {
	cached *lru.Cache
	entryLifeTime *time.Duration
}

func NewCache(maxEntries int, entryLifeTime *time.Duration) *Cache {
	return &Cache{
		cached: lru.New(maxEntries),
		entryLifeTime: entryLifeTime,
	}
}

func (c *Cache) IsEntryAlive(timeAdded time.Time) bool {
	if c.entryLifeTime == nil {
		return true
	}
	return timeAdded.Add(*c.entryLifeTime).After(time.Now())
}
