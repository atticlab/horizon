package resource

import (
	"fmt"

	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/httpx"
	"bitbucket.org/atticlab/horizon/render/hal"

	"golang.org/x/net/context"
	"bitbucket.org/atticlab/horizon/db2/core"
	"time"
)

// Populate fills out the resource's fields
func (as *AccountStatistics) Populate(
	ctx context.Context,
	statistics []core.AccountStatistics,
	ha history.Account,
) (err error) {
	// Populate statistics
	as.Statistics = make([]AccountStatisticsEntry, len(statistics))
	for i, stat := range statistics {
		as.Statistics[i].Populate(stat)
	}
	// Construct links
	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	accountLink := fmt.Sprintf("/accounts/%s", ha.Address)
	self := fmt.Sprintf("/accounts/%s/statistics", ha.Address)
	as.Links.Self = lb.Link(self)
	as.Links.Account = lb.Link(accountLink)

	return
}

// Populate fills out the resource's fields
func (entry *AccountStatisticsEntry) Populate(stats core.AccountStatistics) {
	// Set asset
	entry.Asset.Code = stats.AssetCode
	entry.Asset.Issuer = stats.AssetIssuer
	entry.Asset.Type, _ = assets.String(xdr.AssetType(stats.AssetType))

	// Set counterparty type
	entry.CounterpartyType, entry.CounterpartyTypeName = PopulateAccountType(xdr.AccountType(stats.Counterparty))

	// Populate income
	entry.Income.Daily = amount.String(xdr.Int64(stats.DailyIn))
	entry.Income.Monthly = amount.String(xdr.Int64(stats.MonthlyIn))
	entry.Income.Annual = amount.String(xdr.Int64(stats.AnnualIn))
	// Populate outcome
	entry.Outcome.Daily = amount.String(xdr.Int64(stats.DailyOut))
	entry.Outcome.Monthly = amount.String(xdr.Int64(stats.MonthlyOut))
	entry.Outcome.Annual = amount.String(xdr.Int64(stats.AnnualOut))
	entry.UpdatedAt = time.Unix(stats.UpdatedAt, 0)
}
