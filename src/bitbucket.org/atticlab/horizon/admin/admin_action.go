package admin

import (
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/audit"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/helpers"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"net/http"
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

/* Provides method for admin signature validation and audit_log info storage.
Each action that uses AdminActionInterface must begin it with StartAction and FinishAction.
FinishAction must be called even if action fails.
*/
type AdminActionInterface interface {
	// checks admin signature and starts transaction in History DB.
	StartAction() *problem.P
	// finalizes admin action. If failed is true - rollbacks tx, else commits. All fields in Audit Info must be set before call!
	FinishAction(failed bool) *problem.P
	GetAuditInfo() *audit.AdminActionInfo
}

type adminAction struct {
	info            audit.AdminActionInfo
	signature       string
	contentHash     [32]byte
	timeCreated     time.Time
	rawTimeCreated  string
	signersProvider core.SignersProvider
	r               *http.Request
	config          *config.Config
	history         *history.Q
}

func NewAdminAction(r *http.Request, history *history.Q, signersProvider core.SignersProvider, config *config.Config) AdminActionInterface {
	action := &adminAction{
		signature:       r.Header.Get("X-AuthSignature"),
		rawTimeCreated:  r.Header.Get("X-AuthTimestamp"),
		config:          config,
		history:         history,
		signersProvider: signersProvider,
	}

	rawPubKey := r.Header.Get("X-AuthPublicKey")
	action.info.ActorPublicKey, _ = keypair.Parse(rawPubKey)

	action.timeCreated = helpers.ParseTimestamp(action.rawTimeCreated)
	action.contentHash = GetContentsHash(r, action.rawTimeCreated)
	return action
}

func (action *adminAction) GetAuditInfo() *audit.AdminActionInfo {
	return &action.info
}

func (action *adminAction) StartAction() *problem.P {

	log := log.WithField("service", "admin_action")

	if isValid, err := action.isSignatureValid(); !isValid || err != nil {
		log.Debug("Signature is invalid")
		if err != nil {
			log.WithStack(err).WithError(err).Error("Failed to validate signature")
			return &problem.ServerError
		}
		return &Unauthorized
	}

	log.Debug("Starting admin transaction")
	err := action.history.Begin()
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to begin admin transaction")
		return &problem.ServerError
	}
	log.Debug("Admin transaction started")
	return nil
}

func (action *adminAction) isSignatureValid() (bool, error) {
	l := log.WithField("service", "admin_sig_checker")

	if action.signature == "" || action.info.ActorPublicKey == nil || action.info.ActorPublicKey.Address() == "" {
		l.Debug("Signature or auth public key is not set")
		return false, nil
	}

	sigExpiresAt := action.timeCreated.Add(action.config.AdminSignatureValid)
	now := time.Now()
	if sigExpiresAt.Before(now) {
		l.WithFields(log.F{
			"adminSigValidFor": action.config.AdminSignatureValid,
			"expired":          sigExpiresAt,
			"now":              now,
		}).Debug("Signature expired")
		return false, nil
	}

	var decoratedSign xdr.DecoratedSignature
	err := xdr.SafeUnmarshalBase64(action.signature, &decoratedSign)
	if err != nil {
		l.WithField("err", err).Info("Failed to unmarshal signature")
		return false, nil
	}
	l.WithField("sigBytes", decoratedSign.Signature).WithField("signature", action.signature).Info("signatureBase")

	err = action.info.ActorPublicKey.Verify(action.contentHash[:], decoratedSign.Signature)
	if err != nil {
		l.Info("Failed to verify - Signature is invalid")
		return false, nil
	}

	// checking if signer was admin
	var coreSigners []core.Signer
	err = action.signersProvider.SignersByAddress(&coreSigners, action.config.BankMasterKey)
	if err != nil {
		l.Error("Failed to get signers")
		return false, err
	}

	for _, signer := range coreSigners {
		if signer.Publickey == action.info.ActorPublicKey.Address() && signer.SignerType == uint32(xdr.SignerTypeSignerAdmin) {
			return true, nil
		}
	}
	l.Debug("Signer is not admin")
	return false, nil
}

func (action *adminAction) FinishAction(failed bool) *problem.P {
	if failed {
		if action.history.IsStarted() {
			log.Debug("Admin action failed - rolling back")
			return action.rollback()
		}
		return nil
	}

	if !action.history.IsStarted() {
		log.Panic("When using admin action - StartAction must be called")
		return nil
	}
	if !action.info.IsValid() {
		errorInfo := "Admin action info is invalid"
		log.Error(errorInfo)
		p := &problem.ServerError
		p.Detail = errorInfo
		err := action.rollback()
		if err != nil {
			return err
		}
		return p
	}

	err := action.createAuditLog(&action.info)
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to create audit logs")
		err := action.rollback()
		if err != nil {
			return err
		}
		return &problem.ServerError
	}
	err = action.history.Commit()
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to commit admin action transaction")
		return &problem.ServerError
	}
	return nil
}

func (action *adminAction) rollback() *problem.P {
	err := action.history.Rollback()
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to rollback histroy db admin transaction.")
		return &problem.ServerError
	}
	return nil
}

func (action *adminAction) createAuditLog(info *audit.AdminActionInfo) error {
	return action.history.CreateAuditLogEntry(info.ToHistory())
}
