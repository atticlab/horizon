package horizon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/PuerkitoBio/throttled"
	hlog "bitbucket.org/atticlab/horizon/log"
	"bitbucket.org/atticlab/horizon/render/problem"
	"bitbucket.org/atticlab/horizon/test"
	conf "bitbucket.org/atticlab/horizon/config"
)

func NewTestApp() *App {
	app, err := NewApp(NewTestConfig())

	if err != nil {
		log.Panic(err)
	}

	return app
}

func NewTestConfig() conf.Config {
	return conf.Config{
		DatabaseURL:            test.DatabaseURL(),
		StellarCoreDatabaseURL: test.StellarCoreDatabaseURL(),
		RateLimit:              throttled.PerHour(1000),
		LogLevel:               hlog.InfoLevel,
		AdminSignatureValid:    60,
		BankMasterKey: "GBIMZRVQ3W2OXPAK7RBP6XVAWRTPBRL7STFITRCL4QGXXQIFZPK3NDVJ", //SASDDOKKCWHKKRMZ7I3MA4WMI4F4PHG7LOYZGCRZ6WZNAI7TQIER2RTK
	}
}

func NewRequestHelper(app *App) test.RequestHelper {
	return test.NewRequestHelper(app.web.router)
}

func ShouldBePageOf(actual interface{}, options ...interface{}) string {
	body := actual.(*bytes.Buffer)
	expected := options[0].(int)

	var result map[string]interface{}
	err := json.Unmarshal(body.Bytes(), &result)

	if err != nil {
		return fmt.Sprintf("Could not unmarshal json:\n%s\n", body.String())
	}

	embedded, ok := result["_embedded"]

	if !ok {
		return "No _embedded key in response"
	}

	records, ok := embedded.(map[string]interface{})["records"]

	if !ok {
		return "No records key in _embedded"
	}

	length := len(records.([]interface{}))

	if length != expected {
		return fmt.Sprintf("Expected %d records in page, got %d", expected, length)
	}

	return ""
}

func ShouldBeProblem(a interface{}, options ...interface{}) string {
	body := a.(*bytes.Buffer)
	expected := options[0].(problem.P)

	problem.Inflate(test.Context(), &expected)

	var actual problem.P
	err := json.Unmarshal(body.Bytes(), &actual)

	if err != nil {
		return fmt.Sprintf("Could not unmarshal json into problem struct:\n%s\n", body.String())
	}

	if expected.Type != "" && actual.Type != expected.Type {
		return fmt.Sprintf("Mismatched problem type: %s expected, got %s", expected.Type, actual.Type)
	}

	if expected.Status != 0 && actual.Status != expected.Status {
		return fmt.Sprintf("Mismatched problem status: %s expected, got %s", expected.Status, actual.Status)
	}

	// check extras for invalid field
	if len(options) > 1 {
		expectedName := options[1].(string)
		hlog.WithField("extras", actual.Extras).Debug("Got problem with extras")
		actualName, ok := actual.Extras["invalid_field"]
		if !ok {
			return fmt.Sprintf("Expected extras to have invalid_field")
		}
		if expectedName != actualName.(string) {
			return fmt.Sprintf("Mismatched problem invalid field: %s expected, got %s", expectedName, actualName)
		}

	}

	return ""
}
