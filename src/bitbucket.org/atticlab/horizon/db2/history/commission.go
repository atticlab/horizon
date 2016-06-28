package history

import (
	"bitbucket.org/atticlab/go-smart-base/hash"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/resource/base"
	"encoding/hex"
	"encoding/json"
	sq "github.com/lann/squirrel"
	"strings"
	"sort"
)

type CommissionKey struct {
	From     string     `json:"from,omitempty"`
	To       string     `json:"to,omitempty"`
	FromType int32      `json:"from_type,omitempty"`
	ToType   int32      `json:"to_type,omitempty"`
	Asset    base.Asset `json:"asset,omitempty"`
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

type ByWeight []Commission

func (a ByWeight) Len() int           { return len(a) }
func (a ByWeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByWeight) Less(i, j int) bool { return a[i].weight > a[j].weight }

func (q *Q) GetHighestWeightCommission(keys map[string]CommissionKey) (resultingCommissions []Commission, err error) {
	rawCommissions, err := q.CommissionByKey(keys)
	if err != nil {
		return
	}
	return filterByWeight(rawCommissions), nil
}

func filterByWeight(rawCommissions []Commission) ([]Commission) {
	sort.Sort(ByWeight(rawCommissions))
	bestTo := 0
	for i, val := range rawCommissions {
		if i == 0 {
			continue
		}
		if val.weight != rawCommissions[i -1].weight {
			bestTo = i
			break
		}
	}
	return  rawCommissions[:bestTo]
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
			value.FromType = accountType
		} else {
			value.ToType = accountType
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

func (key *CommissionKey) UnsafeHash() (string) {
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

	if key.FromType != 0 {
		weight += typeWeight
	}

	if key.ToType != 0 {
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
		KeyHash: hash,
		KeyValue: hashData,
		FlatFee: flatFee,
		PercentFee: percentFee,
	}, nil
}

func (q *Q) InsertCommission(commission *Commission) (err error) {
	insert := insertCommission.Values(commission.KeyHash, commission.KeyValue, commission.FlatFee, commission.PercentFee)
	_, err = q.Exec(insert)
	if err != nil {
		log.WithStack(err).WithError(err).Error("Failed to insert commission")
	}
	return
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
	sql := selectCommission.Where("com.key_hash IN (?" + strings.Repeat(", ?", len(hashes) - 1) + ")", hashes...)
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
		if canBeKey == key {
			canBeCom.weight = canBeKey.CountWeight()
			resultingCommissions = append(resultingCommissions, canBeCom)
		}
	}
	return resultingCommissions, nil
}

func (q *Q) deleteCommissions() error {
	_, err := q.Exec(delete)
	return err
}

var selectCommission = sq.Select("com.*").From("commission com")
var insertCommission = sq.Insert("commission").Columns("key_hash", "key_value", "flat_fee", "percent_fee")
var delete = sq.Delete("commission")
