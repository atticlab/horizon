package effects

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/httpx"
	"bitbucket.org/atticlab/horizon/render/hal"
	"golang.org/x/net/context"
)

// PagingToken implements `db2.Pageable`
func (this Base) PagingToken() string {
	return this.PT
}

// Populate loads this resource from `row`
func (this *Base) Populate(ctx context.Context, row history.Effect) {
	this.ID = row.ID()
	this.PT = row.PagingToken()
	this.Account = row.Account
	this.populateType(row)

	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	this.Links.Operation = lb.Linkf("/operations/%d", row.HistoryOperationID)
	this.Links.Succeeds = lb.Linkf("/effects?order=desc&cursor=%s", this.PT)
	this.Links.Precedes = lb.Linkf("/effects?order=asc&cursor=%s", this.PT)
}

func (this *Base) populateType(row history.Effect) {
	var ok bool
	this.TypeI = int32(row.Type)
	this.Type, ok = TypeNames[row.Type]

	if !ok {
		this.Type = "unknown"
	}
}
