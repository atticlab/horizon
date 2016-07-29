package history

import (
	"bitbucket.org/atticlab/go-smart-base/hash"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/db2"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/resource/base"
	"encoding/hex"
	"encoding/json"
	"github.com/go-errors/errors"
	sq "github.com/lann/squirrel"
	"sort"
	"strings"
)

type CommissionKey struct {
	base.Asset
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	FromType *int32 `json:"from_type,omitempty"`
	ToType   *int32 `json:"to_type,omitempty"`
}

func (k *CommissionKey) Equals(o CommissionKey) bool {
	if k.Asset != o.Asset || k.From != o.From || k.To != o.To {
		return false
	}
	return equals(k.FromType, o.FromType) && equals(k.ToType, o.ToType)
}

func equals(l, r *int32) bool {
	if l == nil || r == nil {
		return r == l
	}
	return *l == *r
}

func CreateCommissionKeys(from, to string, fromType, toType int32, asset base.Asset) map[string]CommissionKey {
	result := make(map[string]CommissionKey)
	defaultFee := CommissionKey{}
	result[defaultFee.UnsafeHash()] = defaultFee
	setAsset(result, asset)
	setType(result, fromType, true)
	setType(result, toType, false)
	setAccount(result, from, true)
	setAccount(result, to, false)
	return result
}

func (c *Commission) GetKey() CommissionKey {
	var key CommissionKey
	c.UnmarshalKeyDetails(&key)
	return key
}

func (c Commission) Equals(o Commission) bool {
	if c.KeyHash != o.KeyHash || c.FlatFee != o.FlatFee || c.PercentFee != o.PercentFee {
		return false
	}
	cKey := c.GetKey()
	return cKey.Equals(o.GetKey())
}

type ByWeight []Commission

func (a ByWeight) Len() int           { return len(a) }
func (a ByWeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeight) Less(i, j int) bool { return a[i].weight > a[j].weight }

func (q *Q) GetHighestWeightCommission(keys map[string]CommissionKey) (resultingCommissions []Commission, err error) {
	rawCommissions, err := q.CommissionByKey(keys)
	if err != nil {
		return
	}
	log.WithField("len", len(rawCommissions)).Debug("Got commissions")
	return filterByWeight(rawCommissions), nil
}

func filterByWeight(rawCommissions []Commission) []Commission {
	if len(rawCommissions) == 0 {
		return rawCommissions
	}
	sort.Sort(ByWeight(rawCommissions))
	bestTo := 0
	for i, val := range rawCommissions {
		if i == 0 {
			continue
		}
		if val.weight != rawCommissions[i-1].weight {
			bestTo = i - 1
			break
		}
	}
	result := rawCommissions[:bestTo+1]
	log.WithField("len", len(result)).WithField("commissions", result).Debug("Filtered commissions")
	return result
}

func setAsset(keys map[string]CommissionKey, asset base.Asset) {
	for _, value := range keys {
		value.Asset = asset
		keys[value.UnsafeHash()] = value
	}
}

func setType(keys map[string]CommissionKey, accountType int32, isFrom bool) {
	for _, value := range keys {
		if isFrom {
			value.FromType = &accountType
		} else {
			value.ToType = &accountType
		}
		keys[value.UnsafeHash()] = value
	}
}

func setAccount(keys map[string]CommissionKey, account string, isFrom bool) {
	for _, value := range keys {
		if isFrom {
			value.From = account
		} else {
			value.To = account
		}
		keys[value.UnsafeHash()] = value
	}
}

func (key *CommissionKey) Hash() (string, error) {
	_, hash, err := key.HashData()
	return hash, err
}

func (key *CommissionKey) UnsafeHash() string {
	result, _ := key.Hash()
	return result
}

func (key *CommissionKey) HashData() (hashData string, hashValue string, err error) {
	hashDataByte, err := json.Marshal(key)
	if err != nil {
		log.WithField("Error", err).Error("Failed to marshal commission key")
		return "", "", err
	}
	hashBase := hash.Hash(hashDataByte)
	hashValue = hex.EncodeToString(hashBase[:])
	hashData = string(hashDataByte)
	return
}

