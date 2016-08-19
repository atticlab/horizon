package session

import (
	"encoding/base64"
	"fmt"
	"time"

	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/admin"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/ingest/participants"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/resource/operations"
	"bitbucket.org/atticlab/horizon/ingest/session/helpers"
	"encoding/json"
	"github.com/spf13/viper"
)

// Run starts an attempt to ingest the range of ledgers specified in this
// session.
func (is *Session) Run() {
	is.Err = is.Ingestion.Start()
	if is.Err != nil {
		return
	}

	defer is.Ingestion.Rollback()

	for is.Cursor.NextLedger() {
		if is.Err != nil {
			return
		}

		is.clearLedger()
		is.ingestLedger()
		is.flush()
	}

	if is.Err != nil {
		is.Ingestion.Rollback()
		return
	}

	is.Err = is.Ingestion.Close()

	// TODO: validate ledger chain

}

func (is *Session) clearLedger() {
	if is.Err != nil {
		return
	}

	if !is.ClearExisting {
		return
	}
	start := time.Now()
	is.Err = is.Ingestion.Clear(is.Cursor.LedgerRange())
	if is.Metrics != nil {
		is.Metrics.ClearLedgerTimer.Update(time.Since(start))
	}
}

func (is *Session) flush() {
	if is.Err != nil {
		return
	}
	is.Err = is.Ingestion.Flush()
}

// ingestLedger ingests the current ledger
func (is *Session) ingestLedger() {
	if is.Err != nil {
		return
	}

	start := time.Now()
	is.Ingestion.Ledger(
		is.Cursor.LedgerID(),
		is.Cursor.Ledger(),
		is.Cursor.SuccessfulTransactionCount(),
		is.Cursor.SuccessfulLedgerOperationCount(),
	)

	// If this is ledger 1, create the root account
	if is.Cursor.LedgerSequence() == 1 {
		//config := app.Config
		masterKey := viper.GetString("bank-master-key")
		//config.BankMasterKey //
		commisionKey := viper.GetString("bank-commission-key") // config.BankCommissionKey //
		is.Ingestion.Account(1, masterKey)
		//keypair.Master(is.Network).Address())
		if masterKey != commisionKey {
			is.Ingestion.Account(2, commisionKey)
		}
	}

	for is.Cursor.NextTx() {
		is.ingestTransaction()
	}

	is.Ingested++
	if is.Metrics != nil {
		is.Metrics.IngestLedgerTimer.Update(time.Since(start))
	}

	return
}

func (is *Session) ingestOperation() {
	if is.Err != nil {
		return
	}

	is.Err = is.Ingestion.Operation(
		is.Cursor.OperationID(),
		is.Cursor.TransactionID(),
		is.Cursor.OperationOrder(),
		is.Cursor.OperationSourceAccount(),
		is.Cursor.OperationType(),
		is.operationDetails(),
	)
	if is.Err != nil {
		return
	}

	switch is.Cursor.Operation().Body.Type {
	case xdr.OperationTypePayment:
		// Update statistics for both accounts
		op := is.Cursor.Operation().Body.MustPaymentOp()
		from := is.Cursor.OperationSourceAccount()
		to := op.Destination
		assetCode, _ := getAssetCode(op.Asset)
		is.ingestPayment(from.Address(), to.Address(), op.Amount, op.Amount, assetCode, assetCode)
	case xdr.OperationTypePathPayment:
		op := is.Cursor.Operation().Body.MustPathPaymentOp()
		from := is.Cursor.OperationSourceAccount()
		to := op.Destination
		result := is.Cursor.OperationResult().MustPathPaymentResult()
		sourceAmount := result.SendAmount()
		destAmount := op.DestAmount
		sourceAsset, _ := getAssetCode(op.SendAsset)
		destAsset, _ := getAssetCode(op.DestAsset)
		is.ingestPayment(from.Address(), to.Address(), sourceAmount, destAmount, sourceAsset, destAsset)
	case xdr.OperationTypeCreateAccount:
		// Import the new account if one was created
		op := is.Cursor.Operation().Body.MustCreateAccountOp()
		is.Err = is.Ingestion.Account(is.Cursor.OperationID(), op.Destination.Address())
	case xdr.OperationTypeAdministrative:
		log := log.WithFields(log.F{
			"tx_hash":      is.Cursor.Transaction().TransactionHash,
			"operation_id": is.Cursor.OperationID(),
		})
		op := is.Cursor.Operation().Body.MustAdminOp()
		var opData map[string]interface{}
		err := json.Unmarshal([]byte(op.OpData), &opData)
		if err != nil {
			// skip silently
			log.WithError(err).Error("Failed to unmarshal json data")
			break
		}
		adminActionProvider := admin.NewAdminActionProvider(&history.Q{is.Ingestion.DB})
		adminAction, err := adminActionProvider.CreateNewParser(opData)
		if err != nil {
			log.WithError(err).Error("Failed to create admin action")
			break
		}
		adminAction.Validate()
		if adminAction.GetError() != nil {
			log.WithError(adminAction.GetError()).Error("Failed to validate admin action")
			break
		}
		adminAction.Apply()
		if adminAction.GetError() != nil {
			log.WithError(adminAction.GetError()).Error("Failed to apply admin action")
			break
		}
	}

	is.ingestOperationParticipants()
	is.ingestEffects()
}

