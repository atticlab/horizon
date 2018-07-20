package transactions

import (
	"github.com/atticlab/horizon/cache"
	"github.com/atticlab/horizon/config"
	"github.com/atticlab/horizon/db2/core"
	"github.com/atticlab/horizon/db2/history"
	"github.com/atticlab/horizon/txsub/transactions/statistics"
)

type Manager struct {
	*cache.SharedCache
	CoreQ        core.QInterface
	HistoryQ     history.QInterface
	StatsManager statistics.ManagerInterface
	Config       *config.Config
}

func NewManager(core core.QInterface, history history.QInterface, statsManager statistics.ManagerInterface, config *config.Config, sharedCache *cache.SharedCache) *Manager {
	return &Manager{
		CoreQ:        core,
		HistoryQ:     history,
		StatsManager: statsManager,
		Config:       config,
		SharedCache:  sharedCache,
	}
}
