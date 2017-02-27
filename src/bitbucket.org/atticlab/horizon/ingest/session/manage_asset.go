package session

import (
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"database/sql"
	"github.com/go-errors/errors"
)

func (is *Session) ingestManageAsset(manageAssetOp xdr.ManageAssetOp) error {
	logger := log.WithField("service", "manage_asset_ingester")

	var storedAsset history.Asset
	isNew := false
	err := is.Ingestion.HistoryQ().Asset(&storedAsset, manageAssetOp.Asset)
	if err == sql.ErrNoRows {
		isNew = true
		err = nil
	}

	if err != nil {
		logger.WithError(err).Error("Failed to get asset")
		return err
	}

	if manageAssetOp.IsDelete {
		if isNew {
			return errors.New("Tring to delete non existing asset")
		}

		_, err := is.Ingestion.HistoryQ().DeleteAsset(storedAsset.Id)
		if err != nil {
			logger.WithError(err).Error("Failed to delete asset")
		}

		return err
	}



	if !isNew {
		storedAsset.IsAnonymous = manageAssetOp.IsAnonymous
		_, err = is.Ingestion.HistoryQ().UpdateAsset(&storedAsset)
		if err != nil {
			logger.WithError(err).Error("Failed to update asset")
		}
		return err
	}

	var code, issuer string
	var assetType xdr.AssetType
	err = manageAssetOp.Asset.Extract(&assetType, &code, &issuer)
	if err != nil {
		logger.WithError(err).Error("Failed to extract asset data")
		return err
	}

	storedAsset.Type = int(assetType)
	storedAsset.Code = code
	storedAsset.Issuer = issuer
	storedAsset.IsAnonymous = manageAssetOp.IsAnonymous

	err = is.Ingestion.HistoryQ().InsertAsset(&storedAsset)
	if err != nil {
		logger.WithError(err).Error("Failed to insert asset")
	}
	return err
}
