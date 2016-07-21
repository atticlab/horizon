package validator

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/config"
	"bitbucket.org/atticlab/horizon/cache"
	"errors"
	"database/sql"
)

type AssetsValidator struct {
	log       *log.Entry
	historyDb *history.Q
	conf *config.Config

}

func NewAssetsValidator(historyDb *history.Q, conf *config.Config) *AssetsValidator {
	return &AssetsValidator{
		log:       log.WithField("service", "account_creation_validator"),
		historyDb: historyDb,
		conf: conf,
	}
}

func (v *AssetsValidator) CheckTransaction(tx *xdr.TransactionEnvelope) (*results.RestrictedTransactionError, error) {
	return nil, nil
}

// checkAccountTypes Parse tx and check account types
func (v *AssetsValidator) CheckOperation(source xdr.AccountId, op *xdr.Operation) (opResult xdr.OperationResult, additionalInfo results.AdditionalErrorInfo, err error) {
	isValid, err := v.checkAssets(source, op)
	if err != nil {
		return
	}
	if !isValid {
		opResult, err = results.GetMalformedOpResult(op.Body.Type)
		if err != nil {
			return
		}
		return opResult, results.AdditionalErrorInfoStrError("asset_not_suppoted"), nil
	}

	var destination xdr.AccountId
	var asset xdr.Asset
	switch op.Body.Type {
	case xdr.OperationTypePayment:
		payment := op.Body.MustPaymentOp()
		destination = payment.Destination
		asset = payment.Asset
	case xdr.OperationTypePathPayment:
		payment := op.Body.MustPathPaymentOp()
		destination = payment.Destination
		asset = payment.DestAsset
	default:
		opResult, err = results.GetSuccessResult(op.Body.Type)
		return
	}
	storedAsset, err := cache.NewHistoryAsset(v.historyDb).Get(asset)
	if err != nil {
		return
	}

	if storedAsset == nil {
		opResult, err = results.GetMalformedOpResult(op.Body.Type)
		if err != nil {
			return
		}
		return opResult, results.AdditionalErrorInfoStrError("asset_not_suppoted"), nil
	}

	if storedAsset.IsAnonymous {
		opResult, err = results.GetSuccessResult(op.Body.Type)
		return
	}

	var storedAccount history.Account
	err = v.historyDb.AccountByAddress(&storedAccount, destination.Address())
	if err != nil {
		if err == sql.ErrNoRows {
			switch op.Body.Type {
			case xdr.OperationTypePayment:
				opResult = results.NewPaymentOpResult(xdr.PaymentResultCodePaymentNoDestination)
			case xdr.OperationTypePathPayment:
				opResult = results.NewPathPaymentOpResult(xdr.PathPaymentResultCodePathPaymentNoDestination)
			default:
				return opResult, nil, errors.New("unknown_operation")
			}
			return opResult, nil, nil
		}
		return
	}
	opResult, err = results.GetSuccessResult(op.Body.Type)
	return
}


func (v *AssetsValidator) checkAssets(source xdr.AccountId, op *xdr.Operation) (bool, error) {
	switch op.Body.Type {
	case xdr.OperationTypeCreateAccount:
		// ok
	case xdr.OperationTypePayment:
		asset := op.Body.MustPaymentOp().Asset
		return v.isAssetValid(asset)
	case xdr.OperationTypePathPayment:
		payment := op.Body.MustPathPaymentOp()
		isValid, err := v.isAssetsValid(payment.SendAsset, payment.DestAsset)
		if err != nil || !isValid {
			return isValid, err
		}

		if payment.Path != nil {
			return v.isAssetsValid(payment.Path...)
		}
		return true, nil
	case xdr.OperationTypeManageOffer:
		offer := op.Body.MustManageOfferOp()
		return v.isAssetsValid(offer.Buying, offer.Selling)
	case xdr.OperationTypeCreatePassiveOffer:
		offer := op.Body.MustCreatePassiveOfferOp()
		return v.isAssetsValid(offer.Buying, offer.Selling)
	case xdr.OperationTypeSetOptions:
		// ok
	case xdr.OperationTypeChangeTrust:
		return v.isAssetsValid(op.Body.MustChangeTrustOp().Line)
	case xdr.OperationTypeAllowTrust:
		allowTrust := op.Body.MustAllowTrustOp()
		xdrAsset := xdr.Asset{
			Type: allowTrust.Asset.Type,
		}
		issuer := source
		if op.SourceAccount != nil {
			issuer = *op.SourceAccount
		}
		switch allowTrust.Asset.Type {
		case xdr.AssetTypeAssetTypeCreditAlphanum4:
			if allowTrust.Asset.AssetCode4 == nil {
				return false, nil
			}
			xdrAsset.AlphaNum4 = &xdr.AssetAlphaNum4{
				AssetCode: *allowTrust.Asset.AssetCode4,
				Issuer: issuer,
			}
		case xdr.AssetTypeAssetTypeCreditAlphanum12:
			if allowTrust.Asset.AssetCode12 == nil {
				return false, nil
			}
			xdrAsset.AlphaNum12 = &xdr.AssetAlphaNum12{
				AssetCode: *allowTrust.Asset.AssetCode12,
				Issuer: issuer,
			}
		default:
			return false, nil
		}
		return v.isAssetsValid(xdrAsset)
	case xdr.OperationTypeAccountMerge:
		// ok
	case xdr.OperationTypeInflation:
		// ok
	case xdr.OperationTypeManageData:
		// ok
	case xdr.OperationTypeAdministrative:
		// ok
	default:
		return false, errors.New("Unknown operation type")

	}
	return true, nil
}

func (v *AssetsValidator) isAssetValid(asset xdr.Asset) (bool, error) {
	assetsProvider := cache.NewHistoryAsset(v.historyDb)
	storedAsset, err := assetsProvider.Get(asset)
	return storedAsset != nil, err
}

func (v *AssetsValidator) isAssetsValid(assets... xdr.Asset) (bool, error) {
	for _, asset := range assets {
		isValid, err := v.isAssetValid(asset)
		if err != nil {
			return false, err
		}

		if !isValid {
			return isValid, err
		}
	}
	return true, nil
}
