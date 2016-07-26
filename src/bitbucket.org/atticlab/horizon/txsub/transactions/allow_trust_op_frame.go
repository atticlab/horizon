package transactions

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/txsub/results"
	"bitbucket.org/atticlab/horizon/txsub/transactions/validators"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/config"
)

type AllowTrustOpFrame struct {
	OperationFrame
	operation xdr.AllowTrustOp
}

func NewAllowTrustOpFrame(opFrame OperationFrame) *AllowTrustOpFrame {
	return &AllowTrustOpFrame{
		OperationFrame: opFrame,
		operation:      opFrame.Op.Body.MustAllowTrustOp(),
	}
}

func (frame *AllowTrustOpFrame) DoCheckValid(historyQ history.QInterface, coreQ core.QInterface, config *config.Config) (bool, error) {
	isValid, err := frame.isAssetValid(historyQ)
	if err != nil {
		return false, err
	}

	if !isValid {
		frame.getInnerResult().Code = xdr.AllowTrustResultCodeAllowTrustMalformed
		frame.Result.Info = results.AdditionalErrorInfoError(ASSET_NOT_ALLOWED)
		return false, nil
	}
	frame.getInnerResult().Code = xdr.AllowTrustResultCodeAllowTrustSuccess
	return true, nil
}

func (frame *AllowTrustOpFrame) isAssetValid(historyQ history.QInterface) (bool, error) {
	xdrAsset := xdr.Asset{
		Type: frame.operation.Asset.Type,
	}
	issuer := frame.ParentTx.Tx.SourceAccount
	if frame.Op.SourceAccount != nil {
		issuer = *frame.Op.SourceAccount
	}
	switch frame.operation.Asset.Type {
	case xdr.AssetTypeAssetTypeCreditAlphanum4:
		if frame.operation.Asset.AssetCode4 == nil {
			return false, nil
		}
		xdrAsset.AlphaNum4 = &xdr.AssetAlphaNum4{
			AssetCode: *frame.operation.Asset.AssetCode4,
			Issuer:    issuer,
		}
	case xdr.AssetTypeAssetTypeCreditAlphanum12:
		if frame.operation.Asset.AssetCode12 == nil {
			return false, nil
		}
		xdrAsset.AlphaNum12 = &xdr.AssetAlphaNum12{
			AssetCode: *frame.operation.Asset.AssetCode12,
			Issuer:    issuer,
		}
	default:
		return false, nil
	}
	return validators.NewAssetsValidator(historyQ).IsAssetValid(xdrAsset)
}

func (frame *AllowTrustOpFrame) getInnerResult() *xdr.AllowTrustResult {
	if frame.Result.Result.Tr.AllowTrustResult == nil {
		frame.Result.Result.Tr.AllowTrustResult = &xdr.AllowTrustResult{}
	}
	return frame.Result.Result.Tr.AllowTrustResult
}
