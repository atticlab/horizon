package txsub

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"bitbucket.org/atticlab/go-smart-base/xdr"
	conf "bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"github.com/go-errors/errors"
	"golang.org/x/net/context"
)

const (
	StatusError     = "ERROR"
	StatusPending   = "PENDING"
	StatusDuplicate = "DUPLICATE"
)

// NewDefaultSubmitter returns a new, simple Submitter implementation
// that submits directly to the stellar-core at `url` using the http client
// `h`.
func NewDefaultSubmitter(
	h *http.Client,
	url string,
	coreDb *core.Q,
	historyDb *history.Q,
	config *conf.Config,
) Submitter {
	return &submitter{
		http:      h,
		coreURL:   url,
		coreDb:    coreDb,
		historyDb: historyDb,
		config:    config,
	}
}

// coreSubmissionResponse is the json response from stellar-core's tx endpoint
type coreSubmissionResponse struct {
	Exception string `json:"exception"`
	Error     string `json:"error"`
	Status    string `json:"status"`
}

// submitter is the default implementation for the Submitter interface.  It
// submits directly to the configured stellar-core instance using the
// configured http client.
type submitter struct {
	http      *http.Client
	coreURL   string
	coreDb    *core.Q
	historyDb *history.Q
	config    *conf.Config
}

// Submit sends the provided envelope to stellar-core and parses the response into
// a SubmissionResult
func (sub *submitter) Submit(ctx context.Context, env string) (result SubmissionResult) {
	start := time.Now()
	defer func() { result.Duration = time.Since(start) }()

	// check constraints for tx
	err := sub.checkTransaction(env)
	if err != nil {
		result.Err = err
		return
	}

	// construct the request
	u, err := url.Parse(sub.coreURL)
	if err != nil {
		result.Err = errors.Wrap(err, 1)
		return
	}

	u.Path = "/tx"
	q := u.Query()
	q.Add("blob", env)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		result.Err = errors.Wrap(err, 1)
		return
	}

	// perform the submission
	resp, err := sub.http.Do(req)
	if err != nil {
		result.Err = errors.Wrap(err, 1)
		return
	}
	defer resp.Body.Close()

	// parse response
	var cresp coreSubmissionResponse
	err = json.NewDecoder(resp.Body).Decode(&cresp)
	if err != nil {
		result.Err = errors.Wrap(err, 1)
		return
	}

	// interpet response
	if cresp.Exception != "" {
		result.Err = errors.Errorf("stellar-core exception: %s", cresp.Exception)
		return
	}

	switch cresp.Status {
	case StatusError:
		result.Err = &FailedTransactionError{cresp.Error}
	case StatusPending, StatusDuplicate:
		//noop.  A nil Err indicates success
	default:
		result.Err = errors.Errorf("Unrecognized stellar-core status response: %s", cresp.Status)
	}

	return
}

// checkAccountTypes Parse tx and check account types
func (sub *submitter) checkTransaction(envelope string) error {

	tx, err := parseTransaction(envelope)
	if err != nil {
		return err
	}
	additional := make([]string, len(tx.Tx.Operations))
	var xdrResult xdr.TransactionResult
	operationResults := make([]xdr.OperationResult, len(tx.Tx.Operations))
	allowTx := true
	for i := 0; i < len(tx.Tx.Operations); i++ {
		op := tx.Tx.Operations[i]
		opRes, err := sub.checkOperation(op, tx)
		operationResults[i] = opRes
		if err != nil {
			additional[i] = err.Error()
			allowTx = false
		}
	}
	if !allowTx {
		xdrResult.Result, _ = xdr.NewTransactionResultResult(xdr.TransactionResultCodeTxFailed, operationResults)
		resEnv, err := xdr.MarshalBase64(xdrResult)
		if err != nil {
			return err
		}

		return &RestrictedTransactionError{FailedTransactionError{resEnv}, additional}
	}
	return nil
}

