package cache

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"database/sql"
	"time"
)

var historyAssetCache *Cache

// HistoryAsset provides a cached lookup of asset values from
// xdr.Asset.
type HistoryAsset struct {
	Cache
	db *history.Q
}

// NewHistoryAsset initializes a new instance of `HistoryAsset`
func NewHistoryAsset(db *history.Q) *HistoryAsset {
	if historyAssetCache == nil {
		lifeTime := time.Duration(10)*time.Minute
		historyAssetCache = NewCache(100, &lifeTime)
	}
	return &HistoryAsset{
		Cache: *historyAssetCache,
		db:    db,
	}
}

type historyAssetElem struct {
	Asset     *history.Asset
	timeAdded time.Time
}

// Get looks up the history.Asset for the given xdr.Asset.
func (c *HistoryAsset) Get(asset xdr.Asset) (*history.Asset, error) {
	found, ok := c.cached.Get(asset)

	if ok {
		cacheElem, ok := found.(historyAssetElem)
		if ok {
			if c.IsEntryAlive(cacheElem.timeAdded) {
				return cacheElem.Asset, nil
			}
		}
		c.cached.Remove(asset)
	}

	var result history.Asset
	err := c.db.Asset(&result, asset)
	elem := historyAssetElem{
		Asset:     &result,
		timeAdded: time.Now(),
	}
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
		elem.Asset = nil
	}
	c.cached.Add(asset, elem)
	return elem.Asset, nil
}
