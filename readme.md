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

| Name        | Example                 | Description                                                                                   |
|-------------|-------------------------|-----------------------------------------------------------------------------------------------|
| Contact     | `-contact new`          | search string for contact field                                                               |
| Date        | `-date 01/07/2021`      | search string for date field - must be in the format `DD/MM/YYYY`                             |
| End Time    | `-end-time 5:00pm`      | search string for departure time - represented as a string                                    |
| Endpoint    | `-endpoint https://...` | url of ACT government website page with data to scrape                                        |
| File        | `-file data.csv`        | Provide a file as a data source                                                               |
| Generate    | `-generate`             | Download an official dataset from a mirror and print to stdout                                |
| Limit       | `-limit`                | Specify a maximum quantity of items to show.                                                  |
| Location    | `-location Coles`       | search string of location field                                                               |
| Query       | `--query phillip`       | An arbitrary query - find anything matching input (including regex & multiple values)         |
| Query       | `--query-not phillip`   | An arbitrary query - exclude find anything matching input (including regex & multiple values) |
| Query       | `--q phillip`           | An arbitrary query - find anything matching input (including regex)                           |
| Raw         | `-raw`                  | Performs all search functionality but displays as csv output.                                 |
| Start Time  | `-start-time 9:00am`    | search string for arrival time - represented as a string                                      |
| State       | `-state ACT`            | search string of state field                                                                  |
| Status      | `-status new`           | search string of status field                                                                 |
| Street      | `-street Hibberson`     | search string of street field                                                                 |
| Suburb      | `-suburb woden`         | search string of suburb field                                                                 |
| Width       | `-width 50`             | with of table columns, change to make the table wider                                         |

### Example(s)

```shell
$ covid-check -raw > 20211019.csv
$ covid-check -file 20211019.csv -suburb woden
+----------+-------------------+--------+--------+-------+---------------------------+---------+
|  STATUS  |     LOCATION      | STREET | SUBURB | STATE |         DATE/TIME         | CONTACT |
+----------+-------------------+--------+--------+-------+---------------------------+---------+
| Archived | Boost Juice Woden |        | Woden  | ACT   | 6-10-2021 2:35PM - 3:15PM | Monitor |
+----------+-------------------+--------+--------+-------+---------------------------+---------+
total items found: 1

$ covid-check -file 20211019.csv -q belconnen -location woolworths -contact casual -limit 3
+----------+---------------------------------+---------------------+-----------+-------+----------------------------+---------+
|  STATUS  |            LOCATION             |       STREET        |  SUBURB   | STATE |         DATE/TIME          | CONTACT |
+----------+---------------------------------+---------------------+-----------+-------+----------------------------+---------+
|          | Woolworths Belconnen            | Westfield Belconnen | Belconnen | ACT   | 4-10-2021 3:00PM - 4:00PM  | Casual  |
|          | Woolworths Metro Cameron Avenue | 1/6 Grazier Lane    | Belconnen | ACT   | 5-10-2021 7:00AM - 3:15PM  | Casual  |
|          | Woolworths Metro Cameron Avenue | 1/6 Grazier Lane    | Belconnen | ACT   | 6-10-2021 7:00AM - 8:10AM  | Casual  |
+----------+---------------------------------+---------------------+-----------+-------+----------------------------+---------+
displaying 3 of 9 total items found

$ covid-check -file 20211019.csv -location westfield
+----------+---------------------+---------------------+-----------+-------+-----------------------------+---------+
|  STATUS  |      LOCATION       |       STREET        |  SUBURB   | STATE |          DATE/TIME          | CONTACT |
+----------+---------------------+---------------------+-----------+-------+-----------------------------+---------+
| Archived | Westfield Belconnen | Westfield Belconnen | Belconnen | ACT   | 6-10-2021 10:40AM - 11:25AM | Monitor |
| Archived | Westfield Woden     | Keltie Street       | Phillip   | ACT   | 2-10-2021 2:50PM - 3:50PM   | Monitor |
| Archived | Westfield Belconnen | Westfield Belconnen | Belconnen | ACT   | 23-9-2021 2:20PM - 3:30PM   | Monitor |
+----------+---------------------+---------------------+-----------+-------+-----------------------------+---------+
total items found: 3

$ covid-check -file 20211227.csv -q canberra -q lyneham -qn next -q old
+----------+----------------------+------------------+---------+-------+----------------------------+---------+
|  STATUS  |       LOCATION       |      STREET      | SUBURB  | STATE |         DATE/TIME          | CONTACT |
+----------+----------------------+------------------+---------+-------+----------------------------+---------+
| Archived | The Old Canberra Inn | 195 Mouat Street | Lyneham | ACT   | 11-12-2021 6:00PM - 7:00PM | Casual  |
+----------+----------------------+------------------+---------+-------+----------------------------+---------+
total items found: 1
```

## License

MIT - no obligations or warranties are provided with this application.