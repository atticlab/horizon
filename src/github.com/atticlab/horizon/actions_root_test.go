package horizon

import (
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/atticlab/horizon/resource"
	"github.com/atticlab/horizon/test"
	"testing"
)

func _TestRootAction(t *testing.T) {

	Convey("GET /", t, func() {
		test.LoadScenario("base")
		server := test.NewStaticMockServer(`{
				"info": {
					"network": "test",
					"build": "test-core"
				}
			}`)
		defer server.Close()
		app := NewTestApp()
		app.horizonVersion = "test-horizon"
		app.config.StellarCoreURL = server.URL

		defer app.Close()
		rh := NewRequestHelper(app)

		w := rh.Get("/", test.RequestHelperNoop)

		So(w.Code, ShouldEqual, 200)

		var result resource.Root
		err := json.Unmarshal(w.Body.Bytes(), &result)
		So(err, ShouldBeNil)

		So(result.HorizonVersion, ShouldEqual, "test-horizon")
		So(result.StellarCoreVersion, ShouldEqual, "test-core")

	})
}