// checkAccountTypes Parse tx and check account types
func (sub *submitter) checkOperation(op xdr.Operation, tx xdr.TransactionEnvelope) (opResult xdr.OperationResult, err error) {
	switch op.Body.Type {
	case xdr.OperationTypePayment:
		payment := op.Body.MustPaymentOp()
		destination := payment.Destination.Address()
		var source string
		if len(op.SourceAccount.Address()) > 0 {
			source = op.SourceAccount.Address()
		} else {
			source = tx.Tx.SourceAccount.Address()
		}

		var sourceAcc core.Account
		err = sub.coreDb.AccountByAddress(&sourceAcc, source)
		if err == sql.ErrNoRows {
			opResult = paymentOpResult(xdr.PaymentResultCodePaymentNotAuthorized)
			err = ErrNoAccount
			return
		}
		if err != nil {
			return
		}

		var destinationAcc core.Account
		err = sub.coreDb.AccountByAddress(&destinationAcc, destination)
		if err == sql.ErrNoRows {
			opResult = paymentOpResult(xdr.PaymentResultCodePaymentNoDestination)
			err = ErrNoAccount
			destinationAcc.Accountid = destination
			destinationAcc.AccountType = 0
			return
		}
		if err != nil {
			log.WithStack(err).
				WithField("err", err.Error()).
				Error("destAccError")
			opResult = paymentOpResult(xdr.PaymentResultCodePaymentNotAuthorized)

			return
		}

		// 1. Check account types
		err = VerifyAccountTypesForPayment(sourceAcc, destinationAcc)
		if err != nil {
			log.WithStack(err).
				WithField("err", err.Error()).
				Error("VerifyAccountTypesForPaymentError")
			opResult = paymentOpResult(xdr.PaymentResultCodePaymentNotAuthorized)
			return
		}

		// 2. Check restrictions for accounts
		err = sub.VerifyRestrictions(source, destination)
		if err != nil {
			log.WithStack(err).
				WithField("err", err.Error()).
				Error("VerifyRestrictionsError")
			opResult = paymentOpResult(xdr.PaymentResultCodePaymentNotAuthorized)
			return
		}

		// 3. Check restrictions for sender
		err = sub.VerifyLimitsForSender(sourceAcc, destinationAcc, payment)
		if err != nil {
			log.WithStack(err).
				WithField("err", err.Error()).
				Error("VerifyLimitsForSenderError")
			opResult = paymentOpResult(xdr.PaymentResultCodePaymentSrcNotAuthorized)
			return
		}

		// 4. Check restrictions for receiver
		err = sub.VerifyLimitsForReceiver(sourceAcc, destinationAcc, payment)
		if err != nil {
			log.WithStack(err).
				WithField("err", err.Error()).
				Error("VerifyLimitsForReceiverError")

			opResult = paymentOpResult(xdr.PaymentResultCodePaymentNotAuthorized)
			return
		}
		opResult = paymentOpResult(xdr.PaymentResultCodePaymentSuccess)
	default:
		opResult, err = getSuccessResult(op.Body.Type)
	}
	return
}

func getSuccessResult(opType xdr.OperationType) (opResult xdr.OperationResult, err error) {
	var res interface{}
	switch opType {
	case xdr.OperationTypeCreateAccount:
		res = xdr.CreateAccountResult{xdr.CreateAccountResultCodeCreateAccountSuccess}
	case xdr.OperationTypePathPayment:
		res = xdr.PaymentResult{xdr.PaymentResultCodePaymentSuccess}
	case xdr.OperationTypeManageOffer:
		res = xdr.PaymentResult{xdr.PaymentResultCodePaymentSuccess}
	case xdr.OperationTypeCreatePassiveOffer:
		res = xdr.PaymentResult{xdr.PaymentResultCodePaymentSuccess}
	case xdr.OperationTypeSetOptions:
		res = xdr.SetOptionsResult{xdr.SetOptionsResultCodeSetOptionsSuccess}
	case xdr.OperationTypeChangeTrust:
		res = xdr.ChangeTrustResult{xdr.ChangeTrustResultCodeChangeTrustSuccess}
	case xdr.OperationTypeAllowTrust:
		res = xdr.AllowTrustResult{xdr.AllowTrustResultCodeAllowTrustSuccess}
	case xdr.OperationTypeAccountMerge:
		res = xdr.PaymentResult{xdr.PaymentResultCodePaymentSuccess}
	case xdr.OperationTypeInflation:
		res = xdr.PaymentResult{xdr.PaymentResultCodePaymentSuccess}
	case xdr.OperationTypeManageData:
		res = xdr.ManageDataResult{xdr.ManageDataResultCodeManageDataSuccess}
	default:
		err = &MalformedTransactionError{"unknown_operation"}
		return
	}
	opR, _ := xdr.NewOperationResultTr(opType, res)
	opResult, err = xdr.NewOperationResult(xdr.OperationResultCodeOpInner, opR)
	return
}

func paymentOpResult(code xdr.PaymentResultCode) xdr.OperationResult {
	pr := xdr.PaymentResult{code}
	opR, _ := xdr.NewOperationResultTr(xdr.OperationTypePayment, pr)
	opResult, _ := xdr.NewOperationResult(xdr.OperationResultCodeOpInner, opR)
	return opResult
}

func parseTransaction(envelope string) (tx xdr.TransactionEnvelope, err error) {
	err = xdr.SafeUnmarshalBase64(envelope, &tx)
	if err != nil {
		err = &MalformedTransactionError{envelope}
	}
	return tx, err
}
