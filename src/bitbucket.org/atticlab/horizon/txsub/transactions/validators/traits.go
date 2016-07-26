package validators

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"database/sql"
	"fmt"
)

type TraitsValidatorInterface interface {
	CheckTraits(source string, destination string) (*results.RestrictedForAccountError, error)
	CheckTraitsForAccount(account string, isSource bool) (*results.RestrictedForAccountError, error)
}

type TraitsValidator struct {
	historyQ history.QInterface
}

func NewTraitsValidator(historyQ history.QInterface) *TraitsValidator {
	return &TraitsValidator{
		historyQ: historyQ,
	}
}

// VerifyRestrictions checks traits of the involved accounts
func (v *TraitsValidator) CheckTraits(source string, destination string) (*results.RestrictedForAccountError, error) {
	restriction, err := v.CheckTraitsForAccount(source, true)
	if restriction == nil && err == nil {
		restriction, err = v.CheckTraitsForAccount(destination, false)
	}
	return restriction, err
}

func (v *TraitsValidator) CheckTraitsForAccount(account string, isSource bool) (*results.RestrictedForAccountError, error) {
	// Get account traits
	var accountTraits history.AccountTraits
	err := v.historyQ.GetAccountTraitsByAddress(&accountTraits, account)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Check restrictions
	if isSource && accountTraits.BlockOutcomingPayments {
		return &results.RestrictedForAccountError{
			Reason: fmt.Sprintf("Outcoming payments for account (%s) are restricted by administrator.", account),
		}, nil
	}

	if !isSource && accountTraits.BlockIncomingPayments {
		return &results.RestrictedForAccountError{
			Reason: fmt.Sprintf("Incoming payments for account (%s) are restricted by administrator.", account),
		}, nil
	}

	return nil, nil
}
