package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	// rawOutput tells the app to print the raw csv data instead of
	// rendering a table.
	rawOutput = true
	// endpoint is the URL/endpoint which contains the exposure sites.
	// notably, this is only compatible with the Canberra website.
	// other examples using a similar convention would need to be
	// identified to be compatible.
	endpoint string
	// generate will fetch a known copy of the original source dataset.
	// this will be useful for running the application after covid is
	// no longer a thing, because this data won't exist forever.
	generate bool
	// contact is the filter for the contact fiels, notably it will
	// only return results when set to "casual", "monitor" or "close".
	// There is no way to filter for nil value as the filter checks
	// if the result contains the input.
	contact string
	// file provides a csv input which circumvents downloading a new
	// set of data from the endpoint.
	file string
	// limit will limit the results to a specific number.
	limit int
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
	// SampleEndpointURL is a reference to a mirror of an official data
	// file from official sources during the pandemic which will allow
	// this tool to be used against a source, and for tests to be run
	// against a predictable dataset.
	SampleEndpointURL = "https://gist.githubusercontent.com/fubarhouse/a827e4db69590556a3bf795ab1f93c89/raw/b536c5d534734da3c7484d9e1db8fe7ba56d7af5/sample-dataset-covidcheck.md"
	// Slice input for input queries.

	// NegativeQueries include queries to filter out.
	NegativeQueries negativeQueries
	// PositiveQueries include queries to filter in.
	PositiveQueries positiveQueries
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

	// QueryParams are extra settings for Query operation which aren't associated
	// to the Entry values.
	QueryParams struct {
		// PrintRAWCSV is a bool which will instruct the Query operation to print
		// the values, rather than append them to the output list for rendering.
		PrintRAWCSV bool
		// todo move non-entry associated fields & vars into params. (eg width)
	}

	// Entries is a slice of type Entry.
	Entries struct {
		Items []Entry
	}

	// Entry is a stuct which represents the data to be displayed.
	Entry struct {
		//SHA256 			 sha256.sum224 // todo
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
		ArrivalTime *time.Time
		// Arrival time is the exposure finish time represented as a string.
		DepartureTime *time.Time
		// Contact is the contact category - either Close, Casual or Monitor.
		Contact string
	}

	// negativeQueries are the input queries to exclude.
	negativeQueries []string
	// positiveQueries are the input queries to include.
	positiveQueries []string
)

func (i *negativeQueries) String() string {
	return strings.Join(*i, "|")
}

func (i *negativeQueries) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *positiveQueries) String() string {
	return strings.Join(*i, "|")
}

func (i *positiveQueries) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (e *Entries) Len() int {
	return len(e.Items)
}

func (e *Entries) Less(i, j int) bool {
	//	sort.Sort(students)
	//	fmt.Println(sort.IsSorted(students))
	//	sort.Sort(sort.Reverse(students))
	// https://gist.github.com/dnutiu/a899e48c95ff80fe98bada566e03251e

	// Work out if the full start date comes before another

	return e.Items[i].Date.After(*e.Items[j].Date)
}

