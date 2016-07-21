package validator

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	conf "bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"database/sql"
)

type LimitsValidator struct {
	log       *log.Entry
	coreDb    *core.Q
	historyDb *history.Q
	config    *conf.Config
}

func NewLimitsValidator(coreDb *core.Q, historyDb *history.Q, config *conf.Config) *LimitsValidator {
	return &LimitsValidator{
		log:       log.WithField("service", "limits_validator"),
		coreDb:    coreDb,
		historyDb: historyDb,
		config:    config,
	}
}

func (v *LimitsValidator) CheckTransaction(tx *xdr.TransactionEnvelope) (error) {
	return nil
}

// checkAccountTypes Parse tx and check account types
func (v *LimitsValidator) CheckOperation(source xdr.AccountId, op *xdr.Operation) (opResult xdr.OperationResult, additionalInfo results.AdditionalErrorInfo, err error) {
	switch op.Body.Type {
	case xdr.OperationTypePayment:
		payment := op.Body.MustPaymentOp()
		destination := payment.Destination.Address()
		if op.SourceAccount != nil {
			source = *op.SourceAccount
		}

		var sourceAcc core.Account
		err = v.coreDb.AccountByAddress(&sourceAcc, source.Address())
		if err == sql.ErrNoRows {
			opResult = results.NewPaymentOpResult(xdr.PaymentResultCodePaymentNotAuthorized)
			err = results.ErrNoAccount
			return
		}
		if err != nil {
			return
		}

		var destinationAcc core.Account
		err = v.coreDb.AccountByAddress(&destinationAcc, destination)
		if err == sql.ErrNoRows {
			opResult = results.NewPaymentOpResult(xdr.PaymentResultCodePaymentNoDestination)
			err = nil
			destinationAcc.Accountid = destination
			destinationAcc.AccountType = 0
			return
		}
		if err != nil {
			log.WithStack(err).
				WithField("err", err.Error()).
				Error("destAccError")
			opResult = results.NewPaymentOpResult(xdr.PaymentResultCodePaymentNotAuthorized)

			return
		}

		// 1. Check account types
		err = v.verifyAccountTypesForPayment(sourceAcc, destinationAcc)
		if err != nil {
			log.WithStack(err).
				WithField("err", err.Error()).
				Error("VerifyAccountTypesForPaymentError")
			opResult = results.NewPaymentOpResult(xdr.PaymentResultCodePaymentNotAuthorized)
			return
		}

		// 2. Check restrictions for accounts
		err = v.verifyRestrictions(source.Address(), destination)
		if err != nil {
			log.WithStack(err).
				WithField("err", err.Error()).
				Error("VerifyRestrictionsError")
			opResult = results.NewPaymentOpResult(xdr.PaymentResultCodePaymentNotAuthorized)
			return
		}

		// 3. Check restrictions for sender
		err = v.verifyLimitsForSender(sourceAcc, destinationAcc, payment)
		if err != nil {
			log.WithStack(err).
				WithField("err", err.Error()).
				Error("VerifyLimitsForSenderError")
			opResult = results.NewPaymentOpResult(xdr.PaymentResultCodePaymentSrcNotAuthorized)
			return
		}

		// 4. Check restrictions for receiver
		err = v.verifyLimitsForReceiver(sourceAcc, destinationAcc, payment)
		if err == sql.ErrNoRows {
			opResult = results.NewPaymentOpResult(xdr.PaymentResultCodePaymentNoTrust)
		}
		if err != nil {
			log.WithStack(err).
				WithField("err", err.Error()).
				Error("VerifyLimitsForReceiverError")

			opResult = results.NewPaymentOpResult(xdr.PaymentResultCodePaymentNotAuthorized)
			return
		}
		opResult = results.NewPaymentOpResult(xdr.PaymentResultCodePaymentSuccess)
	default:
		opResult, err = results.GetSuccessResult(op.Body.Type)
	}
	return
}