// returns 1 if key hash higher priority, -1 if lower, 0 if equal
func (key *CommissionKey) Compare(other *CommissionKey) int {
	if other == nil {
		return 1
	}

	keyWeight := key.CountWeight()
	otherWeight := other.CountWeight()
	log.WithField("keyWeight", keyWeight).WithField("otherWeight", otherWeight).Debug("counted weight")

	if keyWeight > otherWeight {
		return 1
	} else if keyWeight < otherWeight {
		return -1
	}
	return 0
}

const (
	assetWeight   = 1
	typeWeight    = assetWeight + 1
	accountWeight = typeWeight*2 + assetWeight + 1
)

func (key *CommissionKey) IsAssetSet() bool {
	asset := &key.Asset
	return asset.Type != "" || asset.Code != "" || asset.Issuer != ""
}

func (key *CommissionKey) CountWeight() int {
	weight := 0

	if key.IsAssetSet() {
		weight += assetWeight
	}

	if key.FromType != nil {
		weight += typeWeight
	}

	if key.ToType != nil {
		weight += typeWeight
	}

	if key.From != "" {
		weight += accountWeight
	}

	if key.To != "" {
		weight += accountWeight
	}

	return weight
}

func NewCommission(key CommissionKey, flatFee, percentFee int64) (*Commission, error) {
	hashData, hash, err := key.HashData()
	if err != nil {
		log.WithStack(err).Error("Failed to get hash for commission key: " + err.Error())
		return nil, err
	}
	return &Commission{
		KeyHash:    hash,
		KeyValue:   hashData,
		FlatFee:    flatFee,
		PercentFee: percentFee,
	}, nil
}

func (q *Q) InsertCommission(commission *Commission) (err error) {
	if commission == nil {
		return
	}

	insert := insertCommission.Values(commission.KeyHash, commission.KeyValue, commission.FlatFee, commission.PercentFee)
	_, err = q.Exec(insert)
	if err != nil {
		log.WithStack(err).WithError(err).WithField("commission", *commission).Error("Failed to insert commission")
	}
	return
}

func (q *Q) UpdateCommission(commission *Commission) (bool, error) {
	if commission == nil {
		return false, nil
	}
	update := updateCommission.SetMap(map[string]interface{}{
		"key_hash":    commission.KeyHash,
		"key_value":   commission.KeyValue,
		"flat_fee":    commission.FlatFee,
		"percent_fee": commission.PercentFee,
	}).Where("id = ?", commission.Id)
	result, err := q.Exec(update)
	if err != nil {
		log.WithStack(err).WithField("commission", *commission).WithError(err).Error("Failed to update commission")
		return false, nil
	}
	rows, err := result.RowsAffected()
	if err != nil {
		log.WithStack(err).WithField("commission", *commission).WithError(err).Error("Failed to update commission")
		return false, nil
	}
	return rows > 0, nil
}

func (q *Q) DeleteCommission(id int64) (bool, error) {
	deleteQ := deleteCommission.Where("id = ?", id)
	result, err := q.Exec(deleteQ)
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to delete commission")
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows != 0, err
}

func getHashes(keys map[string]CommissionKey) []interface{} {
	result := make([]interface{}, len(keys))
	idx := 0
	for key := range keys {
		result[idx] = key
		idx++
	}
	return result
}

// AccountByAddress loads a row from `history_accounts`, by address
func (q *Q) CommissionByKey(keys map[string]CommissionKey) (resultingCommissions []Commission, err error) {
	if len(keys) == 0 {
		return
	}
	hashes := getHashes(keys)
	sql := selectCommission.Where("com.key_hash IN (?"+strings.Repeat(", ?", len(hashes)-1)+")", hashes...)
	var storedCommissions []Commission
	err = q.Select(&storedCommissions, sql)
	if err != nil {
		log.WithStack(err).Error("Failed to get commission by key: " + err.Error())
		return nil, err
	}
	resultingCommissions = make([]Commission, 0, len(storedCommissions))
	for _, canBeCom := range storedCommissions {
		var canBeKey CommissionKey
		err := json.Unmarshal([]byte(canBeCom.KeyValue), &canBeKey)
		if err != nil {
			log.WithField("hash", canBeCom.KeyHash).WithError(err).Error("Failed to get key value for commission")
			return nil, err
		}
		key, isExist := keys[canBeCom.KeyHash]
		if !isExist {
			continue
		}
		if key.Equals(canBeKey) {
			canBeCom.weight = canBeKey.CountWeight()
			resultingCommissions = append(resultingCommissions, canBeCom)
		}
	}
	return resultingCommissions, nil
}

