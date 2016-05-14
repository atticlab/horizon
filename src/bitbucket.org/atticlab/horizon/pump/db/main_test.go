package db

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/test"
	"golang.org/x/net/context"
)

func TestStreaming(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	ctx, cancel := context.WithCancel(tt.Ctx)

	Convey("LedgerClosePump", t, func() {

		Convey("can cancel", func() {
			q := &history.Q{Repo: tt.HorizonRepo()}
			pump := NewLedgerClosePump(ctx, q)
			cancel()
			_, more := <-pump
			So(more, ShouldBeFalse)
		})
	})
}
