package resource

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2/history"
	"fmt"
	"golang.org/x/net/context"
)

func (this *HistoryAsset) Populate(ctx context.Context, row history.Asset) {
	this.ID = row.Id
	this.IsAnonymous = row.IsAnonymous
	var err error
	this.Type, err = assets.String(xdr.AssetType(row.Type))
	if err != nil {
		this.Type = "invalid_type"
	}

	this.Code = row.Code
	this.Issuer = row.Issuer
}

func (this HistoryAsset) PagingToken() string {
	return fmt.Sprintf("%d", this.ID)
}
