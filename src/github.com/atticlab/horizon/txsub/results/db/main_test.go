package results

import (
	"testing"

	"github.com/atticlab/horizon/db2/core"
	"github.com/atticlab/horizon/db2/history"
	"github.com/atticlab/horizon/test"
)

func _TestResultProvider(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("base")
	defer tt.Finish()

	rp := &DB{
		Core:    &core.Q{Repo: tt.CoreRepo()},
		History: &history.Q{Repo: tt.HorizonRepo()},
	}

	// Regression: ensure a transaction that is not ingested still returns the
	// result
	hash := "2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d"
	ret := rp.ResultByHash(tt.Ctx, hash)

	tt.Require.NoError(ret.Err)
	tt.Assert.Equal(hash, ret.Hash)
}
