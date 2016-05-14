package resource

import (
	"bitbucket.org/atticlab/horizon/db2/core"
	"golang.org/x/net/context"
)

// Populate fills out the fields of the signer, using one of an account's
// secondary signers.
func (this *Signer) Populate(ctx context.Context, row core.Signer) {
	this.PublicKey = row.Publickey
	this.Weight = row.Weight
	this.SignerType = row.SignerType
}

// PopulateMaster fills out the fields of the signer, using a stellar account to
// provide the data.
func (this *Signer) PopulateMaster(row core.Account) {
	this.PublicKey = row.Accountid
	this.Weight = int32(row.Thresholds[0])
	this.SignerType = uint32(0)
}
