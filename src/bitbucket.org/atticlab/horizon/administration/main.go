//Package administration provides services for user management
package administration

import (
	conf "bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/history"
)

// AccountManager facilitates methods for accounts management
type AccountManager interface {
	SetTraits(string, map[string]string) error
	SetLimits(history.AccountLimits) error
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

func GetAdminActionSignatureBase(bodyString string, timeCreated string) string {
	return "{method: 'post', body: '" + bodyString + "', timestamp: '" + timeCreated + "'}"
}
