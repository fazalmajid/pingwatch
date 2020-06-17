package main

import (
	"database/sql"
	"log"
	"time"

	"github.com/sparrc/go-ping"
)

type Result = struct {
	Host string
	IP   string
	Rtt  time.Duration
}

func do_one_ping(host string, result chan Result) {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		log.Fatal(err)
	}
	pinger.Count = 1
	pinger.Timeout = 5 * time.Second
	pinger.SetPrivileged(*privileged)
	pinger.OnRecv = func(pkt *ping.Packet) {
		result <- Result{host, pkt.IPAddr.String(), pkt.Rtt}
	}
	pinger.Run()
}

func do_pings(db *sql.DB) {
	ips := get_dests(db)
	results := make(chan Result)
	for _, ip := range ips {
		go do_one_ping(ip, results)
	}
	for _ = range ips {
		result := <-results
		record(db, result)
	}
}

func start_pinger(db *sql.DB, interval time.Duration) {
	next := time.Now().Add(interval)
	for {
		sleep := next.Sub(time.Now())
		if *verbose {
			log.Println("next ping at", next, "sleeping for", sleep)
		}
		time.Sleep(sleep)
		next = next.Add(interval)
		do_pings(db)
	}
}
