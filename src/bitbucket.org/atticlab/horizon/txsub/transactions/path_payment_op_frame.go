package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/transactions/validators"
	"database/sql"
)

type PathPaymentOpFrame struct {
	OperationFrame
	pathPayment   xdr.PathPaymentOp
	sendAsset     history.Asset
	destAsset     history.Asset
	destAccount   core.Account
	destTrustline core.Trustline

	accountTypeValidator      validators.AccountTypeValidatorInterface
	assetsValidator           validators.AssetsValidatorInterface
	traitsValidator           validators.TraitsValidatorInterface
	defaultOutLimitsValidator validators.OutgoingLimitsValidatorInterface
	defaultInLimitsValidator  validators.IncomingLimitsValidatorInterface
}

func NewPathPaymentOpFrame(opFrame OperationFrame) *PathPaymentOpFrame {
	return &PathPaymentOpFrame{
		OperationFrame: opFrame,
		pathPayment:    opFrame.Op.Body.MustPathPaymentOp(),
	}
}

func (p *PathPaymentOpFrame) GetTraitsValidator(historyQ history.QInterface) validators.TraitsValidatorInterface {
	if p.traitsValidator == nil {
		p.traitsValidator = validators.NewTraitsValidator(historyQ)
	}
	return p.traitsValidator
}

func (p *PathPaymentOpFrame) GetAccountTypeValidator() validators.AccountTypeValidatorInterface {
	if p.accountTypeValidator == nil {
		p.accountTypeValidator = validators.NewAccountTypeValidator()
	}
	return p.accountTypeValidator
}

func (p *PathPaymentOpFrame) GetOutgoingLimitsValidator(account, counterparty *core.Account, opAmount int64,
	opAsset history.Asset, historyQ history.QInterface, anonUserRestr config.AnonymousUserRestrictions) validators.OutgoingLimitsValidatorInterface {
	if p.defaultOutLimitsValidator != nil {
		return p.defaultOutLimitsValidator
	}
	return validators.NewOutgoingLimitsValidator(account, counterparty, opAmount, opAsset, historyQ, anonUserRestr)
}

func (p *PathPaymentOpFrame) GetIncomingLimitsValidator(account, counterparty *core.Account,
	accountTrustLine core.Trustline, opAmount int64, opAsset history.Asset, historyQ history.QInterface,
	anonUserRestr config.AnonymousUserRestrictions) validators.IncomingLimitsValidatorInterface {
	if p.defaultInLimitsValidator != nil {
		return p.defaultInLimitsValidator
	}
	return validators.NewIncomingLimitsValidator(account, counterparty, accountTrustLine, opAmount, opAsset, historyQ, anonUserRestr)
}

func (p *PathPaymentOpFrame) GetAssetsValidator(historyQ history.QInterface) validators.AssetsValidatorInterface {
	if p.assetsValidator == nil {
		p.log.Debug("Creating new assets validator")
		p.assetsValidator = validators.NewAssetsValidator(historyQ)
	}
	return p.assetsValidator
}

func (p *PathPaymentOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, conf *config.Config) (bool, error) {
	// check if all assets are valid
	isAssetsValid, err := p.isAssetsValid(historyQ)
	if err != nil {
		p.log.Error("Failed to validate assets")
		return false, err
	}

	if !isAssetsValid {
		p.getInnerResult().Code = xdr.PathPaymentResultCodePathPaymentMalformed
		p.Result.Info = results.AdditionalErrorInfoError(ASSET_NOT_ALLOWED)
		return false, nil
	}

	// check if destination exists or asset is anonymous
	destExists, err := p.tryLoadDestinationAccount(coreQ)
	if err != nil {
		return false, err
	}

	if !destExists && !p.destAsset.IsAnonymous {
		p.getInnerResult().Code = xdr.PathPaymentResultCodePathPaymentNoDestination
		return false, nil
	}

	// check if destination trust line exists or (dest account does not exist and asset is anonymous)
	if destExists {
		err = coreQ.TrustlineByAddressAndAsset(&p.destTrustline, p.pathPayment.Destination.Address(), p.destAsset.Code, p.destAsset.Issuer)
		if err != nil {
			if err != sql.ErrNoRows {
				return false, err
			}
			p.getInnerResult().Code = xdr.PathPaymentResultCodePathPaymentNoTrust
			return false, nil
		}
	}

	isLimitsValid, err := p.checkLimits(historyQ, coreQ, conf)
	if err != nil || !isLimitsValid {
		return isLimitsValid, err
	}

	p.getInnerResult().Code = xdr.PathPaymentResultCodePathPaymentSuccess
	return true, nil
}

