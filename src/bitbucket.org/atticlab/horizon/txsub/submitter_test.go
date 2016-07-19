package txsub

import (
	"bitbucket.org/atticlab/go-smart-base/build"
	"bitbucket.org/atticlab/go-smart-base/keypair"
	"bitbucket.org/atticlab/horizon/db2/core"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/test"
	"bitbucket.org/atticlab/horizon/txsub/results"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"testing"
)

func TestDefaultSubmitter(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel
	log.Debug("TestDefaultSubmitter")
	ctx := test.Context()
	historyQ := &history.Q{tt.HorizonRepo()}
	coreQ := &core.Q{tt.CoreRepo()}
	config := test.NewTestConfig()

	Convey("submitter (The default Submitter implementation)", t, func() {
		newAccount, err := keypair.Random()
		So(err, ShouldBeNil)
		createAccount := build.CreateAccount(build.Destination{newAccount.Address()})
		tx := build.Transaction(createAccount, build.Sequence{1}, build.SourceAccount{newAccount.Address()})
		txE := tx.Sign(newAccount.Seed())
		rawTxE, err := txE.Base64()
		So(err, ShouldBeNil)

		Convey("submits to the configured stellar-core instance correctly", func() {
			server := test.NewStaticMockServer(`{
				"status": "PENDING",
				"error": null
				}`)
			defer server.Close()

			s := NewDefaultSubmitter(http.DefaultClient, server.URL, coreQ, historyQ, &config)
			log.Debug("Submiting tx")
			sr := s.Submit(ctx, rawTxE)
			log.Debug("Checking submition result")
			So(sr.Err, ShouldBeNil)
			So(sr.Duration, ShouldBeGreaterThan, 0)
		})

		Convey("succeeds when the stellar-core responds with DUPLICATE status", func() {
			server := test.NewStaticMockServer(`{
				"status": "DUPLICATE",
				"error": null
				}`)
			defer server.Close()

			s := NewDefaultSubmitter(http.DefaultClient, server.URL, coreQ, historyQ, &config)
			sr := s.Submit(ctx, rawTxE)
			So(sr.Err, ShouldBeNil)
		})

		Convey("errors when the stellar-core url is not reachable", func() {
			s := NewDefaultSubmitter(http.DefaultClient, "http://127.0.0.1:65535", coreQ, historyQ, &config)
			sr := s.Submit(ctx, rawTxE)
			So(sr.Err, ShouldNotBeNil)
		})

		Convey("errors when the stellar-core returns an unparseable response", func() {
			server := test.NewStaticMockServer(`{`)
			defer server.Close()

			s := NewDefaultSubmitter(http.DefaultClient, server.URL, coreQ, historyQ, &config)
			sr := s.Submit(ctx, rawTxE)
			So(sr.Err, ShouldNotBeNil)
		})

		Convey("errors when the stellar-core returns an exception response", func() {
			server := test.NewStaticMockServer(`{"exception": "Invalid XDR"}`)
			defer server.Close()

			s := NewDefaultSubmitter(http.DefaultClient, server.URL, coreQ, historyQ, &config)
			sr := s.Submit(ctx, rawTxE)
			So(sr.Err, ShouldNotBeNil)
			So(sr.Err.Error(), ShouldContainSubstring, "Invalid XDR")
		})

		Convey("errors when the stellar-core returns an unrecognized status", func() {
			server := test.NewStaticMockServer(`{"status": "NOTREAL"}`)
			defer server.Close()

			s := NewDefaultSubmitter(http.DefaultClient, server.URL, coreQ, historyQ, &config)
			sr := s.Submit(ctx, rawTxE)
			So(sr.Err, ShouldNotBeNil)
			So(sr.Err.Error(), ShouldContainSubstring, "NOTREAL")
		})

		Convey("errors when the stellar-core returns an error response", func() {
			server := test.NewStaticMockServer(`{"status": "ERROR", "error": "1234"}`)
			defer server.Close()

			s := NewDefaultSubmitter(http.DefaultClient, server.URL, coreQ, historyQ, &config)
			sr := s.Submit(ctx, rawTxE)
			So(sr.Err, ShouldHaveSameTypeAs, &results.FailedTransactionError{})
			ferr := sr.Err.(*results.FailedTransactionError)
			So(ferr.ResultXDR, ShouldEqual, "1234")
		})
	})
}
