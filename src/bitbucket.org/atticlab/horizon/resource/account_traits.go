package resource

import (
	"fmt"

	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/httpx"
	"bitbucket.org/atticlab/horizon/render/hal"

	"golang.org/x/net/context"
)

// Populate fills out the resource's fields
func (at *AccountTraits) Populate(
	ctx context.Context,
    address string,
	hat history.AccountTraits,
) (err error) {
	// Populate traits
    at.BlockIncomingPayments = hat.BlockIncomingPayments
    at.BlockOutcomingPayments = hat.BlockOutcomingPayments
	// Construct links
	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	accountLink := fmt.Sprintf("/accounts/%s", address)
	self := fmt.Sprintf("/accounts/%s/traits", address)
	at.Links.Self = lb.Link(self)
	at.Links.Account = lb.Link(accountLink)

	return
}
