//Package administration provides services for user management
package administration

import (
    sql "database/sql"
    "fmt"
    "strconv"
	"bitbucket.org/atticlab/horizon/db2/history"
)

func (m *accountManager) SetTraits(address string, traits map[string]string) error {
    // 1. Check if account exitsts
    var acc history.Account
    err := m.historyDb.AccountByAddress(&acc, address)
    if err == sql.ErrNoRows {
        return AccountNotFoundError{Address: address}
    } else if err != nil {
        return err
    }
    
    // 2. Try get traits for account
    var accTraits history.AccountTraits
    err = m.historyDb.GetAccountTraits(&accTraits, acc.ID)
    if err != nil {
        return err
    }
    
    // 3. Validate and set traits
    errors := NewInvalidFieldsError()
    
    if val, ok := traits["block_incoming_payments"]; ok {
        blockIncomingPayments, parseErr := strconv.ParseBool(val)
        if parseErr == nil {
            accTraits.BlockIncomingPayments = blockIncomingPayments
        } else {
            errors.Errors["block_incoming_payments"] = fmt.Errorf("Boolean value expected, instead got %s", val)
        }
    }
    
    if val, ok := traits["block_outcoming_payments"]; ok {
        blockOutcomingPayments, parseErr := strconv.ParseBool(val)
        if parseErr == nil {
            accTraits.BlockOutcomingPayments = blockOutcomingPayments
        } else {
            errors.Errors["block_outcoming_payments"] = fmt.Errorf("Boolean value expected, instead got %s", val)
        }
    }
    
    if len(errors.Errors) > 0 {
        return errors
    }
    
    // 4. Persist changes
    err = m.historyDb.UpdateAccountTraits(accTraits)
    
    _ = m.historyDb.CreateAuditLogEntry(
        "TODO: add invocer address",
        address,
        "Change account traits",
        getSetTraitsMeta(traits), 
    )
    
    return err
}

func getSetTraitsMeta(traits map[string]string) string {
    meta := ""
    for key, value := range traits {
        meta = meta + fmt.Sprintf("%s: %s\n", key, value)
    }
    
    return meta
}