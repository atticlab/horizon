package resource

import (
	"fmt"
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/httpx"
	"bitbucket.org/atticlab/horizon/render/hal"
	"golang.org/x/net/context"
)

func (this *Ledger) Populate(ctx context.Context, row history.Ledger) {
	this.ID = row.LedgerHash
	this.PT = row.PagingToken()
	this.Hash = row.LedgerHash
	this.PrevHash = row.PreviousLedgerHash.String
	this.Sequence = row.Sequence
	this.TransactionCount = row.TransactionCount
	this.OperationCount = row.OperationCount
	this.ClosedAt = row.ClosedAt
	this.TotalCoins = amount.String(xdr.Int64(row.TotalCoins))
	this.FeePool = amount.String(xdr.Int64(row.FeePool))
	this.BaseFee = row.BaseFee
	this.BaseReserve = amount.String(xdr.Int64(row.BaseReserve))
	this.MaxTxSetSize = row.MaxTxSetSize

	self := fmt.Sprintf("/ledgers/%d", row.Sequence)
	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	this.Links.Self = lb.Link(self)
	this.Links.Transactions = lb.PagedLink(self, "transactions")
	this.Links.Operations = lb.PagedLink(self, "operations")
	this.Links.Payments = lb.PagedLink(self, "payments")
	this.Links.Effects = lb.PagedLink(self, "effects")

	return
}

func (this Ledger) PagingToken() string {
	return this.PT
}
