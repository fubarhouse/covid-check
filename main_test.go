package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

var testEndpoint = "https://www.covid19.act.gov.au/act-status-and-response/act-covid-19-exposure-locations"

// TestDataLengthDynamic will check for entries known to be specific
// lengths to be those specific lengths. This is tested dynamically
// with by querying known data - opposed to the tests which follow
// which provide static data to the same test.
func TestDataLengthDynamic(t *testing.T) {
	covid := &x{}
	var err error
	t.Run("Getting Endpoint", func(t *testing.T) {
		err = covid.GetHTML(testEndpoint)
		if err != nil {
			t.Fail()
		}
	})
	t.Run("Getting CSV File URL", func(t *testing.T) {
		err = covid.GetCSVReference()
		if err != nil {
			t.Fail()
		}
	})
	t.Run("Getting CSV File Contents", func(t *testing.T) {
		err = covid.GetCSVData()
		if err != nil {
			t.Fail()
		}
	})
	t.Run("Translating CSV File to Struct", func(t *testing.T) {
		covid.SetCSVData()
		if len(covid.RawResults.Items) == 0 {
			t.Fail()
		}
	})
	t.Run("CLeaning Raw CSV data", func(t *testing.T) {
		covid.Clean()
		for _, line := range strings.Split(covid.RawCSV, "\n") {
			if strings.HasPrefix(line, string(rune(13))) {
				t.Fail()
			}
			if strings.HasSuffix(line, string(rune(13))) {
				t.Fail()
			}
			if strings.HasPrefix(line, string(rune(33))) {
				t.Fail()
			}
			if strings.HasSuffix(line, string(rune(33))) {
				t.Fail()
			}
			if strings.HasPrefix(line, string(rune(44))) {
				t.Fail()
			}
			if strings.HasSuffix(line, string(rune(44))) {
				t.Fail()
			}
		}
	})
}

func TestQueryResults(t *testing.T) {
	covid := generateData()
	t.Run("Running query 1/3", func(t *testing.T) {
		result := false
		timeFilter, _ := time.Parse("02/01/2006", "28/09/2021")

		covid.Query(&Entry{
			ExposureLocation: "7-Eleven Holt",
			Date:             &timeFilter,
		}, QueryParams{
			PrintRAWCSV: false,
		})

		if len(covid.FilteredResults.Items) > 0 {
			result = true

		}

		if !result {
			t.Fail()
		}
	})

	t.Run("Running query 2/3", func(t *testing.T) {
		result := false
		timeFilter, _ := time.Parse("02/01/2006", "04/10/2021")
		covid.Query(&Entry{
			ExposureLocation: "ALDI Belconnen",
			Date:             &timeFilter,
		}, QueryParams{
			PrintRAWCSV: false,
		})

		if len(covid.FilteredResults.Items) > 0 {
			result = true

		}
		if !result {
			t.Fail()
		}
	})

	t.Run("Running query 3/3", func(t *testing.T) {
		result := false
		timeFilter, _ := time.Parse("02/01/2006", "09/10/2021")
		covid.Query(&Entry{
			ExposureLocation: "Coles Kaleen",
			Date:             &timeFilter,
		}, QueryParams{
			PrintRAWCSV: false,
		})

		if len(covid.FilteredResults.Items) > 0 {
			result = true

		}
		if !result {
			t.Fail()
		}
	})
}

// TestDataLengthStatic will take expected values as static content, and run
// some basic validation directly from an existing data set from the
// authoriative source. The check will validate the length of the row in the
// CSV given addresses/locations can also contain ',', and not have a street
// and/or location. To complicate things, the ',' is our delimiter.
func TestDataLengthStatic(t *testing.T) {
	t.Run("Validating static content constraints (1/3)", func(t *testing.T) {
		var example = ",,\"7-Eleven Holt\",\"88 Hardwick Crescent\",\"Holt\",\"ACT\",\"01/09/2021 - Wednesday\",2:15pm,3:00pm,\"Monitor\""
		if len(strings.Split(example, ",")) != 10 {
			t.Fail()
		}
	})
	t.Run("Validating static content constraints (2/3)", func(t *testing.T) {
		var example = ",,\"ALDI Belconnen\",\"Westfield Belconnen, Benjamin Way\",\"Belconnen\",\"ACT\",\"01/09/2021 - Wednesday\",7:00pm,7:30pm,\"Monitor\""
		if len(strings.Split(example, ",")) != 11 {
			t.Fail()
		}
	})
	t.Run("Validating static content constraints (3/3)", func(t *testing.T) {
		var example = ",,\"Kaleen Plaza Pharmacy\",\"Shop 5, Kaleen Shopping Centre, Georgina Crescent\",\"Kaleen\",\"ACT\",\"01/09/2021 - Wednesday\",6:15pm,7:10pm,\"Casual\""
		if len(strings.Split(example, ",")) != 12 {
			t.Fail()
		}
	})
}

// TestData is an integration test to ensure certain thresholds are as
// expected. Failing these tests would indicate a change in data
// structure which would mean adjustments need to be made.
func TestData(t *testing.T) {
	covid := &x{}
	var err error
	t.Run("Getting Endpoint", func(t *testing.T) {
		err = covid.GetHTML(testEndpoint)
		if err != nil {
			t.Fail()
		}
	})
	t.Run("Getting CSV File URL", func(t *testing.T) {
		err = covid.GetCSVReference()
		if err != nil {
			t.Fail()
		}
	})
	t.Run("Getting CSV File Contents", func(t *testing.T) {
		err = covid.GetCSVData()
		if err != nil {
			t.Fail()
		}
	})
	t.Run("Cleaning CSV content", func(t *testing.T) {
		covid.Clean()
		for _, line := range strings.Split(covid.RawCSV, "\n") {
			if strings.HasPrefix(line, string(rune(13))) {
				t.Fail()
			}
			if strings.HasSuffix(line, string(rune(13))) {
				t.Fail()
			}
			if strings.HasPrefix(line, string(rune(33))) {
				t.Fail()
			}
			if strings.HasSuffix(line, string(rune(33))) {
				t.Fail()
			}
			if strings.HasPrefix(line, string(rune(44))) {
				t.Fail()
			}
			if strings.HasSuffix(line, string(rune(44))) {
				t.Fail()
			}
		}
	})
	t.Run("Translating CSV File to Struct", func(t *testing.T) {
		covid.SetCSVData()
		if len(covid.RawResults.Items) == 0 {
			t.Fail()
		}
	})
	t.Run("Perform a query without filter", func(t *testing.T) {
		covid.Query(&Entry{}, QueryParams{
			PrintRAWCSV: false,
		})
	})
	t.Run("Assert results pass validation criteria", func(t *testing.T) {
		for _, item := range covid.FilteredResults.Items {
			// Is row item nil?
			if fmt.Sprint(&Entry{}) == fmt.Sprint(item) {
				t.Fail()
			}
		}
	})
}