func (is *Session) ingestOperationParticipants() {
	if is.Err != nil {
		return
	}

	// Find the participants
	var p []xdr.AccountId
	p, is.Err = participants.ForOperation(
		&is.Cursor.Transaction().Envelope.Tx,
		is.Cursor.Operation(),
	)
	if is.Err != nil {
		return
	}

	var aids []int64
	aids, is.Err = is.lookupParticipantIDs(p)
	if is.Err != nil {
		return
	}

	is.Err = is.Ingestion.OperationParticipants(is.Cursor.OperationID(), aids)
	if is.Err != nil {
		return
	}
}
func (is *Session) ingestTransaction() {
	if is.Err != nil {
		return
	}

	// skip ingesting failed transactions
	if !is.Cursor.Transaction().IsSuccessful() {
		return
	}

	is.Ingestion.Transaction(
		is.Cursor.TransactionID(),
		is.Cursor.Transaction(),
		is.Cursor.TransactionFee(),
	)

	for is.Cursor.NextOp() {
		is.ingestOperation()
	}

	is.ingestTransactionParticipants()
}

func (is *Session) ingestTransactionParticipants() {
	if is.Err != nil {
		return
	}

	// Find the participants
	var p []xdr.AccountId
	p, is.Err = participants.ForTransaction(
		&is.Cursor.Transaction().Envelope.Tx,
		&is.Cursor.Transaction().ResultMeta,
		&is.Cursor.TransactionFee().Changes,
	)
	if is.Err != nil {
		return
	}

	var aids []int64
	aids, is.Err = is.lookupParticipantIDs(p)
	if is.Err != nil {
		return
	}

	is.Err = is.Ingestion.TransactionParticipants(is.Cursor.TransactionID(), aids)
	if is.Err != nil {
		return
	}

}

func (is *Session) ingestEffects() {
	if is.Err != nil {
		return
	}

	effects := NewEffectIngestion(is.Ingestion, is.accountCache, is.Cursor.OperationID())

	effects.Ingest(is.Cursor)

	is.Err = effects.Finish()
}

func (is *Session) lookupParticipantIDs(aids []xdr.AccountId) (ret []int64, err error) {
	found := map[int64]bool{}

	for _, in := range aids {
		var out int64
		out, err = is.accountCache.Get(in.Address())
		if err != nil {
			return
		}

		// De-duplicate
		if _, ok := found[out]; ok {
			continue
		}

		found[out] = true
		ret = append(ret, out)
	}

	return
}

func getAssetCode(a xdr.Asset) (string, error) {
	var (
		t    string
		code string
		i    string
	)
	err := a.Extract(&t, &code, &i)

	return code, err
}

func (is *Session) feeDetails(xdrFee xdr.OperationFee) map[string]interface{} {
	fee := operations.Fee{}
	fee.Populate(xdrFee)
	return fee.ToMap()
}

