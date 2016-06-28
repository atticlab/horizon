package commissions

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"errors"
	"math"
	"math/big"
)

type CommissionsManager struct {
	CoreQ    *core.Q
	HistoryQ *history.Q
}

func New(coreQ *core.Q, histQ *history.Q) CommissionsManager {
	return CommissionsManager{
		CoreQ: coreQ,
		HistoryQ: histQ,
	}
}

func (cm *CommissionsManager) SetCommissions(env *xdr.TransactionEnvelope) (err error) {
	if env == nil {
		return errors.New("SetCommissions: tx must not be nil")
	}
	env.OperationFees = make([]xdr.OperationFee, len(env.Tx.Operations))
	for i, op := range env.Tx.Operations {
		commission, err := cm.CountCommission(env.Tx.SourceAccount, op)
		if err != nil {
			log.WithStack(err).WithError(err).Error("Failed to count commission")
			return errors.New("failed to count commission")
		}
		env.OperationFees[i] = *commission
	}
	return
}

func (cm *CommissionsManager) CountCommission(txSource xdr.AccountId, op xdr.Operation) (*xdr.OperationFee, error) {
	opSource := txSource
	if op.SourceAccount != nil {
		opSource = *op.SourceAccount
	}
	switch op.Body.Type {
	case xdr.OperationTypePayment:
		payment := op.Body.MustPaymentOp()
		return cm.countCommission(opSource, payment.Destination, payment.Amount, payment.Asset)
	case xdr.OperationTypePathPayment:
		payment := op.Body.MustPathPaymentOp()
		return cm.countCommission(opSource, payment.Destination, payment.DestAmount, payment.DestAsset)
	default:
		return &xdr.OperationFee{
			Type: xdr.OperationFeeTypeOpFeeNone,
		}, nil
	}
}

func (cm *CommissionsManager) getCommission(sourceId, destinationId xdr.AccountId, amount xdr.Int64, asset xdr.Asset) (*history.Commission, error) {
	var sourceAccount, destAccount core.Account
	err := cm.CoreQ.AccountByAddress(&sourceAccount, sourceId.Address())
	if err != nil {
		return nil, err
	}

	err = cm.CoreQ.AccountByAddress(&destAccount, destinationId.Address())
	if err != nil {
		return nil, err
	}

	baseAsset := assets.ToBaseAsset(asset)
	keys := history.CreateCommissionKeys(sourceId.Address(), destinationId.Address(), int32(sourceAccount.AccountType), int32(destAccount.AccountType), baseAsset)
	commissions, err := cm.HistoryQ.GetHighestWeightCommission(keys)
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to GetHighestWeightCommission")
		return nil, err
	}

	histCommission := new(history.Commission)
	fee := xdr.Int64(math.MaxInt64)
	for _, comm := range commissions {
		newFee := cm.countPercentFee(amount, xdr.Int64(comm.PercentFee)) + xdr.Int64(comm.FlatFee)
		if newFee <= fee {
			*histCommission = comm
		}
	}
	return histCommission, nil
}

func (cm *CommissionsManager) countCommission(source, destination xdr.AccountId, amount xdr.Int64, asset xdr.Asset) (*xdr.OperationFee, error) {
	commission, err := cm.getCommission(source, destination, amount, asset)
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to getCommission")
		return nil, err
	}
	if commission == nil {
		return &xdr.OperationFee{
			Type: xdr.OperationFeeTypeOpFeeNone,
		}, nil
	}
	percent := xdr.Int64(commission.PercentFee)
	flatFee := xdr.Int64(commission.FlatFee)
	return &xdr.OperationFee{
		Type: xdr.OperationFeeTypeOpFeeCharged,
		Fee: &xdr.OperationFeeFee{
			Asset:          asset,
			AmountToCharge: cm.countPercentFee(amount, percent) + xdr.Int64(flatFee),
			PercentFee:     &percent,
			FlatFee:        &flatFee,
		},
	}, nil
}

func (cm *CommissionsManager) countPercentFee(paymentAmountI, percentI xdr.Int64) xdr.Int64 {
	zero := xdr.Int64(0)
	if percentI == zero {
		return zero
	}
	// (amount/100) * percent
	paymentAmount := big.NewRat(int64(paymentAmountI), 100)
	percentR := big.NewRat(int64(percentI), amount.One)
	var result big.Rat
	result.Mul(paymentAmount, percentR)
	return xdr.Int64(result.Num().Int64() / result.Denom().Int64())
}
