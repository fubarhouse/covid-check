package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/olekukonko/tablewriter"
)

var (
	// fieldCount is a debugging variable which will filter results
	// by the count of the fields in the row of the RawCSV line item.
	fieldCount int
	// endpoint is the URL/endpoint which contains the exposure sites.
	// notably, this is only compatible with the Canberra website.
	// other examples using a similar convention would need to be
	// identified to be compatible.
	endpoint string
	// contact is the filter for the contact fiels, notably it will
	// only return results when set to "casual", "monitor" or "close".
	// There is no way to filter for nil value as the filter checks
	// if the result contains the input.
	contact string
	// location is the filter for the location field, and will check
	// if the result contains the input information.
	location string
	// suburb is the filter for the suburb field, and will check
	// if the result contains the input information.
	suburb string
	// status is the filter for the status field, and will check
	// if the result contains the input information. Results will
	// only be returned for "archived", "updated" or "new".
	status string
	// street is the filter for the street field, and will check
	// if the result contains the input information.
	street string
	// state is the filter for the state field, and will check
	// if the result contains the input information. Results will
	// only be returned if the value is not set, or set to "ACT".
	state string
	// udate is the filter for the time field, and will check
	// if the result contains the input information. You will need
	// to set this to something in the format of 01/02/2006 for
	// this to actually work - failing this the application will panic
	// unless it is not set.
	udate string
	// atime is the filter for the arrival time field, and will check
	// if the result contains the input information. This is treated
	// strictly as a string at this time.
	atime string
	// dtime is the filter for the finish time field, and will check
	// if the result contains the input information. This is treated
	//	// strictly as a string at this time.
	dtime string
	// width is the width of the table column, should you be so inclined.
	width int
	// query is an arbitrary, non-specific query
	query string
)

type (

	// MultiQuery is a bool slice which filtered results must validate against.
	MultiQuery []bool

	// MultiQueries is a struct with a MultiQuery to store filter results for
	// an individual Entry. It is intended that a successful filter will have
	// all items in Items value as true, otherwise the item will be omitted
	// from the final result.
	MultiQueries struct {
		Items MultiQuery
	}

	// Entries is a slice of type Entry.
	Entries struct {
		Items []Entry
	}

	// Entry is a stuct which represents the data to be displayed.
	Entry struct {
		//SHA256 			 sha256.sum224 // todo
		// FieldCount is the amount of fields in the row of the raw CSV Entry
		FieldCount int
		// Status is the status of the Entry - either New, Updated, Archived,
		// or without a value - nil.
		Status string
		// Location is the location as provided by the data.
		ExposureLocation string
		// Street is supposed to be the street address - the data
		// is a little inconsistent - we've tried to fix that.
		Street string
		// Suburb is the suburb of the Entry.
		Suburb string
		// State is the state of the Entry - can only be "ACT" or nil.
		State string
		// Date is a valid *time.Time entry used for querying or presenting.
		Date *time.Time
		// Arrival time is the exposure start time represented as a string.
		ArrivalTime string
		// Arrival time is the exposure finish time represented as a string.
		DepartureTime string
		// Contact is the contact category - either Close, Casual or Monitor.
		Contact string
	}
)

// Add will add an Entry into the Entries - can be applied to RawResults
// or RawFilteredResults, depending on where in the application.
func (entries *Entries) Add(entry Entry) {
	entries.Items = append(entries.Items, entry)
}

// trimQuotes will simply check if the input is wrapped in double quotes
// and stip them, and return the contents. It will trim the beginning and
// end, but not in the middle. It will return the second item (index item 1)
// of the slice after splitting it. If no quotes are found, the input is
// return unaltered.
func trimQuotes(in string) (out string) {
	if strings.Contains(in, "\"") {
		return strings.Split(in, "\"")[1]
	}
	return in
}

// x is a client for our API which contains all of the functionality
// we need to put data into the system and display it to the user.
type x struct {
	// DataEndPoint is the endpoint of the input CSV file to scrape and process
	DataEndpoint string
	// RawCSV is the raw CSV data represented as a string.
	RawCSV string
	// RawHTML is the raw HTML of the web page endpoint represented as a string
	RawHTML string
	// RawResults is the unchanged, processed input from the CSV file.
	RawResults Entries
	// FilteredResults is the Entries object of all values matching input queries.
	// If no input queries are provided, this objeect will match the length of
	// RawResults.
	FilteredResults Entries
	// Filter is a single input Entry which is used to query against the results
	// in order to filter the list of results to the end users preference.
	Filter Entry
}

// GetHTML will retrieve the HTML endpoint and add it to the RawHTML field.
func (x *x) GetHTML(endpoint string) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("failed to fetch data: %d %s", resp.StatusCode, resp.Status)
	}

	rawHTML, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	x.RawHTML = string(rawHTML)
	return nil
}

// GetCSVReference will try to grab the URL path of the CSV to process.
// This is highly opinionated but could be manipulated with an interface.
func (x *x) GetCSVReference() error {

	reader := bytes.NewReader([]byte(x.RawHTML))
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return err
	}
	html, _ := doc.Html()
	htmlData := strings.Split(html, "\n")
	for _, line := range htmlData {
		if strings.Contains(line, "Papa.parse(") {
			component := strings.Split(line, "\"")[1]
			if strings.HasSuffix(component, ".csv") {
				x.DataEndpoint = component
				return nil
			}
		}
	}
	return nil
}

