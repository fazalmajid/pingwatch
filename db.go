package main

import (
	"database/sql"
	"log"
	"math"
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

func db_add_dest(db *sql.DB, host string) {
    log.Printf("adding host %s", host)
    _ , err := db.Exec("insert into dests (host) values (?)", host);
    if err != nil {
		log.Fatal("could not insert host into destinations", err)
	}
}

func db_del_dest(db *sql.DB, host string) {
    log.Printf("deleting host %s", host)
    _ , err := db.Exec("delete from dests where host=?", host);
    if err != nil {
		log.Fatal("could not delete host from destinations", err)
	}
}

func record(db *sql.DB, r *Result) {
	if *verbose {
		log.Printf("ping %s(%s) = %v\n", r.Host, r.IP, r.Rtt)
	}
	res, err := db.Exec("insert or replace into pings (time, host, ip, rtt) values (julianday('now'), ?, ?, ?)", r.Host, r.IP, 1e-6*float32(r.Rtt.Nanoseconds()))
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

func get_data(db *sql.DB, since float64) (header []string, ordered []int64, points map[int64][]float64) {
	rows, err := db.Query("SELECT time, host, ip, rtt FROM pings WHERE time > julianday('now')-? AND time > ?ORDER by 1, 2", display.Seconds()/86400.0, 2440587.5+since/86400000.0)
	if err != nil {
		log.Fatal(err)
	}
	points = make(map[int64][]float64, 0)
	colnum := 0
	cols := make(map[string]int, 0)
	var col int
	var ok bool
	var rounded time.Time
	var row []float64
	ordered = make([]int64, 0)

	for rows.Next() {
		var fts, rtt float64
		var host, ip string
		err = rows.Scan(&fts, &host, &ip, &rtt)
		if rtt == 0.0 || rtt == -3600e3 {
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		// julian day to Unix seconds
		fts = (fts - 2440587.5) * 86400.0
		rounded = time.Unix(int64(fts), int64(1e9*(fts-math.Trunc(fts))))
		// XXX this rounding down is not very clean, should try to run pings
		// XXX at the exact times instead
		rounded = rounded.Round(*interval)
		ts := rounded.Unix() * 1000
		col, ok = cols[host]
		if !ok {
			colnum += 1
			cols[host] = colnum
			col = colnum
		}
		row, ok = points[ts]
		if !ok {
			row = make([]float64, colnum)
			ordered = append(ordered, ts)
		}
		for len(row) < col {
			row = append(row, 0.0)
		}
		row[col-1] = rtt
		points[ts] = row
	}
	header = make([]string, colnum+1)
	header[0] = "Date"
	for colname, col := range cols {
		header[col] = colname
	}
	// fill out the first rows that may be missing columns
	num_cols := len(cols)
	for _, ts := range ordered {
		for len(points[ts]) < num_cols {
			points[ts] = append(points[ts], 0.0)
		}
	}
	return header, ordered, points
}
