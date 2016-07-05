package horizon

import (
	"bitbucket.org/atticlab/go-smart-base/hash"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/administration"
	"bitbucket.org/atticlab/horizon/audit"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"time"
)

var (
	Unauthorized = problem.P{
		Type:   "unauthorized",
		Title:  "Unauthorized request",
		Status: http.StatusUnauthorized,
		Detail: "Request should be signed.",
	}
)

/* Action that provides method for admin signature validation and audit_log info storage.
Each action that uses AdminAction must begin it with StartAdminAction and FinishAdminAction.
FinishAdminAction must be called even if action fails.
*/
type AdminAction struct {
	Action
	Info            audit.AdminActionInfo
	Signature       string
	TimeCreated     time.Time
	RawTimeCreated  string
	signersProvider core.SignersProvider
}

func (action *AdminAction) SignersProvider() core.SignersProvider {
	if action.signersProvider == nil {
		appSignersProvider := action.App.SignersProvider()
		if appSignersProvider == nil {
			action.signersProvider = action.CoreQ()
		} else {
			action.signersProvider = appSignersProvider
		}
	}
	return action.signersProvider
}

func (action *AdminAction) prepare() {
	if action.Err != nil {
		return
	}
	action.Signature = action.R.Header.Get("X-AuthSignature")

	rawPubKey := action.R.Header.Get("X-AuthPublicKey")
	action.Info.ActorPublicKey, _ = keypair.Parse(rawPubKey)

	action.RawTimeCreated = action.R.Header.Get("X-AuthTimestamp")
	action.TimeCreated = getTimestamp(action.RawTimeCreated)
}

func getTimestamp(rawTimestamp string) time.Time {
	i, err := strconv.ParseInt(rawTimestamp, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(i, 0)
}

// checks admin signature and starts transaction in History DB.
func (action *AdminAction) StartAdminAction() {

	log := log.WithField("service", "admin_action")

	if action.Err != nil {
		return
	}

	action.prepare()

	if !action.isSignatureValid() {
		log.Debug("Signature is invalid")
		if action.Err == nil {
			action.Err = &Unauthorized
		}
		return
	}

	log.Debug("Starting admin transaction")
	err := action.HistoryQ().Begin()
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to begin admin transaction")
		action.Err = &problem.ServerError
		return
	}
	log.Debug("Admin transaction started")
	return
}

func (action *AdminAction) getContentHash() [32]byte {
	// Read the content
	var bodyBytes []byte
	if action.R.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(action.R.Body)
		// Restore the io.ReadCloser to its original state
		action.R.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// Use the content
	bodyString := string(bodyBytes)

	signatureBase := administration.GetAdminActionSignatureBase(bodyString, action.RawTimeCreated)
	hashBase := hash.Hash([]byte(signatureBase))

	actual := hex.EncodeToString(hashBase[:])
	log.WithField("signatureBase", signatureBase).Info("signatureBase")
	log.WithField("actual", actual).WithField("hashBase", hashBase).WithField("publicKey", action.Info.ActorPublicKey.Address()).Info("signatureBase")

	return hashBase
}

func (action *AdminAction) isSignatureValid() bool {
	l := log.WithField("service", "admin_sig_checker")

	if action.Signature == "" || action.Info.ActorPublicKey == nil {
		l.Debug("Signature or auth public key is not set")
		return false
	}

	validFor := time.Duration(action.App.config.AdminSignatureValid) * time.Second
	sigExpiresAt := action.TimeCreated.Add(validFor)
	now := time.Now()
	if sigExpiresAt.Before(now) {
		l.WithFields(log.F{
			"adminSigValidFor": validFor,
			"expired":          sigExpiresAt,
			"now":              now,
		}).Debug("Signature expired")
		return false
	}

	var decoratedSign xdr.DecoratedSignature
	err := xdr.SafeUnmarshalBase64(action.Signature, &decoratedSign)
	if err != nil {
		l.WithField("err", err).Info("Failed to unmarshal signature")
		return false
	}
	l.WithField("sigBytes", decoratedSign.Signature).WithField("signature", action.Signature).Info("signatureBase")

	contentHash := action.getContentHash()
	err = action.Info.ActorPublicKey.Verify(contentHash[:], decoratedSign.Signature)
	if err != nil {
		l.Info("Failed to verify - Signature is invalid")
		return false
	}

	l.WithField("SignersProvider type:", reflect.TypeOf(action.SignersProvider())).Debug("Checking trype")
	// checking if signer was admin
	var coreSigners []core.Signer
	err = action.SignersProvider().SignersByAddress(&coreSigners, action.App.config.BankMasterKey)
	if err != nil {
		l.WithStack(err).WithError(err).Error("Failed to get signers")
		action.Err = &problem.ServerError
		return false
	}

	for _, signer := range coreSigners {
		if signer.Publickey == action.Info.ActorPublicKey.Address() && signer.SignerType == uint32(xdr.SignerTypeSignerAdmin) {
			return true
		}
	}
	l.Debug("Signer is not admin")
	return false
}

// Finalizes work with History DB transaction.
// If not all fields for audit log are set returns ServerError
func (action *AdminAction) FinishAdminAction() {
	if action.Err != nil {
		if action.HistoryQ().IsStarted() {
			log.Debug("Admin action failed - rolling back")
			action.Rollback()
		}
		return
	}

	if !action.HistoryQ().IsStarted() {
		log.Panic("When using admin action - StartAdminAction must be called")
		return
	}
	if !action.Info.IsValid() {
		errorInfo := "Admin action info is invalid"
		log.Error(errorInfo)
		p := problem.ServerError
		p.Detail = errorInfo
		action.Err = &p
		action.Rollback()
		return
	}

	err := action.CreateAuditLog(&action.Info)
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to create audit logs")
		action.Err = &problem.ServerError
		action.Rollback()
		return
	}
	err = action.HistoryQ().Commit()
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to commit admin action transaction")
		action.Err = &problem.ServerError
		return
	}
}

func (action *AdminAction) Rollback() {
	err := action.HistoryQ().Rollback()
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to rollback histroy db admin transaction.")
		action.Err = &problem.ServerError
		return
	}
}

func (action *AdminAction) CreateAuditLog(info *audit.AdminActionInfo) error {
	return action.HistoryQ().CreateAuditLogEntry(info.ToHistory())
}