func (q *Q) CommissionById(id int64) (*Commission, error) {
	sql := selectCommission.Where("com.id = ?", id)
	var storedCommissions []Commission
	err := q.Select(&storedCommissions, sql)
	if err != nil {
		log.Error("Failed to get commission by key: " + err.Error())
		return nil, err
	}

	if len(storedCommissions) == 0 {
		return nil, nil
	}
	return &storedCommissions[0], nil
}

func (q *Q) DeleteCommissions() error {
	_, err := q.Exec(deleteCommission)
	return err
}

// UnmarshalDetails unmarshals the details of this effect into `dest`
func (r *Commission) UnmarshalKeyDetails(dest interface{}) error {

	err := json.Unmarshal([]byte(r.KeyValue), &dest)
	if err != nil {
		err = errors.Wrap(err, 1)
	}
	return err
}

// Commissions provides a helper to filter rows from the `commission`
// table with pre-defined filters.
func (q *Q) Commissions() *CommissionQ {
	return &CommissionQ{
		parent: q,
		sql:    selectCommission,
	}
}

// ForAccount filters the commission collection to a specific account
func (q *CommissionQ) ForAccount(aid string) *CommissionQ {
	q.sql = q.sql.Where("(com.key_value->>'from' = ? OR com.key_value->>'to' = ?)", aid, aid)
	return q
}

// ForAccountType filters the query to only commission for a specific account type
func (q *CommissionQ) ForAccountType(accountType int32) *CommissionQ {
	q.sql = q.sql.Where("(com.key_value->>'from_type' = ? OR com.key_value->>'to_type' = ?)", accountType, accountType)
	return q
}

// ForAccountType filters the query to only commission for a specific asset
func (q *CommissionQ) ForAsset(asset base.Asset) *CommissionQ {

	if asset.Type == xdr.AssetTypeAssetTypeNative.String() {
		clause := `(com.key_value->>'asset_type' = ?
		AND com.key_value ?? 'asset_code' = false
		AND com.key_value ?? 'asset_issuer' = false)`
		q.sql = q.sql.Where(clause, asset.Type)
		return q
	}

	clause := `(com.key_value->>'asset_type' = ?
	AND com.key_value->>'asset_code' = ?
	AND com.key_value->>'asset_issuer' = ?)`
	q.sql = q.sql.Where(clause, asset.Type, asset.Code, asset.Issuer)
	return q
}

// Page specifies the paging constraints for the query being built by `q`.
func (q *CommissionQ) Page(page db2.PageQuery) *CommissionQ {
	if q.Err != nil {
		return q
	}

	q.sql, q.Err = page.ApplyTo(q.sql, "com.id")
	return q
}

// Select loads the results of the query specified by `q` into `dest`.
func (q *CommissionQ) Select(dest interface{}) error {
	if q.Err != nil {
		log.WithStack(q.Err).WithError(q.Err).Error("Failed to create query to select commissions")
		return q.Err
	}

	strSql, args, _ := q.sql.ToSql()
	log.WithField("query", strSql).WithField("args", args).Debug("Tring to get commissions")
	q.Err = q.parent.Select(dest, q.sql)
	if q.Err != nil {
		log.WithStack(q.Err).WithError(q.Err).WithField("query", strSql).Error("Failed to select commissions")
	}
	return q.Err
}

var selectCommission = sq.Select("com.*").From("commission com")
var insertCommission = sq.Insert("commission").Columns("key_hash", "key_value", "flat_fee", "percent_fee")
var updateCommission = sq.Update("commission")
var deleteCommission = sq.Delete("commission")
