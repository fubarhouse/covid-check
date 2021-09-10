package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
)

type (
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
	defer resp.Body.Close()

	if err != nil {
		return err
	}

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

// Query will clear out the FilteredResults field and repopulate it by querying
// each result against the input Entry object.
func (x *x) Query(e *Entry) {
	if fmt.Sprint(*e) == fmt.Sprint(x.Filter) {
		return
	}
	x.Filter = *e
	x.FilteredResults = Entries{}
	newResults := []Entry{}
	for _, dataEntry := range x.RawResults.Items {

		// TODO this needs an elegant solution.
		// TODO compare against lowercase.
		if strings.Contains(dataEntry.Suburb, e.Suburb) && strings.Contains(dataEntry.ExposureLocation, e.ExposureLocation) {
			if strings.Contains(dataEntry.Contact, e.Contact) {
				if strings.Contains(dataEntry.Status, e.Status) {
					if strings.Contains(dataEntry.State, e.State) {
						if strings.Contains(dataEntry.Street, e.Street) {
							if strings.Contains(dataEntry.State, e.State) {
								if strings.Contains(dataEntry.ArrivalTime, e.ArrivalTime) {
									if strings.Contains(dataEntry.DepartureTime, e.DepartureTime) {
										dateOne := fmt.Sprintf("%d/%d/%d", dataEntry.Date.Day(), dataEntry.Date.Month(), dataEntry.Date.Year())
										dateTwo := fmt.Sprintf("%d/%d/%d", e.Date.Day(), e.Date.Month(), e.Date.Year())
										if dateTwo != "1/1/1" {
											if strings.Contains(dateOne, dateTwo) {
												if dataEntry.FieldCount == e.FieldCount {
													newResults = append(newResults, dataEntry)
												}
												if e.FieldCount == 0 {
													newResults = append(newResults, dataEntry)
												}
											}
										} else {
											if dataEntry.FieldCount == e.FieldCount {
												newResults = append(newResults, dataEntry)
											}
											if e.FieldCount == 0 {
												newResults = append(newResults, dataEntry)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
		x.FilteredResults.Items = newResults
	}
}

// GetCSVData will grab the CSV data file and process set the RawCSV
// field to the contents of that file.
func (x *x) GetCSVData() error {
	resp, err := http.Get(x.DataEndpoint)
	defer resp.Body.Close()

	if err != nil {
		return err
	}

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

// SetCSVData will populate the RawResults field with the inputs after
// processing the RawCSV data into the expected format (type Entry)
func (x *x) SetCSVData() {
	for _, dataEntry := range strings.Split(x.RawCSV, "\n") {
		components := strings.Split(dataEntry, ",")
		newEntry := &Entry{}

		// In order to display the information correctly, we're going to do some
		// trickery with the input fields, which components will have a length of 10, 11 or 12
		// depending on the edge-case. We should probably make this easier later...

		switch len(components) {
		case 12:
			datestring := strings.Split(trimQuotes(components[8]), " ")[0]
			t, _ := time.Parse("02/01/2006", datestring)
			newEntry = &Entry{
				Status:           trimQuotes(components[1]),
				ExposureLocation: fmt.Sprintf("%s, %s", trimQuotes(components[2]), trimQuotes(components[3])),
				Street:           trimQuotes(components[5]),
				Suburb:           trimQuotes(components[6]),
				State:            trimQuotes(components[7]),
				Date:             &t,
				ArrivalTime:      trimQuotes(components[9]),
				DepartureTime:    trimQuotes(components[10]),
				Contact:          trimQuotes(components[11]),
			}
		case 11:
			datestring := strings.Split(trimQuotes(components[7]), " ")[0]
			t, _ := time.Parse("02/01/2006", datestring)
			newEntry = &Entry{
				Status:           trimQuotes(components[1]),
				ExposureLocation: trimQuotes(components[2]),
				Street:           "",
				Suburb:           trimQuotes(components[5]),
				State:            trimQuotes(components[6]),
				Date:             &t,
				ArrivalTime:      trimQuotes(components[8]),
				DepartureTime:    trimQuotes(components[9]),
				Contact:          trimQuotes(components[10]),
			}
		case 10:
			datestring := strings.Split(trimQuotes(components[6]), " ")[0]
			t, _ := time.Parse("02/01/2006", datestring)
			newEntry = &Entry{
				Status:           trimQuotes(components[1]),
				ExposureLocation: trimQuotes(components[2]),
				Street:           trimQuotes(components[3]),
				Suburb:           trimQuotes(components[4]),
				State:            trimQuotes(components[5]),
				Date:             &t,
				ArrivalTime:      trimQuotes(components[7]),
				DepartureTime:    trimQuotes(components[8]),
				Contact:          trimQuotes(components[9]),
			}
		}

		newEntry.FieldCount = len(components)
		x.AddRaw(newEntry)
		x.AddFiltered(newEntry)
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
	flag.IntVar(&width, "width", 50, "width of table columns")
	flag.IntVar(&fieldCount, "field-count", 0, "count of fields in row")
	flag.Parse()

	// validate input
	contact = strings.Title(contact)
	location = strings.Title(location)
	suburb = strings.Title(suburb)
	status = strings.Title(status)
	state = strings.ToUpper(state)
	// ignore street, date & time fields

	// debugging note
	// Kaleen has an example of a location w/ a ',' char. (address)

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
