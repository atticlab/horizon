package txsub

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/commissions"
	conf "bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/validator"
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
	sub := submitter{
		http:      h,
		coreURL:   url,
		coreDb:    coreDb,
		historyDb: historyDb,
		config:    config,
		Log:       log.WithField("service", "submitter"),
	}
	sub.validators = []validator.ValidatorInterface{
		//validator.NewLimitsValidator(sub.coreDb, sub.historyDb, sub.config),
		validator.NewAdministrativeValidator(sub.historyDb),
		validator.NewAssetsValidator(sub.historyDb, sub.config),
	}
	return &sub
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
	http       *http.Client
	coreURL    string
	coreDb     *core.Q
	historyDb  *history.Q
	config     *conf.Config
	validators []validator.ValidatorInterface
	Log        *log.Entry
}

// Submit sends the provided envelope to stellar-core and parses the response into
// a SubmissionResult
func (sub *submitter) Submit(ctx context.Context, env string) (result SubmissionResult) {
	start := time.Now()
	defer func() { result.Duration = time.Since(start) }()

	// parse tx
	sub.Log.Debug("Parsing tx")
	tx, err := parseTransaction(env)
	if err != nil {
		result.Err = err
		return
	}

	// check constraints for tx
	sub.Log.Debug("Checking tx")
	err = sub.checkTransaction(&tx)
	if err != nil {
		result.Err = err
		return
	}

	cm := commissions.New(sub.coreDb, sub.historyDb)
	err = cm.SetCommissions(&tx)
	if err != nil {
		log.WithField("Error", err).Error("Failed to set commissions")
		result.Err = &problem.ServerError
		return
	}

	updatedEnv, err := writeTransaction(&tx)
	if err != nil {
		result.Err = err
		return
	}

	env = *updatedEnv

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
		result.Err = &results.FailedTransactionError{cresp.Error}
	case StatusPending, StatusDuplicate:
		//noop.  A nil Err indicates success
	default:
		result.Err = errors.Errorf("Unrecognized stellar-core status response: %s", cresp.Status)
	}

	return
}

// checkAccountTypes Parse tx and check account types
func (sub *submitter) checkTransaction(tx *xdr.TransactionEnvelope) error {
	for _, v := range sub.validators {
		result, err := v.CheckTransaction(tx)
		// failed to validate
		if err != nil {
			sub.Log.WithStack(err).WithError(err).Error("Failed to validate tx")
			return &problem.ServerError
		}

		// invalid tx
		if result != nil {
			return result
		}

		additional := make([]results.AdditionalErrorInfo, len(tx.Tx.Operations))
		operationResults := make([]xdr.OperationResult, len(tx.Tx.Operations))
		allowTx := true
		for i, op := range tx.Tx.Operations {
			operationResults[i], additional[i], err = v.CheckOperation(tx.Tx.SourceAccount, &op)
			// failed to validate
			if err != nil {
				sub.Log.WithStack(err).WithError(err).Error("Failed to validate op")
				return &problem.ServerError
			}

			sub.Log.WithField("opType", op.Body.Type).WithField("opResult", operationResults[i].MustTr().Type).Debug("Checking if operation is successful")
			isSuccess, err := results.IsSuccessful(operationResults[i])
			if err != nil {
				sub.Log.WithField("opType", op.Body.Type).WithError(err).Debug("Failed to check if operation IsSuccessful")
				return &problem.ServerError
			}
			if !isSuccess {
				allowTx = false
			}
		}
		if !allowTx {
			restricted, err := results.NewRestrictedTransactionErrorOp(xdr.TransactionResultCodeTxFailed, operationResults, additional)
			if err != nil {
				sub.Log.WithStack(err).WithError(err).Error("Failed to create NewRestrictedTransactionErrorOp")
				return &problem.ServerError
			}
			return restricted
		}

	}
	return nil
}

func parseTransaction(envelope string) (tx xdr.TransactionEnvelope, err error) {
	err = xdr.SafeUnmarshalBase64(envelope, &tx)
	if err != nil {
		err = &results.MalformedTransactionError{envelope}
	}
	return tx, err
}

func writeTransaction(tx *xdr.TransactionEnvelope) (*string, error) {
	res, err := xdr.MarshalBase64(tx)
	if err != nil {
		log.WithField("Erorr", err).Error("Failed to marshal tx")
		err = &results.MalformedTransactionError{}
		return nil, err
	}
	return &res, nil
}
