//Package administration provides services for user management
package administration

import (
	sql "database/sql"
	"fmt"

	"bitbucket.org/atticlab/horizon/db2/history"
)

// SetLimits sets limits for an account and asset
func (m *accountManager) SetLimits(limit history.AccountLimits) error {
	// 1. Check if account exitsts
	var acc history.Account
	address := limit.Account
	err := m.historyDb.AccountByAddress(&acc, address)
	if err == sql.ErrNoRows {
		return AccountNotFoundError{Address: address}
	} else if err != nil {
		return err
	}

	// 2. Try get limits for account
	var isNewEntry bool
	var accLimits history.AccountLimits
	err = m.historyDb.GetAccountLimits(&accLimits, limit.Account, limit.AssetCode)
	if err == sql.ErrNoRows {
		isNewEntry = true
	} else if err != nil {
		return err
	}

	// 3. Validate and set limits
	accLimits = limit

	// 4. Persist changes
	if isNewEntry {
		err = m.historyDb.CreateAccountLimits(accLimits)
	} else {
		err = m.historyDb.UpdateAccountLimits(accLimits)
	}

	_ = m.historyDb.CreateAuditLogEntry(
		"TODO: add invocer address",
		address,
		"Change account limit",
		getSetLimitsMeta(limit),
	)

	return err

}

func getSetLimitsMeta(limit history.AccountLimits) string {
	meta := fmt.Sprintf("%+v", limit)
	return meta
}
