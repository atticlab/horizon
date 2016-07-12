package actions

import (
	"mime"
	"strconv"

	"bitbucket.org/atticlab/go-smart-base/amount"
	"bitbucket.org/atticlab/go-smart-base/strkey"
	"bitbucket.org/atticlab/go-smart-base/xdr"
	"bitbucket.org/atticlab/horizon/assets"
	"bitbucket.org/atticlab/horizon/db2"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"errors"
)

const (
	// ParamCursor is a query string param name
	ParamCursor = "cursor"
	// ParamOrder is a query string param name
	ParamOrder = "order"
	// ParamLimit is a query string param name
	ParamLimit = "limit"
)

// GetString retrieves a string from either the URLParams, form or query string.
// This method uses the priority (URLParams, Form, Query).
func (base *Base) GetString(name string) string {
	if base.Err != nil {
		return ""
	}

	fromURL, ok := base.GojiCtx.URLParams[name]

	if ok {
		return fromURL
	}

	fromForm := base.R.FormValue(name)

	if fromForm != "" {
		return fromForm
	}

	return base.R.URL.Query().Get(name)
}

func (base *Base) GetOptionalBool(name string) *bool {
	if base.Err != nil {
		return nil
	}

	asStr := base.GetString(name)
	if asStr == "" {
		return nil
	}

	result, err := strconv.ParseBool(asStr)
	if err != nil {
		base.SetInvalidField(name, err)
		return nil
	}
	return &result
}

func (base *Base) GetBool(name string) bool {
	result := base.GetOptionalBool(name)
	if result != nil {
		return *result
	}
	return false
}

// GetInt64 retrieves an int64 from the action parameter of the given name.
// Populates err if the value is not a valid int64
func (base *Base) GetInt64(name string) int64 {
	if base.Err != nil {
		return 0
	}

	asStr := base.GetString(name)

	if asStr == "" {
		return 0
	}

	asI64, err := strconv.ParseInt(asStr, 10, 64)

	if err != nil {
		base.SetInvalidField(name, err)
		return 0
	}

	return asI64
}

func (base *Base) GetInt32(name string) int32 {
	result := base.GetInt32Pointer(name)
	if result == nil {
		return 0
	}
	return *result
}

// GetInt32 retrieves an int32 from the action parameter of the given name.
// Populates err if the value is not a valid int32
func (base *Base) GetInt32Pointer(name string) *int32 {
	if base.Err != nil {
		return nil
	}

	asStr := base.GetString(name)

	if asStr == "" {
		return nil
	}

	asI64, err := strconv.ParseInt(asStr, 10, 32)

	if err != nil {
		base.SetInvalidField(name, err)
		return nil
	}

	result := int32(asI64)
	return &result
}

// GetPagingParams returns the cursor/order/limit triplet that is the
// standard way of communicating paging data to a horizon endpoint.
func (base *Base) GetPagingParams() (cursor string, order string, limit uint64) {
	if base.Err != nil {
		return
	}

	cursor = base.GetString(ParamCursor)
	order = base.GetString(ParamOrder)
	// TODO: add GetUint64 helpers
	limit = uint64(base.GetInt64(ParamLimit))

	if lei := base.R.Header.Get("Last-Event-ID"); lei != "" {
		cursor = lei
	}

	return
}

// GetPageQuery is a helper that returns a new db.PageQuery struct initialized
// using the results from a call to GetPagingParams()
func (base *Base) GetPageQuery() db2.PageQuery {
	if base.Err != nil {
		return db2.PageQuery{}
	}

	r, err := db2.NewPageQuery(base.GetPagingParams())

	if err != nil {
		base.Err = err
	}

	return r
}

// GetAddress retrieves a stellar address.  It confirms the value loaded is a
// valid stellar address, setting an invalid field error if it is not.
func (base *Base) GetAddress(name string) (result string) {
	if base.Err != nil {
		return
	}

	result = base.GetString(name)

	_, err := strkey.Decode(strkey.VersionByteAccountID, result)

	if err != nil {
		base.SetInvalidField(name, err)
	}

	return result
}

func (base *Base) GetAccountID(name string) (result xdr.AccountId) {
	if base.Err != nil {
		return
	}

	accountId := base.GetOptionalAccountID(name)
	if base.Err != nil {
		return
	}

	if accountId == nil {
		base.SetInvalidField(name, errors.New("can not be empty"))
		return
	}
	result = *accountId
	return
}

func (base *Base) GetOptionalAccountType(name string) (result *xdr.AccountType) {
	if base.Err != nil {
		return
	}
	rawType := base.GetInt32Pointer(name)
	if rawType == nil {
		return nil
	}

	if !xdr.AccountTypeAccountAnonymousUser.ValidEnum(*rawType) {
		base.SetInvalidField(name, errors.New("invalid value for account type"))
		return
	}
	accountType := xdr.AccountType(*rawType)
	return &accountType
}

