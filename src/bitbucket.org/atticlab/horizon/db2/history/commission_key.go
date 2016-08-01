package history

import (
	"bitbucket.org/atticlab/go-smart-base/hash"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/resource/base"
	"encoding/hex"
	"encoding/json"
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

func (c *Commission) GetKey() CommissionKey {
	var key CommissionKey
	c.UnmarshalKeyDetails(&key)
	return key
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
