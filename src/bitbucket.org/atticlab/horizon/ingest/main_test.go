package ingest

import (
	"testing"

	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/go-smart-base/network"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/test"
)

func TestIngest(t *testing.T) {
	tt := test.Start(t).ScenarioWithoutHorizon("kahuna")
	defer tt.Finish()

	s := ingest(tt)
	tt.Require.NoError(s.Err)
	tt.Assert.Equal(59, s.Ingested)

	hq := &history.Q{Repo: tt.HorizonRepo()}

	// Ensure the root account was created
	var root history.Account
	err := hq.AccountByID(&root, 1)

	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), root.ID)
	tt.Assert.Equal(keypair.Master(network.TestNetworkPassphrase).Address(), root.Address)

	// Test that re-importing fails
	s.Err = nil
	s.Run()
	tt.Require.Error(s.Err, "Reimport didn't fail as expected")

	// Test that re-importing fails with allowing clear succeeds
	s.Err = nil
	s.ClearExisting = true
	s.Run()
	tt.Require.NoError(s.Err, "Couldn't re-import, even with clear allowed")
}

func ingest(tt *test.T) *Session {
	s, _ := RunOnce(
		network.TestNetworkPassphrase,
		tt.CoreRepo(),
		tt.HorizonRepo(),
	)
	return s
}
