package session

import (
	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/meta"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/cache"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/ingest/session/helpers"
	"bitbucket.org/atticlab/horizon/ingest/session/ingestion"
	"encoding/base64"
	"fmt"
)

// EffectIngestion is a helper struct to smooth the ingestion of effects.  this
// struct will track what the correct operation to use and order to use when
// adding effects into an ingestion.
type EffectIngestion struct {
	Dest        *ingestion.Ingestion
	OperationID int64
	Accounts    *cache.HistoryAccount
	err         error
	added       int
}

func NewEffectIngestion(dest *ingestion.Ingestion, accounts *cache.HistoryAccount, operationId int64) *EffectIngestion {
	return &EffectIngestion{
		Dest:        dest,
		Accounts:    accounts,
		OperationID: operationId,
	}
}

// Add writes an effect to the database while automatically tracking the index
// to use.
func (ei *EffectIngestion) Add(aid xdr.AccountId, typ history.EffectType, details interface{}) bool {
	if ei.err != nil {
		return false
	}

	ei.added++
	var haid int64
	haid, ei.err = ei.Accounts.Get(aid.Address())
	if ei.err != nil {
		return false
	}

	ei.err = ei.Dest.Effect(haid, ei.OperationID, ei.added, typ, details)
	if ei.err != nil {
		return false
	}

	return true
}

// Finish marks this ingestion as complete, returning any error that was recorded.
func (ei *EffectIngestion) Finish() error {
	err := ei.err
	ei.err = nil
	return err
}

func (effects *EffectIngestion) Ingest(cursor *Cursor) {
	source := cursor.OperationSourceAccount()
	opbody := cursor.Operation().Body

	switch cursor.OperationType() {
	case xdr.OperationTypeCreateAccount:
		op := opbody.MustCreateAccountOp()

		effects.Add(op.Destination, history.EffectAccountCreated,
			map[string]interface{}{
				"account_type": uint32(op.AccountType),
			},
		)

		effects.Add(op.Destination, history.EffectSignerCreated,
			map[string]interface{}{
				"public_key": op.Destination.Address(),
				"weight":     keypair.DefaultSignerWeight,
			},
		)

	case xdr.OperationTypePayment:
		op := opbody.MustPaymentOp()
		dets := map[string]interface{}{"amount": amount.String(op.Amount)}
		helpers.AssetDetails(dets, op.Asset, "")
		effects.Add(op.Destination, history.EffectAccountCredited, dets)
		effects.Add(source, history.EffectAccountDebited, dets)
	case xdr.OperationTypePathPayment:
		op := opbody.MustPathPaymentOp()
		dets := map[string]interface{}{"amount": amount.String(op.DestAmount)}
		helpers.AssetDetails(dets, op.DestAsset, "")
		effects.Add(op.Destination, history.EffectAccountCredited, dets)

		result := cursor.OperationResult().MustPathPaymentResult()
		dets = map[string]interface{}{"amount": amount.String(result.SendAmount())}
		helpers.AssetDetails(dets, op.SendAsset, "")
		effects.Add(source, history.EffectAccountDebited, dets)
		effects.ingestTrades(source, result.MustSuccess().Offers)
	case xdr.OperationTypeManageOffer:
		result := cursor.OperationResult().MustManageOfferResult().MustSuccess()
		effects.ingestTrades(source, result.OffersClaimed)
	case xdr.OperationTypeCreatePassiveOffer:
		claims := []xdr.ClaimOfferAtom{}
		result := cursor.OperationResult()

		// KNOWN ISSUE:  stellar-core creates results for CreatePassiveOffer operations
		// with the wrong result arm set.
		if result.Type == xdr.OperationTypeManageOffer {
			claims = result.MustManageOfferResult().MustSuccess().OffersClaimed
		} else {
			claims = result.MustCreatePassiveOfferResult().MustSuccess().OffersClaimed
		}

		effects.ingestTrades(source, claims)
	case xdr.OperationTypeSetOptions:
		op := opbody.MustSetOptionsOp()

		if op.HomeDomain != nil {
			effects.Add(source, history.EffectAccountHomeDomainUpdated,
				map[string]interface{}{
					"home_domain": string(*op.HomeDomain),
				},
			)
		}

		thresholdDetails := map[string]interface{}{}

		if op.LowThreshold != nil {
			thresholdDetails["low_threshold"] = *op.LowThreshold
		}

		if op.MedThreshold != nil {
			thresholdDetails["med_threshold"] = *op.MedThreshold
		}

		if op.HighThreshold != nil {
			thresholdDetails["high_threshold"] = *op.HighThreshold
		}

		if len(thresholdDetails) > 0 {
			effects.Add(source, history.EffectAccountThresholdsUpdated, thresholdDetails)
		}

		flagDetails := map[string]bool{}
		helpers.FlagDetails(flagDetails, op.SetFlags, true)
		helpers.FlagDetails(flagDetails, op.ClearFlags, false)

		if len(flagDetails) > 0 {
			effects.Add(source, history.EffectAccountFlagsUpdated, flagDetails)
		}

		effects.ingestSignerEffects(cursor, op)

	case xdr.OperationTypeChangeTrust:
		op := opbody.MustChangeTrustOp()
		dets := map[string]interface{}{"limit": amount.String(op.Limit)}
		key := xdr.LedgerKey{}
		effect := history.EffectType(0)

		helpers.AssetDetails(dets, op.Line, "")

		key.SetTrustline(source, op.Line)

		before, after, err := cursor.BeforeAndAfter(key)

		// NOTE:  when an account trusts itself, the transaction is successful but
		// no ledger entries are actually modified, leading to an "empty meta"
		// situation.  We simply continue on to the next operation in that scenario.
		if err == meta.ErrMetaNotFound {
			return
		}

		if err != nil {
			effects.err = err
			return
		}

		switch {
		case before == nil && after != nil:
			effect = history.EffectTrustlineCreated
		case before != nil && after == nil:
			effect = history.EffectTrustlineRemoved
		case before != nil && after != nil:
			effect = history.EffectTrustlineUpdated
		default:
			panic("Invalid before-and-after state")
		}

		effects.Add(source, effect, dets)
	case xdr.OperationTypeAllowTrust:
		op := opbody.MustAllowTrustOp()
		asset := op.Asset.ToAsset(source)
		dets := map[string]interface{}{
			"trustor": op.Trustor.Address(),
		}
		helpers.AssetDetails(dets, asset, "")

		if op.Authorize {
			effects.Add(source, history.EffectTrustlineAuthorized, dets)
		} else {
			effects.Add(source, history.EffectTrustlineDeauthorized, dets)
		}

	case xdr.OperationTypeAccountMerge:
		dest := opbody.MustDestination()
		result := cursor.OperationResult().MustAccountMergeResult()
		dets := map[string]interface{}{
			"amount":     amount.String(result.MustSourceAccountBalance()),
			"asset_type": "native",
		}
		effects.Add(source, history.EffectAccountDebited, dets)
		effects.Add(dest, history.EffectAccountCredited, dets)
		effects.Add(source, history.EffectAccountRemoved, map[string]interface{}{})
	case xdr.OperationTypeInflation:
		payouts := cursor.OperationResult().MustInflationResult().MustPayouts()
		for _, payout := range payouts {
			effects.Add(payout.Destination, history.EffectAccountCredited,
				map[string]interface{}{
					"amount":     amount.String(payout.Amount),
					"asset_type": "native",
				},
			)
		}
	case xdr.OperationTypeManageData:
		op := opbody.MustManageDataOp()
		dets := map[string]interface{}{"name": op.DataName}
		key := xdr.LedgerKey{}
		effect := history.EffectType(0)

		key.SetData(source, string(op.DataName))

		before, after, err := cursor.BeforeAndAfter(key)
		if err != nil {
			effects.err = err
			return
		}

		if after != nil {
			raw := after.Data.MustData().DataValue
			dets["value"] = base64.StdEncoding.EncodeToString(raw)
		}

		switch {
		case before == nil && after != nil:
			effect = history.EffectDataCreated
		case before != nil && after == nil:
			effect = history.EffectDataRemoved
		case before != nil && after != nil:
			effect = history.EffectDataUpdated
		default:
			panic("Invalid before-and-after state")
		}

		effects.Add(source, effect, dets)
	case xdr.OperationTypeAdministrative:
		opbody.MustAdminOp()
	// no need to duplicate data

	default:
		effects.err = fmt.Errorf("Unknown operation type: %s", cursor.OperationType())
		return
	}
}

