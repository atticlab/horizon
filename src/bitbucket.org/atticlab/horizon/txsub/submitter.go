package txsub

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/accounttypes"
	"bitbucket.org/atticlab/horizon/cache"
	"bitbucket.org/atticlab/horizon/commissions"
	conf "bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/transactions"
	"bitbucket.org/atticlab/horizon/txsub/transactions/statistics"
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
	sharedCache *cache.SharedCache,
	otherHorizonUrl string,
	) Submitter {
	return createSubmitter(h, url, coreDb, historyDb, config, sharedCache, otherHorizonUrl)
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
	http     *http.Client
	coreURL  string
	horizonUrl string
	coreQ    *core.Q
	historyQ *history.Q
	config   *conf.Config
	Log      *log.Entry

	defaultTxValidator TransactionValidatorInterface
	commissionManager  *commissions.CommissionsManager
}

func createSubmitter(h *http.Client, url string, coreDb *core.Q, historyDb *history.Q, config *conf.Config, sharedCache *cache.SharedCache, otherHorizonUrl string) *submitter {
	return &submitter{
		http:               h,
		coreURL:            url,
		horizonUrl:			otherHorizonUrl,
		coreQ:              coreDb,
		historyQ:           historyDb,
		config:             config,
		commissionManager:  commissions.New(sharedCache, historyDb),
		defaultTxValidator: NewTransactionValidator(transactions.NewManager(coreDb, historyDb, statistics.NewManager(historyDb, accounttype.GetAll(), config), config, sharedCache)),
		Log:                log.WithField("service", "submitter"),
	}
}
 
// Submit sends the provided envelope to stellar-core and parses the response into
// a SubmissionResult
func (sub *submitter) Submit(ctx context.Context, env *transactions.EnvelopeInfo) (result SubmissionResult) {
	start := time.Now()
	defer func() { result.Duration = time.Since(start) }()

	// check constraints for tx
	sub.Log.Debug("Checking tx")
	err := sub.defaultTxValidator.CheckTransaction(env)
	if err != nil {
		result.Err = err
		return
	}

	// now we should decide what to do with the transaction - 
	// sign it and send to the core, or forward to another core

	if !sub.config.IsSigner(){
		result = sub.Forward(ctx, env)
		return
	}
	// ok, looks like we have a secret key and are approved to sign
	sub.Log.Debug("Setting commission")
	err = sub.commissionManager.SetCommissions(env.Tx)
	if err != nil {
		log.WithField("Error", err).Error("Failed to set commissions")
		result.Err = &problem.ServerError
		return
	}

	tx1 := env.Tx.Tx.Sign(sub.config.ApproveSecret)
	// err := SignByHorizon(env.Tx, sub.config.ApproveSecret)

	updatedEnv, err := writeTransaction(env.Tx)
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
	q.Add("blob", *updatedEnv)
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

func writeTransaction(tx *xdr.TransactionEnvelope) (*string, error) {
	res, err := xdr.MarshalBase64(tx)
	if err != nil {
		log.WithField("Erorr", err).Error("Failed to marshal tx")
		err = &results.MalformedTransactionError{}
		return nil, err
	}
	return &res, nil
}

// Forward sends the provided envelope to another horizon and parses the response into
// a SubmissionResult
func (sub *submitter) Forward(ctx context.Context, env *transactions.EnvelopeInfo) (result SubmissionResult) {

}

func SignByHorizon(env *xdr.TransactionEnvelope, sk string) (err error) {

}