func (e *Entries) Swap(i, j int) {
	e.Items[i], e.Items[j] = e.Items[j], e.Items[i]
}

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
		return strings.Trim(strings.Split(in, "\"")[1], " ")
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
	case string:
		// Note: time is also handled via string.
		if strings.Contains(strings.ToLower(b.(string)), strings.ToLower(a.(string))) {
			found = true
		}
		if c, _ := regexp.Match(strings.ToLower(a.(string)), []byte(strings.ToLower(b.(string)))); c {
			found = true
		}
		// nil checks for strings.
		if strings.ToLower(a.(string)) == "nil" && strings.ToLower(b.(string)) == "" {
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

// checkNot will provide field validation, and will add the result to a
// *MultiQueries if the validation passes. This will later be checked
// before being added to the filtered results in Query.
func checkNot(a, b interface{}, mq *MultiQueries) bool {
	found := true
	if a == nil {
		return false
	}
	switch v := a.(type) {
	case string:
		// Note: time is also handled via string.
		if !strings.Contains(strings.ToLower(b.(string)), strings.ToLower(a.(string))) {
			found = false
		}
		if c, _ := regexp.Match(strings.ToLower(a.(string)), []byte(strings.ToLower(b.(string)))); !c {
			found = false
		}
		// nil checks for strings.
		if strings.ToLower(a.(string)) == "nil" && strings.ToLower(b.(string)) == "" {
			found = false
		}
	default:
		fmt.Printf("no handler for %v was found\n", v)
	}

	if !found {
		mq.Items = append(mq.Items, true)
		return false
	}

	mq.Items = append(mq.Items, false)
	return true
}

// Query will clear out the FilteredResults field and repopulate it by querying
// each result against the input Entry object.
func (x *x) Query(e *Entry, params QueryParams) {
	if fmt.Sprint(*e) == fmt.Sprint(x.Filter) {
		return
	}
	x.Filter = *e
	x.FilteredResults = Entries{}
	for _, dataEntry := range x.RawResults.Items {

		mq := MultiQueries{}
		match := true

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
		if e.Date != nil && fmt.Sprint(e.Date) != "1-1-1" {
			dateOne := fmt.Sprintf("%d-%d-%d", e.Date.Day(), e.Date.Month(), e.Date.Year())
			dateTwo := fmt.Sprintf("%d-%d-%d", dataEntry.Date.Day(), dataEntry.Date.Month(), dataEntry.Date.Year())
			if dateOne != "1-1-1" {
				if b := check(dateOne, dateTwo, &mq); b {
					match = true
				}
			}
		}
		if e.ArrivalTime != nil {
			if b := check(e.ArrivalTime, dataEntry.ArrivalTime, &mq); b {
				match = true
			}
		}
		if e.DepartureTime != nil {
			if b := check(e.DepartureTime, dataEntry.DepartureTime, &mq); b {
				match = true
			}
		}
		if e.Contact != "" {
			if b := check(e.Contact, dataEntry.Contact, &mq); b {
				match = true
			}
		}

		if len(PositiveQueries) != 0 {
			for _, q := range PositiveQueries {
				if b := check(q, fmt.Sprint(dataEntry), &mq); b {
					match = true
				} else {
					match = false
				}
			}
		}

		if len(NegativeQueries) != 0 {
			for _, q := range NegativeQueries {
				if b := checkNot(q, fmt.Sprint(dataEntry), &mq); !b {
					match = true
				} else {
					match = false
				}
			}
		}

		for _, v := range mq.Items {
			if !v {
				match = false
			}
		}

		if match && !params.PrintRAWCSV {
			x.FilteredResults.Items = append(x.FilteredResults.Items, dataEntry)
		}

		if match && params.PrintRAWCSV {

			fmt.Printf("\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\",\"%v\"\n", dataEntry.Status, dataEntry.ExposureLocation, dataEntry.Street, dataEntry.Suburb, dataEntry.State, fmt.Sprintf("%02d/%v/%v - %v", dataEntry.Date.Day(), int(dataEntry.Date.Month()), dataEntry.Date.Year(), dataEntry.Date.Weekday()), dataEntry.ArrivalTime.Format(time.Kitchen), dataEntry.DepartureTime.Format(time.Kitchen), dataEntry.Contact)
		}
	}
}

// GetCSVData will grabx the CSV data file and set the RawCSV
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

	if len(components) < 9 {
		return *newEntry
	}

	// location, street are less predictable...

	// In order to display the information correctly, we're going to do some
	// trickery with the input fields, which components will have a length of 10, 11 or 12
	// depending on the edge-case. We should probably make this easier later...
	date := time.Now()
	Status := ""
	Contact := ""
	State := ""
	TimeStart := &time.Time{}
	TimeEnd := &time.Time{}
	Suburb := ""
	Street := ""
	Location := ""
	for i, v := range components {
		{
			// Dynamic discovery of Date
			datestring := strings.Split(trimQuotes(components[i]), " ")[0]
			re, err := regexp.Compile(`^.*[0-9]+\/[0-9]+\/[0-9][0-9]+.*$`)
			if err != nil {
				fmt.Println(err.Error())
			}
			if re.MatchString(datestring) {
				t, err := time.Parse("2/1/2006", strings.Trim(datestring, " "))
				if err == nil {
					date = t
				}
			}
		}

		fieldData := trimQuotes(v)

		{
			// Dynamic discovery of Status
			re, err := regexp.Compile("^(New||Updated||Archived)$")
			if err != nil {
				fmt.Println(err.Error())
			}
			if re.MatchString(fieldData) {
				if Status == "" {
					Status = fieldData
					continue
				}
			}
		}

		{
			// Dynamic discovery of Contact
			re, err := regexp.Compile("^(Close||Casual||Monitor)$")
			if err != nil {
				fmt.Println(err.Error())
			}
			if re.MatchString(fieldData) {
				if Contact == "" {
					Contact = fieldData
					continue
				}
			}
		}

		{
			re, err := regexp.Compile("^(ACT||NSW||VIC||TAS||SA||WA||NT||QLD)$")
			if err != nil {
				fmt.Println(err.Error())
			}
			if re.MatchString(fieldData) {
				if State == "" {
					State = fieldData
					continue
				}
			}
		}

		{
			re, err := regexp.Compile("^[A-Z][a-z]+$")
			if err != nil {
				fmt.Println(err.Error())
			}
			if re.MatchString(fieldData) {
				if Suburb == "" {
					Suburb = fieldData
					continue
				}
			} else if fieldData == "Public Transport" {
				Suburb = fieldData
				continue
			}
		}

		{
			re, err := regexp.Compile("^[0-9]+(:)[0-9]+(am||pm||AM||PM)$")
			if err != nil {
				fmt.Println(err.Error())
			}
			if re.MatchString(fieldData) {
				// Start Time is expected to precede End Time directly, so we make sure they're
				// paired up to identify the pair of values.

				fieldData = strings.Replace(fieldData, "am", "AM", -1)
				fieldData = strings.Replace(fieldData, "pm", "PM", -1)
				timeOne, eOne := time.Parse(time.Kitchen, fieldData)

				adjacentFieldData := trimQuotes(components[i+1])
				adjacentFieldData = strings.Replace(adjacentFieldData, "am", "AM", -1)
				adjacentFieldData = strings.Replace(adjacentFieldData, "pm", "PM", -1)
				timeTwo, eTwo := time.Parse(time.Kitchen, adjacentFieldData)

				if eOne == nil && eTwo == nil {
					TimeStart = &timeOne
					TimeEnd = &timeTwo
				}
			}
		}

		{
			re, err := regexp.Compile("^([A-Z]||[0-9]).*[a-z].*$")
			if err != nil {
				fmt.Println(err.Error())
			}
			if re.MatchString(fieldData) {
				if Location == "" {
					Location = fieldData
				}
			}
		}

		{
			re, err := regexp.Compile(`^([0-9-\/]+\ [A-Z][a-z].*||[A-Z][a-z].*)$`)
			if err != nil {
				fmt.Println(err.Error())
			}
			if re.MatchString(fieldData) {
				if Street == "" {
					Street = fieldData
				}
			}
		}
	}

	newEntry = &Entry{
		Status:           Status,
		ExposureLocation: Location,
		Street:           Street,
		Suburb:           Suburb,
		State:            State,
		Date:             &date,
		ArrivalTime:      TimeStart,
		DepartureTime:    TimeEnd,
		Contact:          Contact,
	}

	return *newEntry
}

