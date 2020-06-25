package main

import (
	"database/sql"
	"log"
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