// operationDetails returns the details regarding the current operation, suitable
// for ingestion into a history_operation row
func (is *Session) operationDetails() map[string]interface{} {
	details := map[string]interface{}{}
	c := is.Cursor
	source := c.OperationSourceAccount()

	fee := c.Transaction().Envelope.OperationFees[c.OperationOrder()-1]
	details["fee"] = is.feeDetails(fee)

	switch c.OperationType() {
	case xdr.OperationTypeCreateAccount:
		op := c.Operation().Body.MustCreateAccountOp()
		details["funder"] = source.Address()
		details["account"] = op.Destination.Address()
		details["account_type"] = uint32(op.AccountType)
	case xdr.OperationTypePayment:
		op := c.Operation().Body.MustPaymentOp()
		details["from"] = source.Address()
		details["to"] = op.Destination.Address()
		details["amount"] = amount.String(op.Amount)
		helpers.AssetDetails(details, op.Asset, "")
	case xdr.OperationTypePathPayment:
		op := c.Operation().Body.MustPathPaymentOp()
		details["from"] = source.Address()
		details["to"] = op.Destination.Address()

		result := c.OperationResult().MustPathPaymentResult()

		details["amount"] = amount.String(op.DestAmount)
		details["source_amount"] = amount.String(result.SendAmount())
		details["source_max"] = amount.String(op.SendMax)
		helpers.AssetDetails(details, op.DestAsset, "")
		helpers.AssetDetails(details, op.SendAsset, "source_")

		var path = make([]map[string]interface{}, len(op.Path))
		for i := range op.Path {
			path[i] = make(map[string]interface{})
			helpers.AssetDetails(path[i], op.Path[i], "")
		}
		details["path"] = path
	case xdr.OperationTypeManageOffer:
		op := c.Operation().Body.MustManageOfferOp()
		details["offer_id"] = op.OfferId
		details["amount"] = amount.String(op.Amount)
		details["price"] = op.Price.String()
		details["price_r"] = map[string]interface{}{
			"n": op.Price.N,
			"d": op.Price.D,
		}
		helpers.AssetDetails(details, op.Buying, "buying_")
		helpers.AssetDetails(details, op.Selling, "selling_")

	case xdr.OperationTypeCreatePassiveOffer:
		op := c.Operation().Body.MustCreatePassiveOfferOp()
		details["amount"] = amount.String(op.Amount)
		details["price"] = op.Price.String()
		details["price_r"] = map[string]interface{}{
			"n": op.Price.N,
			"d": op.Price.D,
		}
		helpers.AssetDetails(details, op.Buying, "buying_")
		helpers.AssetDetails(details, op.Selling, "selling_")
	case xdr.OperationTypeSetOptions:
		op := c.Operation().Body.MustSetOptionsOp()

		if op.InflationDest != nil {
			details["inflation_dest"] = op.InflationDest.Address()
		}

		if op.SetFlags != nil && *op.SetFlags > 0 {
			is.operationFlagDetails(details, int32(*op.SetFlags), "set")
		}

		if op.ClearFlags != nil && *op.ClearFlags > 0 {
			is.operationFlagDetails(details, int32(*op.ClearFlags), "clear")
		}

		if op.MasterWeight != nil {
			details["master_key_weight"] = *op.MasterWeight
		}

		if op.LowThreshold != nil {
			details["low_threshold"] = *op.LowThreshold
		}

		if op.MedThreshold != nil {
			details["med_threshold"] = *op.MedThreshold
		}

		if op.HighThreshold != nil {
			details["high_threshold"] = *op.HighThreshold
		}

		if op.HomeDomain != nil {
			details["home_domain"] = *op.HomeDomain
		}

		if op.Signer != nil {
			details["signer_key"] = op.Signer.PubKey.Address()
			details["signer_weight"] = op.Signer.Weight
		}
	case xdr.OperationTypeChangeTrust:
		op := c.Operation().Body.MustChangeTrustOp()
		helpers.AssetDetails(details, op.Line, "")
		details["trustor"] = source.Address()
		details["trustee"] = details["asset_issuer"]
		details["limit"] = amount.String(op.Limit)
	case xdr.OperationTypeAllowTrust:
		op := c.Operation().Body.MustAllowTrustOp()
		helpers.AssetDetails(details, op.Asset.ToAsset(source), "")
		details["trustee"] = source.Address()
		details["trustor"] = op.Trustor.Address()
		details["authorize"] = op.Authorize
	case xdr.OperationTypeAccountMerge:
		aid := c.Operation().Body.MustDestination()
		details["account"] = source.Address()
		details["into"] = aid.Address()
	case xdr.OperationTypeInflation:
		// no inflation details, presently
	case xdr.OperationTypeManageData:
		op := c.Operation().Body.MustManageDataOp()
		details["name"] = string(op.DataName)
		if op.DataValue != nil {
			details["value"] = base64.StdEncoding.EncodeToString(*op.DataValue)
		} else {
			details["value"] = nil
		}
	case xdr.OperationTypeAdministrative:
		op := c.Operation().Body.MustAdminOp()
		var adminOpDetails map[string]interface{}
		err := json.Unmarshal([]byte(op.OpData), &adminOpDetails)
		if err != nil {
			log.WithField("tx_hash", c.Transaction().TransactionHash).WithError(err).Error("Failed to unmarshal admin op details")
		}
		details["details"] = adminOpDetails
	default:
		panic(fmt.Errorf("Unknown operation type: %s", c.OperationType()))
	}

	return details
}

func (is *Session) getAccountType(accountType *xdr.AccountType, address string) error {
	var acc core.Account
	sql := core.SelectAccount.Limit(1).Where("accountid = ?", address)
	err := is.Cursor.DB.Get(&acc, sql)

	if err != nil {
		return err
	}
	*accountType = acc.AccountType

	return nil
}

// operationFlagDetails sets the account flag details for `f` on `result`.
func (is *Session) operationFlagDetails(result map[string]interface{}, f int32, prefix string) {
	var (
		n []int32
		s []string
	)

	if (f & int32(xdr.AccountFlagsAuthRequiredFlag)) > 0 {
		n = append(n, int32(xdr.AccountFlagsAuthRequiredFlag))
		s = append(s, "auth_required")
	}

	if (f & int32(xdr.AccountFlagsAuthRevocableFlag)) > 0 {
		n = append(n, int32(xdr.AccountFlagsAuthRevocableFlag))
		s = append(s, "auth_revocable")
	}

	if (f & int32(xdr.AccountFlagsAuthImmutableFlag)) > 0 {
		n = append(n, int32(xdr.AccountFlagsAuthImmutableFlag))
		s = append(s, "auth_immutable")
	}

	result[prefix+"_flags"] = n
	result[prefix+"_flags_s"] = s
}
