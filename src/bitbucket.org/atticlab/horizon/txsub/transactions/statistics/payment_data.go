package statistics

import (
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
)

type PaymentDirection string

const (
	PaymentDirectionOutgoing PaymentDirection = "outgoing"
	PaymentDirectionIncoming PaymentDirection = "incoming"
)

func (d *PaymentDirection) IsIncoming() bool {
	return *d == PaymentDirectionIncoming
}

type OperationData struct {
	Source *core.Account
	Index  int
	TxHash string
}

func NewOperationData(source *core.Account, index int, txHash string) OperationData {
	return OperationData{
		Source: source,
		Index:  index,
		TxHash: txHash,
	}
}

type PaymentData struct {
	OperationData
	Destination *core.Account
	Amount      int64
	Asset       history.Asset
}

func NewPaymentData(destination *core.Account, opAsset history.Asset, opAmount int64, opData OperationData) PaymentData {
	return PaymentData{
		OperationData: opData,
		Destination:   destination,
		Amount:      opAmount,
		Asset:       opAsset,
	}
}

func (p *PaymentData) GetAccount(direction PaymentDirection) *core.Account {
	if direction == PaymentDirectionOutgoing {
		return p.Source
	}
	return p.Destination
}

func (p *PaymentData) GetCounterparty(direction PaymentDirection) *core.Account {
	if direction == PaymentDirectionIncoming {
		return p.Source
	}
	return p.Destination
}
