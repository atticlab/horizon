package admin

import (
	"bitbucket.org/atticlab/horizon/db2/history"
	"bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/test"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestAdminActionProvider(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	log.DefaultLogger.Entry.Logger.Level = log.DebugLevel
	historyQ := &history.Q{tt.HorizonRepo()}

	Convey("Set commission Actions:", t, func() {
		actionProvider := NewAdminActionProvider(historyQ)
		Convey("Several data objects", func() {
			_, err := actionProvider.CreateNewParser(map[string]interface{} {
				string(SubjectCommission): map[string]interface{}{},
				string(SubjectTraits): map[string]interface{}{},
			})
			So(err, ShouldNotBeNil)
		})
		Convey("Unknown action type", func() {
			_, err := actionProvider.CreateNewParser(map[string]interface{} {
				"random_action": map[string]interface{}{},
			})
			So(err, ShouldNotBeNil)
		})
		Convey("Create commission set action", func() {
			action, err := actionProvider.CreateNewParser(map[string]interface{} {
				string(SubjectCommission): map[string]interface{}{},
			})
			So(err, ShouldBeNil)
			switch action.(type) {
			case *SetCommissionAction:
				//ok
			default:
				//not ok
				assert.Fail(t, "Expected SetCommissionAction")
			}
			Convey("Invalid type", func() {
				_, err := actionProvider.CreateNewParser(map[string]interface{} {
					string(SubjectCommission): "random_data",
				})
				So(err, ShouldNotBeNil)
			})
		})
		Convey("Create limits set action", func() {
			action, err := actionProvider.CreateNewParser(map[string]interface{} {
				string(SubjectAccountLimits): map[string]interface{}{},
			})
			So(err, ShouldBeNil)
			switch action.(type) {
			case *SetLimitsAction:
			//ok
			default:
				//not ok
				assert.Fail(t, "Expected SetLimitsAction")
			}
			Convey("Invalid type", func() {
				_, err := actionProvider.CreateNewParser(map[string]interface{} {
					string(SubjectAccountLimits): "random_data",
				})
				So(err, ShouldNotBeNil)
			})
		})
		Convey("Set traits action", func() {
			action, err := actionProvider.CreateNewParser(map[string]interface{} {
				string(SubjectTraits): map[string]interface{}{},
			})
			So(err, ShouldBeNil)
			switch action.(type) {
			case *SetTraitsAction:
			//ok
			default:
				//not ok
				assert.Fail(t, "Expected SetTraitsAction")
			}
			Convey("Invalid type", func() {
				_, err := actionProvider.CreateNewParser(map[string]interface{} {
					string(SubjectTraits): "random_data",
				})
				So(err, ShouldNotBeNil)
			})
		})
	})
}
