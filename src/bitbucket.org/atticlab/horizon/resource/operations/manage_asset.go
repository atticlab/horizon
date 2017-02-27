package operations

import "bitbucket.org/atticlab/horizon/db2/history/details"

type ManageAsset struct {
	Base
	details.Asset
	IsAnonymous bool `json:"is_anonymous"`
	IsDelete    bool `json:"is_delete"`
}
