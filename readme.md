# Canberra Covid Check

A very simple, no obligation terminal app that will search and display data
directly from Canberra's official COVID-19 exposure locations website.

This was built purely for selfish needs - I was sick of going to the website
to check information regularly. With this tool and Twitter notifications, I
can just go to the terminal and request the exact piece of information I need
and if needed, cross-reference with my check-in locations/times/dates.

It's actually pretty sweet, which is why you can use this. If this sounds 
useful to you, then please give it a try.

## Installation

I don't intend to ship this on the AUR or any other distribution platform. 
Instead, you can install or build from source using the `go` toolchain:
```shell
go install github.com/fubarhouse/covid-check@latest
```

## Usage

```shell
covid-check [flags]
```

### Flags

| Name       | Example                  | Description |
|------------|--------------------------|---------|
| Contact    | `-contact new`           | search string for contact field                                   |
| Date       | `-date 01/07/2021`       | search string for date field - must be in the format `DD/MM/YYYY` |
| Start Time | `-start-time 9:00am`     | search string for arrival time - represented as a string          |
| End Time   | `-end-time 5:00pm`       | search string for departure time - represented as a string        |
| Endpoint   | `-endpoint https://...`  | url of ACT government website page with data to scrape       |
| Location   | `-location Coles`        | search string of location field                                   |
| State      | `-state ACT`             | search string of state field                                      |
| Status     | `-status new`            | search string of status field                                     |
| Street     | `-street Hibberson`      | search string of street field                                     |
| Suburb     | `-suburb woden`          | search string of suburb field                                     |
| Width      | `-width 50`              | with of table columns, change to make the table wider             |

## License

MIT - no obligations or warranties are provided with this application.