// check will provide field validation, and will add the result to a
// *MultiQueries if the validation passes. This will later be checked
// before being added to the filtered results in Query.
func check(a, b interface{}, mq *MultiQueries) bool {
	found := false
	if a == nil {
		return false
	}
	switch v := a.(type) {
	case int:
		// Here, our only option is FieldCount, so comparing against full value is best.
		if a.(int) == b.(int) {
			found = true
		}
	case string:
		// Note: time is also handled via string.
		if strings.Contains(strings.ToLower(b.(string)), strings.ToLower(a.(string))) {
			found = true
		}
		if c, _ := regexp.Match(strings.ToLower(a.(string)), []byte(strings.ToLower(b.(string)))); c {
			found = true
		}
		// If the dates are queried, we check for absolute equality.
		if a == b {
			found = true
		}
	default:
		fmt.Printf("no handler for %v was found\n", v)
	}

	if found {
		mq.Items = append(mq.Items, true)
		return true
	}

	mq.Items = append(mq.Items, false)
	return false
}

// Query will clear out the FilteredResults field and repopulate it by querying
// each result against the input Entry object.
func (x *x) Query(e *Entry) {
	if fmt.Sprint(*e) == fmt.Sprint(x.Filter) {
		return
	}
	x.Filter = *e
	x.FilteredResults = Entries{}
	for _, dataEntry := range x.RawResults.Items {

		mq := MultiQueries{}
		match := true

		if e.FieldCount != 0 {
			if b := check(e.FieldCount, dataEntry.FieldCount, &mq); b {
				match = true
			}
		}
		if e.Status != "" {
			if b := check(e.Status, dataEntry.Status, &mq); b {
				match = true
			}
		}
		if e.ExposureLocation != "" {
			if b := check(e.ExposureLocation, dataEntry.ExposureLocation, &mq); b {
				match = true
			}
		}
		if e.Street != "" {
			if b := check(e.Street, dataEntry.Street, &mq); b {
				match = true
			}
		}
		if e.Suburb != "" {
			if b := check(e.Suburb, dataEntry.Suburb, &mq); b {
				match = true
			}
		}
		if e.State != "" {
			if b := check(e.State, dataEntry.State, &mq); b {
				match = true
			}
		}
		dateOne := fmt.Sprintf("%d-%d-%d", e.Date.Day(), e.Date.Month(), e.Date.Year())
		dateTwo := fmt.Sprintf("%d-%d-%d", dataEntry.Date.Day(), dataEntry.Date.Month(), dataEntry.Date.Year())
		if dateOne != "1-1-1" {
			if b := check(dateOne, dateTwo, &mq); b {
				match = true
			}
		}
		if e.ArrivalTime != "" {
			if b := check(e.ArrivalTime, dataEntry.ArrivalTime, &mq); b {
				match = true
			}
		}
		if e.DepartureTime != "" {
			if b := check(e.DepartureTime, dataEntry.DepartureTime, &mq); b {
				match = true
			}
		}
		if e.Contact != "" {
			if b := check(e.Contact, dataEntry.Contact, &mq); b {
				match = true
			}
		}
		if query != "" {
			if b := check(query, fmt.Sprint(dataEntry), &mq); b {
				match = true
			}
		}

		for _, v := range mq.Items {
			if !v {
				match = false
			}
		}

		if match {
			x.FilteredResults.Items = append(x.FilteredResults.Items, dataEntry)
		}
	}
}

// GetCSVData will grabx.FilteredResults.Items = append(x.FilteredResults.Items, dataEntry) the CSV data file and process set the RawCSV
// field to the contents of that file.
func (x *x) GetCSVData() error {
	resp, err := http.Get(x.DataEndpoint)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("failed to fetch data: %d %s", resp.StatusCode, resp.Status)
	}

	RawCSV, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	x.RawCSV = string(RawCSV)
	return nil
}

// fieldTranslate will ensure the Entry is processed and displayed correctly,
// as structural changes will impact this. Daily so far the tool has broken
// because of some of the logic, so here we find a better way.
func fieldTranslate(e *string) Entry {

	components := strings.Split(*e, ",")
	newEntry := &Entry{}

	// location, street are less predictable...

	// In order to display the information correctly, we're going to do some
	// trickery with the input fields, which components will have a length of 10, 11 or 12
	// depending on the edge-case. We should probably make this easier later...

	datestring := strings.Split(trimQuotes(components[len(components)-4]), " ")[0]
	t, _ := time.Parse("02/01/2006", datestring)

	newEntry = &Entry{
		Status:        trimQuotes(components[1]),
		Suburb:        trimQuotes(components[len(components)-6]),
		State:         trimQuotes(components[len(components)-5]),
		Date:          &t,
		ArrivalTime:   trimQuotes(components[len(components)-3]),
		DepartureTime: trimQuotes(components[len(components)-2]),
		Contact:       trimQuotes(components[len(components)-1]),
	}

	newEntry.FieldCount = len(components)

	// Handle edge-cases in data here:
	switch newEntry.FieldCount {
	case 10:
		newEntry.ExposureLocation = trimQuotes(components[2])
		newEntry.Street = trimQuotes(components[3])
	case 11:
		var location string
		if trimQuotes(components[2]) != "" {
			location = fmt.Sprintf("%s", trimQuotes(components[2]))
		}
		if trimQuotes(components[3]) != "" {
			location = fmt.Sprintf("%s, %s", location, trimQuotes(components[3]))
		}
		newEntry.ExposureLocation = location
		newEntry.Street = ""
	case 12:
		newEntry.ExposureLocation = fmt.Sprintf("%s, %s, %s", trimQuotes(components[3]), trimQuotes(components[2]), trimQuotes(components[4]))
		newEntry.Street = ""
	}

	return *newEntry

}

