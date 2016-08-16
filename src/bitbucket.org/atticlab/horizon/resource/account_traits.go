package resource

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/httpx"
	"bitbucket.org/atticlab/horizon/render/hal"
	"fmt"
	"golang.org/x/net/context"
)

// AccountTraits shows if account's incoming, outgoing payments are blocked
type AccountTraits struct {
	Links struct {
		Self    hal.Link `json:"self"`
		Account hal.Link `json:"account"`
	} `json:"_links"`
	PT        string `json:"paging_token"`
	AccountID string `json:"account_id"`
	BlockIn   bool   `json:"block_incoming_payments"`
	BlockOut  bool   `json:"block_outcoming_payments"`
}

func (at *AccountTraits) Populate(ctx context.Context, hat history.AccountTraits) (err error) {
	at.AccountID = hat.AccountAddress
	at.PT = hat.PagingToken()
	at.BlockIn = hat.BlockIncomingPayments
	at.BlockOut = hat.BlockOutcomingPayments
	lb := hal.LinkBuilder{httpx.BaseURL(ctx)}
	at.Links.Account = lb.Link(fmt.Sprintf("/accounts/%s", hat.AccountAddress))
	at.Links.Self = lb.Link(fmt.Sprintf("/accounts/%s/traits", hat.AccountAddress))
	return
}

func (at AccountTraits) PagingToken() string {
	return at.PT
}
