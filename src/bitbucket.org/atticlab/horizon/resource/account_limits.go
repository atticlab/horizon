package resource

import (
	"fmt"

	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/httpx"
	"bitbucket.org/atticlab/horizon/render/hal"

	"golang.org/x/net/context"
)

// Populate fills out the resource's fields
func (al *AccountLimits) Populate(
	ctx context.Context,
	address string,
	limits []history.AccountLimits,
) (err error) {
	// Populate limits
	al.Account = address
	al.Limits = make([]AccountLimitsEntry, len(limits))
	for i, limit := range limits {
		al.Limits[i].Populate(limit)
	}

	// Construct links
	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	accountLink := fmt.Sprintf("/accounts/%s", address)
	self := fmt.Sprintf("/accounts/%s/limits", address)
	al.Links.Self = lb.Link(self)
	al.Links.Account = lb.Link(accountLink)

	return
}

// Populate fills out the resource's fields
func (ale *AccountLimitsEntry) Populate(entry history.AccountLimits) {
	ale.AssetCode = entry.AssetCode
	ale.MaxOperationOut = entry.MaxOperationOut
	ale.DailyMaxOut = entry.DailyMaxOut
	ale.MonthlyMaxOut = entry.MonthlyMaxOut
	ale.MaxOperationIn = entry.MaxOperationIn
	ale.DailyMaxIn = entry.DailyMaxIn
	ale.MonthlyMaxIn = entry.MonthlyMaxIn
}
