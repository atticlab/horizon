package commissions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"errors"
	"bitbucket.org/atticlab/go-smart-base/amount"
	"math/big"
)

const (
	FLAT_FEE xdr.Int64 = 3 * amount.One
	PERCENT_FEE xdr.Int64 = 2 * amount.One
)

func SetCommissions(env *xdr.TransactionEnvelope) (err error) {
	if env == nil {
		return errors.New("SetCommissions: tx must not be nil")
	}
	env.OperationFees = make([]xdr.OperationFee, len(env.Tx.Operations))
	for i, op := range env.Tx.Operations {
		env.OperationFees[i] = CountCommission(env.Tx.SourceAccount, op)
	}
	return
}

func CountCommission(txSource xdr.AccountId, op xdr.Operation) xdr.OperationFee {
	opSource := txSource
	if op.SourceAccount != nil {
		opSource = *op.SourceAccount
	}
	switch op.Body.Type {
	case xdr.OperationTypePayment:
		payment := op.Body.MustPaymentOp()
		return countCommission(opSource, payment.Destination, payment.Amount, payment.Asset)
	case xdr.OperationTypePathPayment:
		payment := op.Body.MustPathPaymentOp()
		return countCommission(opSource, payment.Destination, payment.DestAmount, payment.DestAsset)
	default:
		return xdr.OperationFee{
			Type: xdr.OperationFeeTypeOpFeeNone,
		}
	}
}

func countCommission(source, dest xdr.AccountId, amount xdr.Int64, asset xdr.Asset) xdr.OperationFee {
	percentFee := PERCENT_FEE
	flatFee := FLAT_FEE
	return xdr.OperationFee {
		Type: xdr.OperationFeeTypeOpFeeCharged,
		Fee: &xdr.OperationFeeFee{
			Asset: asset,
			AmountToCharge: countPercentFee(amount, PERCENT_FEE) + FLAT_FEE,
			PercentFee: &percentFee,
			FlatFee: &flatFee,
		},
	}
}

func countPercentFee(paymentAmountI, percentI xdr.Int64) xdr.Int64 {
	// (amount/100) * percent
	paymentAmount := big.NewRat(int64(paymentAmountI), 100)
	percentR := big.NewRat(int64(percentI), amount.One)
	var result big.Rat
	result.Mul(paymentAmount, percentR)
	// if denom is > 1, then 0
	if result.Denom().Int64() > 1 {
		return xdr.Int64(0)
	}
	return xdr.Int64(result.Num().Int64())
}