func (p *PathPaymentOpFrame) tryLoadDestinationAccount(coreQ core.QInterface) (bool, error) {
	err := coreQ.AccountByAddress(&p.destAccount, p.pathPayment.Destination.Address())
	if err == nil {
		return true, nil
	} else if err != sql.ErrNoRows {
		return false, err
	}
	p.destAccount.Accountid = p.pathPayment.Destination.Address()
	p.destAccount.AccountType = xdr.AccountTypeAccountAnonymousUser
	return false, nil
}

func (p *PathPaymentOpFrame) getInnerResult() *xdr.PathPaymentResult {
	if p.Result.Result.Tr.PathPaymentResult == nil {
		p.Result.Result.Tr.PathPaymentResult = &xdr.PathPaymentResult{}
	}
	return p.Result.Result.Tr.PathPaymentResult
}

func (p *PathPaymentOpFrame) isAssetsValid(historyQ history.QInterface) (bool, error) {
	// check if assets are valid
	assetsValidator := p.GetAssetsValidator(historyQ)
	var err error
	sendAsset, err := assetsValidator.GetValidAsset(p.pathPayment.SendAsset)
	if err != nil || sendAsset == nil {
		return false, err
	}

	p.sendAsset = *sendAsset

	destAsset, err := assetsValidator.GetValidAsset(p.pathPayment.DestAsset)
	if err != nil || destAsset == nil {
		return false, err
	}

	p.destAsset = *destAsset

	if p.pathPayment.Path != nil {
		return assetsValidator.IsAssetsValid(p.pathPayment.Path...)
	}

	return true, nil
}

func (p *PathPaymentOpFrame) checkLimits(historyQ history.QInterface, coreQ core.QInterface, conf *config.Config) (bool, error) {

	// 1. Check account types
	p.log.Debug("Validating account types")
	accountTypesRestricted := p.GetAccountTypeValidator().VerifyAccountTypesForPayment(*p.SourceAccount, p.destAccount)
	if accountTypesRestricted != nil {
		p.getInnerResult().Code = xdr.PathPaymentResultCodePathPaymentMalformed
		p.Result.Info = results.AdditionalErrorInfoError(accountTypesRestricted)
		return false, nil
	}

	// 2. Check traits for accounts
	p.log.WithField("sourceAccount", p.SourceAccount.Accountid).WithField("destAccount", p.destAccount.Accountid).Debug("Checking traits")
	accountRestricted, err := p.GetTraitsValidator(historyQ).CheckTraits(p.SourceAccount.Accountid, p.destAccount.Accountid)
	if err != nil {
		return false, err
	}

	if accountRestricted != nil {
		p.getInnerResult().Code = xdr.PathPaymentResultCodePathPaymentMalformed
		p.Result.Info = results.AdditionalErrorInfoError(accountRestricted)
		return false, nil
	}

	// 3. Check restrictions for sender
	outgoingValidator := p.GetOutgoingLimitsValidator(p.SourceAccount,
		&p.destAccount,
		int64(p.pathPayment.SendMax),
		p.sendAsset,
		historyQ,
		conf.AnonymousUserRestrictions)
	outLimitsResult, err := outgoingValidator.VerifyLimits()
	if err != nil {
		return false, err
	}

	if outLimitsResult != nil {
		p.getInnerResult().Code = xdr.PathPaymentResultCodePathPaymentMalformed
		p.Result.Info = results.AdditionalErrorInfoError(outLimitsResult)
		return false, nil
	}

	incomingValidator := p.GetIncomingLimitsValidator(&p.destAccount, p.SourceAccount,
		p.destTrustline, int64(p.pathPayment.DestAmount), p.destAsset, historyQ, conf.AnonymousUserRestrictions)
	inLimitsResult, err := incomingValidator.VerifyLimits()
	if err != nil {
		return false, err
	}

	if inLimitsResult != nil {
		p.getInnerResult().Code = xdr.PathPaymentResultCodePathPaymentMalformed
		p.Result.Info = results.AdditionalErrorInfoError(inLimitsResult)
		return false, nil
	}

	return true, nil
}