// SetCSVData will populate the RawResultsww field with the inputs after
// processing the RawCSV data into the expected format (type Entry)
func (x *x) SetCSVData() {
	for _, dataEntry := range strings.Split(x.RawCSV, "\n") {
		newEntry := fieldTranslate(&dataEntry)
		x.AddRaw(&newEntry)
		x.AddFiltered(&newEntry)
	}

	// Sorting is implemented but not working.
	sort.Sort(&x.FilteredResults)
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
	table.SetHeader([]string{"Status", "Location", "Street", "Suburb", "State", "Date/Time", "Contact"})
	table.SetCaption(false, "COVID-19 Exposure Sites")
	table.SetColWidth(width)

	for i, item := range x.FilteredResults.Items {

		d := fmt.Sprintf("%d-%d-%d", item.Date.Day(), item.Date.Month(), item.Date.Year())

		s := []string{
			item.Status,
			item.ExposureLocation,
			item.Street,
			item.Suburb,
			item.State,
			fmt.Sprintf("%v %v - %v", d, item.ArrivalTime.Format(time.Kitchen), item.DepartureTime.Format(time.Kitchen)),
			item.Contact,
		}

		if limit != 0 && i < limit {
			table.Append(s)
		} else if limit == 0 {
			table.Append(s)
		}
	}

	if !rawOutput && len(x.FilteredResults.Items) == 0 {
		fmt.Println("no results found")
		return
	} else if rawOutput {
		return
	}

	table.Render()

}