// SetCSVData will populate the RawResults field with the inputs after
// processing the RawCSV data into the expected format (type Entry)
func (x *x) SetCSVData() {
	for _, dataEntry := range strings.Split(x.RawCSV, "\n") {
		newEntry := fieldTranslate(&dataEntry)
		x.AddRaw(&newEntry)
		x.AddFiltered(&newEntry)
	}

	// todo: sort by date
	//for i, dataEntry := range x.FilteredResults.Items {
	//	sort.Slice(x.FilteredResults.Items, func(i, j int) bool {
	//		return x.FilteredResults.Items[i].Date > products[j].Price
	//	})
	//}
}

// AddFiltered will check if the input has a suburb associated to it and
// adds the result to the FilteredResults slice for rendering.
func (x *x) AddFiltered(e *Entry) {
	if e.Suburb == "" {
		return
	}
	x.FilteredResults.Items = append(x.FilteredResults.Items, *e)
}

// AddRaw will check if the input has a suburb associated to it and
// adds the result to the FilteredResults slice for rendering.
func (x *x) AddRaw(e *Entry) {
	if e.Suburb == "" {
		return
	}
	x.RawResults.Items = append(x.RawResults.Items, *e)
}

// Render will render the table displaying the data to the user.
func (x *x) Render() {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"", "Status", "Location", "Street", "Suburb", "State", "Date", "Start Time", "Finish Time", "Contact"})
	table.SetCaption(false, "COVID-19 Exposure Sites")
	table.SetColWidth(width)

	for _, item := range x.FilteredResults.Items {

		s := []string{
			fmt.Sprintf("%d", item.FieldCount),
			item.Status,
			item.ExposureLocation,
			item.Street,
			item.Suburb,
			item.State,
			fmt.Sprintf("%d-%d-%d", item.Date.Day(), item.Date.Month(), item.Date.Year()),
			item.ArrivalTime,
			item.DepartureTime,
			item.Contact,
		}

		table.Append(s)
	}

	if len(x.FilteredResults.Items) == 0 {
		fmt.Println("no results found")
		return
	}

	table.Render()

}

// main is main, our programs starting point.
func main() {

	// flags
	flag.StringVar(&endpoint, "endpoint", "https://www.covid19.act.gov.au/act-status-and-response/act-covid-19-exposure-locations", "endpoint of Canberra's covid exposure list")
	flag.StringVar(&contact, "contact", "", "contact rating [|close|casual|monitor]")
	flag.StringVar(&location, "location", "", "location")
	flag.StringVar(&suburb, "suburb", "", "suburb")
	flag.StringVar(&status, "status", "", "status rating [|new|archived|updated]")
	flag.StringVar(&street, "street", "", "street")
	flag.StringVar(&state, "state", "", "state")
	flag.StringVar(&udate, "date", "", "date (formatted strictly as DD/MM/YYYY)")
	flag.StringVar(&atime, "start-time", "", "start time")
	flag.StringVar(&dtime, "end-time", "", "end time")
	flag.StringVar(&query, "query", "", "arbitrary query")
	flag.IntVar(&width, "width", 50, "width of table columns")
	flag.IntVar(&fieldCount, "field-count", 0, "count of fields in row")
	flag.Parse()

	covid := &x{}
	covid.GetHTML(endpoint)
	covid.GetCSVReference()
	covid.GetCSVData()
	covid.SetCSVData()

	// validate input date requirements
	t := &time.Time{}
	if udate != "" {
		tparse, err := time.Parse("02/01/2006", udate)
		if err != nil {
			fmt.Printf("date format is strictly DD/MM/YYYY: could not parse '%s'\n", udate)
			panic(err.Error())
		}
		t = &tparse
	}

	covid.Query(&Entry{
		FieldCount:       fieldCount,
		Status:           status,
		ExposureLocation: location,
		Street:           street,
		Suburb:           suburb,
		State:            state,
		Date:             t,
		ArrivalTime:      atime,
		DepartureTime:    dtime,
		Contact:          contact,
	})

	// Render!
	covid.Render()
	if len(covid.FilteredResults.Items) > 0 {
		fmt.Printf("total items found: %d\n", len(covid.FilteredResults.Items))
	}

}