func (effects *EffectIngestion) ingestTrades(buyer xdr.AccountId, claims []xdr.ClaimOfferAtom) {
	for _, claim := range claims {
		seller := claim.SellerId
		bd, sd := helpers.TradeDetails(buyer, seller, claim)
		effects.Add(buyer, history.EffectTrade, bd)
		effects.Add(seller, history.EffectTrade, sd)
	}
}

func (effects *EffectIngestion) ingestSignerEffects(cursor *Cursor, op xdr.SetOptionsOp) {
	source := cursor.OperationSourceAccount()

	be, ae, err := cursor.BeforeAndAfter(source.LedgerKey())
	if err != nil {
		effects.err = err
		return
	}

	beforeAccount := be.Data.MustAccount()
	afterAccount := ae.Data.MustAccount()

	before := beforeAccount.SignerSummary()
	after := afterAccount.SignerSummary()

	for addy := range before {
		weight, ok := after[addy]
		if !ok {
			effects.Add(source, history.EffectSignerRemoved, map[string]interface{}{
				"public_key": addy,
			})
			continue
		}
		effects.Add(source, history.EffectSignerUpdated, map[string]interface{}{
			"public_key": addy,
			"weight":     weight,
		})
	}
	// Add the "created" effects
	for addy, weight := range after {
		// if `addy` is in before, the previous for loop should have recorded
		// the update, so skip this key
		if _, ok := before[addy]; ok {
			continue
		}

		effects.Add(source, history.EffectSignerCreated, map[string]interface{}{
			"public_key": addy,
			"weight":     weight,
		})
	}

}