// Clean will filter garbage in raw CSV data.
func (x *x) Clean() {
	var cleaned string

	for _, line := range strings.Split(x.RawCSV, "\n") {
		if len(strings.Split(line, ",")) > 9 {

			// I don't even know how this garbage ended up here...

			line = strings.Replace(line, "\n", "", 1)
			line = strings.Trim(line, string(rune(13)))
			line = strings.Trim(line, string(rune(33)))
			line = strings.Trim(line, string(rune(44)))

			cleaned = cleaned + fmt.Sprintf("%v\n", line)
		}
	}

	if len(cleaned) != 0 {
		x.RawCSV = cleaned
	}
}

func generateData() *x {
	c := &x{}
	c.DataEndpoint = SampleEndpointURL
	e := c.GetCSVData()
	if e != nil {
		fmt.Println(e.Error())
	}
	if e != nil {
		fmt.Println(e.Error())
	}
	c.SetCSVData()
	return c
}

// main is main, our programs starting point.
func main() {

	// flags

	flag.StringVar(&file, "file", "", "relative path to csv file to use instead of new data.")
	flag.IntVar(&limit, "limit", 0, "Limit how many results are shown.")

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
	flag.Var(&PositiveQueries, "query", "arbitrary query")
	flag.Var(&NegativeQueries, "query-not", "arbitrary query reversed (not)")
	flag.Var(&PositiveQueries, "q", "arbitrary query")
	flag.Var(&NegativeQueries, "qn", "arbitrary query reversed (not)")
	flag.BoolVar(&rawOutput, "raw", false, "display output as csv")
	flag.IntVar(&width, "width", 50, "width of table columns")

	flag.BoolVar(&rawOutput, "generate", false, "download a mirror of a source dataset to stdout")

	flag.Parse()

	if generate {
		c := generateData()
		fmt.Println(c.RawCSV)
		os.Exit(0)
	}

	covid := &x{}

	if file == "" {
		e := covid.GetHTML(endpoint)
		if e != nil {
			fmt.Println(e.Error())
		}
		e = covid.GetCSVReference()
		if e != nil {
			fmt.Println(e.Error())
		}
		e = covid.GetCSVData()
		if e != nil {
			fmt.Println(e.Error())
		}
	} else {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			panic("could not read file")
		}
		covid.RawCSV = string(content)
	}

	covid.Clean()
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
		Status:           status,
		ExposureLocation: location,
		Street:           street,
		Suburb:           suburb,
		State:            state,
		Date:             t,
		//ArrivalTime:      atime,
		//DepartureTime:    dtime,
		Contact: contact,
	}, QueryParams{
		PrintRAWCSV: rawOutput,
	})

	// Render!
	covid.Render()
	if !rawOutput && limit == 0 && len(covid.FilteredResults.Items) > 0 {
		fmt.Printf("total items found: %d\n", len(covid.FilteredResults.Items))
	}
	if !rawOutput && limit != 0 && len(covid.FilteredResults.Items) > 0 {
		count := limit
		if count > len(covid.FilteredResults.Items) {
			count = len(covid.FilteredResults.Items)
		}
		fmt.Printf("displaying %d of %d total items found\n", count, len(covid.FilteredResults.Items))
	}
}
