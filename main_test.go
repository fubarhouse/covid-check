package main

import (
	"fmt"
	"testing"
)

var testEndpoint = "https://www.covid19.act.gov.au/act-status-and-response/act-covid-19-exposure-locations"

// TestData is an integration test to ensure certain thresholds are as
// expected. Failing these tests would indicate a change in data
// structure which would mean adjustments need to be made.
func TestData(t *testing.T) {
	covid := &x{}
	var err error
	err = covid.GetHTML(testEndpoint)
	if err != nil {
		t.Fail()
	}
	err = covid.GetCSVReference()
	if err != nil {
		t.Fail()
	}
	err = covid.GetCSVData()
	if err != nil {
		t.Fail()
	}
	covid.SetCSVData()
	if len(covid.RawResults.Items) == 0 {
		t.Fail()
	}

	covid.Query(&Entry{})
	for _, item := range covid.FilteredResults.Items {
		// Is row item nil?
		if fmt.Sprint(&Entry{}) == fmt.Sprint(item) {
			t.Fail()
		}
		// Do the field counts match expected values?
		if item.FieldCount != 13 && item.FieldCount != 14 {
			t.Fail()
		}
	}

}