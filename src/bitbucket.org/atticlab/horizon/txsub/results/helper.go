package results

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/log"
)

func IsSuccessful(opResult xdr.OperationResult) (isSuccess bool, err error) {
	if opResult.Code != xdr.OperationResultCodeOpInner {
		return false, nil
	}
	inner := opResult.Tr
	var code int32
	switch inner.Type {
	case xdr.OperationTypeCreateAccount:
		code = int32(inner.MustCreateAccountResult().Code)
	case xdr.OperationTypePayment:
		code = int32(inner.MustPaymentResult().Code)
	case xdr.OperationTypePathPayment:
		code = int32(inner.MustPathPaymentResult().Code)
	case xdr.OperationTypeManageOffer:
		code = int32(inner.MustManageOfferResult().Code)
	case xdr.OperationTypeCreatePassiveOffer:
		code = int32(inner.MustCreatePassiveOfferResult().Code)
	case xdr.OperationTypeSetOptions:
		code = int32(inner.MustSetOptionsResult().Code)
	case xdr.OperationTypeChangeTrust:
		code = int32(inner.MustChangeTrustResult().Code)
	case xdr.OperationTypeAllowTrust:
		code = int32(inner.MustAllowTrustResult().Code)
	case xdr.OperationTypeAccountMerge:
		code = int32(inner.MustAccountMergeResult().Code)
	case xdr.OperationTypeInflation:
		code = int32(inner.MustInflationResult().Code)
	case xdr.OperationTypeManageData:
		code = int32(inner.MustCreateAccountResult().Code)
	case xdr.OperationTypeAdministrative:
		code = int32(inner.MustAdminResult().Code)
	default:
		err = &MalformedTransactionError{"unknown_operation"}
	}
	if err != nil {
		log.Error("Failed to check if operation is successful")
		return false, nil
	}
	return code == 0, nil
}

func GetSuccessResult(opType xdr.OperationType) (opResult xdr.OperationResult, err error) {
	var res interface{}
	switch opType {
	case xdr.OperationTypeCreateAccount:
		res, err = xdr.NewCreateAccountResult(xdr.CreateAccountResultCodeCreateAccountSuccess, nil)
	case xdr.OperationTypePayment:
		res, err = xdr.NewPaymentResult(xdr.PaymentResultCodePaymentSuccess, nil)
	case xdr.OperationTypePathPayment:
		res, err = xdr.NewPathPaymentResult(xdr.PathPaymentResultCodePathPaymentSuccess, xdr.PathPaymentResultSuccess{})
	case xdr.OperationTypeManageOffer:
		res, err = xdr.NewManageOfferResult(xdr.ManageOfferResultCodeManageOfferSuccess, xdr.ManageOfferSuccessResult{})
	case xdr.OperationTypeCreatePassiveOffer:
		res, err = xdr.NewManageOfferResult(xdr.ManageOfferResultCodeManageOfferSuccess, xdr.ManageOfferSuccessResult{})
	case xdr.OperationTypeSetOptions:
		res, err = xdr.NewSetOptionsResult(xdr.SetOptionsResultCodeSetOptionsSuccess, nil)
	case xdr.OperationTypeChangeTrust:
		res, err = xdr.NewChangeTrustResult(xdr.ChangeTrustResultCodeChangeTrustSuccess, nil)
	case xdr.OperationTypeAllowTrust:
		res, err = xdr.NewAllowTrustResult(xdr.AllowTrustResultCodeAllowTrustSuccess, nil)
	case xdr.OperationTypeAccountMerge:
		res, err = xdr.NewAccountMergeResult(xdr.AccountMergeResultCodeAccountMergeSuccess, xdr.Int64(0))
	case xdr.OperationTypeInflation:
		res, err = xdr.NewInflationResult(xdr.InflationResultCodeInflationSuccess, []xdr.InflationPayout{})
	case xdr.OperationTypeManageData:
		res, err = xdr.NewManageDataResult(xdr.ManageDataResultCodeManageDataSuccess, nil)
	case xdr.OperationTypeAdministrative:
		res, err = xdr.NewAdministrativeResult(xdr.AdministrativeResultCodeAdministrativeSuccess, nil)
	default:
		err = &MalformedTransactionError{"unknown_operation"}
	}
	if err != nil {
		log.Error("Failed to get successful result")
		return
	}
	opR, _ := xdr.NewOperationResultTr(opType, res)
	opResult, err = xdr.NewOperationResult(xdr.OperationResultCodeOpInner, opR)
	return
}

func NewPaymentOpResult(code xdr.PaymentResultCode) xdr.OperationResult {
	pr, _ := xdr.NewPaymentResult(code, nil)
	opR, _ := xdr.NewOperationResultTr(xdr.OperationTypePayment, pr)
	opResult, _ := xdr.NewOperationResult(xdr.OperationResultCodeOpInner, opR)
	return opResult
}

func NewAdminOpResult(code xdr.AdministrativeResultCode) xdr.OperationResult {
	pr, _ := xdr.NewAdministrativeResult(code, nil)
	opR, _ := xdr.NewOperationResultTr(xdr.OperationTypeAdministrative, pr)
	opResult, _ := xdr.NewOperationResult(xdr.OperationResultCodeOpInner, opR)
	return opResult
}
