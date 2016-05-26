//Package administration provides services for user management
package administration

import (
	conf "bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/history"
)

// AccountManager facilitates methods for accounts management
type AccountManager interface {
	
}

// NewAccountManager returns AccountManager implementation
func NewAccountManager(
	historyDb *history.Q,
	config *conf.Config,
) AccountManager {
	return &accountManager{
		historyDb: historyDb,
		config:    config,
	}
}

// accountManager is the default implementation for the AccountManager interface.
type accountManager struct {
	historyDb *history.Q
	config    *conf.Config
}