// GetAccountID retireves an xdr.AccountID by attempting to decode a stellar
// address at the provided name.
func (base *Base) GetOptionalAccountID(name string) (result *xdr.AccountId) {
	if base.Err != nil {
		return nil
	}

	strData := base.GetString(name)
	if strData == "" {
		return nil
	}
	raw, err := strkey.Decode(strkey.VersionByteAccountID, strData)

	if base.Err != nil {
		return nil
	}

	if err != nil {
		base.SetInvalidField(name, err)
		return nil
	}

	var key xdr.Uint256
	copy(key[:], raw)

	rawResult, err := xdr.NewAccountId(xdr.CryptoKeyTypeKeyTypeEd25519, key)
	if err != nil {
		base.SetInvalidField(name, err)
		return nil
	}

	return &rawResult
}

func (base *Base) GetPositiveAmount(name string) (result xdr.Int64) {
	if base.Err != nil {
		return 0
	}
	result = base.GetAmount(name)
	if base.Err != nil {
		return 0
	}

	if result <= 0 {
		base.SetInvalidField(name, errors.New("must be positive"))
	}
	return
}

// GetAmount returns a native amount (i.e. 64-bit integer) by parsing
// the string at the provided name in accordance with the stellar client
// conventions
func (base *Base) GetAmount(name string) (result xdr.Int64) {
	if base.Err != nil {
		return 0
	}
	strAmount := base.GetString(name)
	log.WithField("strAmount", strAmount).WithField("name", name).Debug("Got raw amount")
	result, err := amount.Parse(strAmount)

	if err != nil {
		base.SetInvalidField(name, err)
		return
	}

	return
}

// GetAssetType is a helper that returns a xdr.AssetType by reading a string
func (base *Base) GetAssetType(name string) xdr.AssetType {
	if base.Err != nil {
		return xdr.AssetTypeAssetTypeNative
	}

	r, err := assets.Parse(base.GetString(name))

	if base.Err != nil {
		return xdr.AssetTypeAssetTypeNative
	}

	if err != nil {
		base.SetInvalidField(name, err)
	}

	return r
}

// GetAsset decodes an asset from the request fields prefixed by `prefix`.  To
// succeed, three prefixed fields must be present: asset_type, asset_code, and
// asset_issuer.
func (base *Base) GetAsset(prefix string) (result xdr.Asset) {
	if base.Err != nil {
		return
	}
	var value interface{}

	t := base.GetAssetType(prefix + "asset_type")

	switch t {
	case xdr.AssetTypeAssetTypeCreditAlphanum4:
		a := xdr.AssetAlphaNum4{}
		a.Issuer = base.GetAccountID(prefix + "asset_issuer")

		c := base.GetString(prefix + "asset_code")
		if len(c) > len(a.AssetCode) {
			base.SetInvalidField(prefix+"asset_code", nil)
			return
		}

		copy(a.AssetCode[:len(c)], []byte(c))
		value = a
	case xdr.AssetTypeAssetTypeCreditAlphanum12:
		a := xdr.AssetAlphaNum12{}
		a.Issuer = base.GetAccountID(prefix + "asset_issuer")

		c := base.GetString(prefix + "asset_code")
		if len(c) > len(a.AssetCode) {
			base.SetInvalidField(prefix+"asset_code", nil)
			return
		}

		copy(a.AssetCode[:len(c)], []byte(c))
		value = a
	}

	result, err := xdr.NewAsset(t, value)
	if err != nil {
		panic(err)
	}
	return
}

// SetInvalidField establishes an error response triggered by an invalid
// input field from the user.
func (base *Base) SetInvalidField(name string, reason error) {
	log.WithField("name", name).WithError(reason).Info("Setting invalid field")
	br := problem.BadRequest

	br.Extras = map[string]interface{}{}
	br.Extras["invalid_field"] = name
	br.Extras["reason"] = reason.Error()

	base.Err = &br
}

// Path returns the current action's path, as determined by the http.Request of
// this action
func (base *Base) Path() string {
	return base.R.URL.Path
}

// ValidateBodyType sets an error on the action if the requests Content-Type
//  is not `application/x-www-form-urlencoded`
func (base *Base) ValidateBodyType() {
	c := base.R.Header.Get("Content-Type")

	if c == "" {
		return
	}

	mt, _, err := mime.ParseMediaType(c)

	if err != nil {
		base.Err = err
		return
	}

	switch {
	case mt == "application/x-www-form-urlencoded":
		return
	case mt == "multipart/form-data":
		return
	default:
		base.Err = &problem.UnsupportedMediaType
	}
}
