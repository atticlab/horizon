package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"database/sql"
	"github.com/go-errors/errors"
)

var ASSET_NOT_ALLOWED = errors.New("asset not allowed")

type OperationInterface interface {
	DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error)
}

type OperationFrame struct {
	Op            *xdr.Operation
	ParentTx      *xdr.TransactionEnvelope
	Result        *results.OperationResult
	innerOp       OperationInterface
	SourceAccount *core.Account
	log           *log.Entry
}

func NewOperationFrame(op *xdr.Operation, tx *xdr.TransactionEnvelope) OperationFrame {
	return OperationFrame{
		Op:       op,
		ParentTx: tx,
		Result:   &results.OperationResult{},
		log:      log.WithField("service", op.Body.Type.String()),
		SourceAccount: new(core.Account),
	}
}

func (opFrame *OperationFrame) GetInnerOp() (OperationInterface, error) {
	if opFrame.innerOp != nil {
		return opFrame.innerOp, nil
	}
	var innerOp OperationInterface
	switch opFrame.Op.Body.Type {
	case xdr.OperationTypeCreateAccount:
		innerOp = NewCreateAccountOpFrame(*opFrame)
	case xdr.OperationTypePayment:
		innerOp = NewPaymentOpFrame(*opFrame)
	case xdr.OperationTypePathPayment:
		innerOp = NewPathPaymentOpFrame(*opFrame)
	case xdr.OperationTypeManageOffer:
		innerOp = NewManageOfferOpFrame(*opFrame)
	case xdr.OperationTypeCreatePassiveOffer:
		innerOp = NewCreatePassiveOfferOpFrame(*opFrame)
	case xdr.OperationTypeSetOptions:
		innerOp = NewSetOptionsOpFrame(*opFrame)
	case xdr.OperationTypeChangeTrust:
		innerOp = NewChangeTrustOpFrame(*opFrame)
	case xdr.OperationTypeAllowTrust:
		innerOp = NewAllowTrustOpFrame(*opFrame)
	case xdr.OperationTypeAccountMerge:
		innerOp = NewAccountMergeOpFrame(*opFrame)
	case xdr.OperationTypeInflation:
		innerOp = NewInflationOpFrame(*opFrame)
	case xdr.OperationTypeManageData:
		innerOp = NewManageDataOpFrame(*opFrame)
	case xdr.OperationTypeAdministrative:
		innerOp = NewAdministrativeOpFrame(*opFrame)
	default:
		return nil, errors.New("unknown operation")
	}
	opFrame.innerOp = innerOp
	return opFrame.innerOp, nil
}

func (op *OperationFrame) GetResult() results.OperationResult {
	return *op.Result
}

func (opFrame *OperationFrame) CheckValid(historyQ history.QInterface, coreQ core.QInterface, conf *config.Config) (bool, error) {
	sourceAddress := opFrame.ParentTx.Tx.SourceAccount.Address()
	if opFrame.Op.SourceAccount != nil {
		sourceAddress = opFrame.Op.SourceAccount.Address()
	}

	// check if source account exists
	err := coreQ.AccountByAddress(opFrame.SourceAccount, sourceAddress)
	if err != nil {
		if err == sql.ErrNoRows {
			opFrame.Result.Result = xdr.OperationResult{
				Code: xdr.OperationResultCodeOpNoAccount,
			}
			return false, nil
		}
		return false, err
	}

	opFrame.log.WithField("sourceAccount", opFrame.SourceAccount.Accountid).Debug("Loaded source account")
	// prepare result for op Result
	opFrame.Result.Result = xdr.OperationResult{
		Code: xdr.OperationResultCodeOpInner,
		Tr: &xdr.OperationResultTr{
			Type: opFrame.Op.Body.Type,
		},
	}

	innerOp, err := opFrame.GetInnerOp()
	if err != nil {
		return false, err
	}

	// validate
	return innerOp.DoCheckValid(historyQ, coreQ, conf)
}
