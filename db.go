package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

func create_table(db *sql.DB, name string, ddl string) {
	row := db.QueryRow("SELECT count(sql) FROM sqlite_master WHERE name=?", name)
	var count int32
	err := row.Scan(&count)
	if err != nil {
		log.Fatal("Could not check table status ")
	}
	if count == 0 {
		_, err = db.Exec(ddl)
		if err != nil {
			log.Fatal("Could not create dests table", err)
		}
	}
}

func db_init(db *sql.DB) {
	create_table(db, "dests", "CREATE TABLE dests (host TEXT PRIMARY KEY)")
	create_table(db, "pings", "CREATE TABLE pings (time REAL, host TEXT, ip TEXT, rtt REAL, PRIMARY KEY(time, host))")
}

func get_dests(db *sql.DB) []string {
	dests := make([]string, 0)
	rows, err := db.Query("SELECT host FROM dests")
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		var ip string
		err = rows.Scan(&ip)
		if err != nil {
			log.Fatal(err)
		}
		dests = append(dests, ip)
	}
	return dests
}

func record(db *sql.DB, r *Result) {
	if *verbose {
		log.Printf("ping %s(%s) = %v\n", r.Host, r.IP, r.Rtt)
	}
	res, err := db.Exec("insert into pings (time, host, ip, rtt) values (julianday('now'), ?, ?, ?)", r.Host, r.IP, 1e-6*float32(r.Rtt.Nanoseconds()))
	if err != nil {
		log.Fatal(err)
	}
	rowCount, err := res.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	if rowCount != 1 {
		log.Fatal("could not insert row in pings table", rowCount)
	}
}

func ResultWorker(db *sql.DB) chan *Result {
	in := make(chan *Result, 5)

	go func() {
		for {
			r := <-in
			if r == nil {
				return
			}
			record(db, r)
		}
	}()

	return in
}

func get_data(db *sql.DB) []string {
	rows, err := db.Query("SELECT time, host, ip, rtt FROM pings WHERE time > julianday('now')-14 ORDER by 1, 2")
	if err != nil {
		log.Fatal(err)
	}
	points := make(map[time.Time][]float64, 0)
	colnum := 0
	cols := make(map[string]int, 0)
	var col int
	var ok bool
	var rounded time.Time
	var row []float64
	ordered := make([]time.Time, 0)

	for rows.Next() {
		var ts, rtt float64
		var host, ip string
		err = rows.Scan(&ts, &host, &ip, &rtt)
		if rtt == 0.0 || rtt == -3600e3 {
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		// julian day to Unix seconds
		ts = (ts - 2440587.5) * 86400.0
		rounded = time.Unix(int64(ts), int64(1e9*(ts-math.Trunc(ts))))
		// XXX this rounding down is not very clean, should try to run pings
		// XXX at the exact times instead
		rounded = rounded.Round(*interval)
		col, ok = cols[host]
		if !ok {
			colnum += 1
			cols[host] = colnum
			col = colnum
		}
		row, ok = points[rounded]
		if !ok {
			row = make([]float64, colnum)
			ordered = append(ordered, rounded)
		}
		for len(row) < col {
			row = append(row, 0.0)
		}
		row[col-1] = rtt
		points[rounded] = row
	}
	header := make([]string, colnum+1)
	header[0] = "Date"
	for colname, col := range cols {
		header[col] = colname
	}
	data := []string{strings.Join(header, ",") + "\n"}
	for _, rounded = range ordered {
		row = points[rounded]
		s := rounded.Format("2006-01-02T15:04:05")
		for i := 0; i < len(row); i++ {
			if row[i] == 0.0 || row[i] == -3600e3 {
				s = s + ",-100.0"
			} else {
				s = fmt.Sprintf("%s,%f", s, row[i])
			}
		}
		if len(row) < colnum {
			s = s + strings.Repeat(",-100.0", colnum-len(row))
		}
		s = s + "\n"

		data = append(data, s)
	}
	return data
}
