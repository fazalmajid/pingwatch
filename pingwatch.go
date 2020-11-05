package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const date_fmt = "[02/Jan/2006:15:04:05 -0700]"
const iso_8601 = "2006-01-02 15:04:05"

var (
	verbose    *bool
	privileged *bool
	interval   *time.Duration
	display    *time.Duration
	port       *string
	addhost    *string
	delhost    *string	
)

func main() {
	// command-line options
	verbose = flag.Bool("v", false, "Verbose error reporting")
    addhost = flag.String("add","","Add a host to list of ping destinations")
    delhost = flag.String("del","","Delete a host from list of ping destinations")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	dsn := flag.String("db", "pingwatch.sqlite", "SQLite DB to use for the search index")
	interval = flag.Duration("interval", 60*time.Second, "ping interval in seconds")
	display = flag.Duration("display", 14*86400*time.Second, "default date range to display in the Web UI")
	days := flag.Int("days", 0, "alternate way to specify display window, in days")
	privileged = flag.Bool("privileged", true, "whether to use privileged ICMP or unprivileged UDP")
	port = flag.String("p", "localhost:8086", "host address and port to bind to")
	flag.Parse()
	if *days != 0 {
		day_window := time.Duration(*days*86400) * time.Second
		display = &day_window
	}

	var err error
	var f *os.File
	var db *sql.DB
	// Profiler
	if *cpuprofile != "" {
		f, err = os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

    if *addhost != "" {

        
    log.Printf("Adding host %s", *addhost)
    if *dsn != "" {
		db, err = sql.Open("sqlite3", *dsn)
		if err != nil {
			log.Fatalf("ERROR: opening SQLite DB %q, error: %s", *dsn, err)
		}
    
    
        db_add_dest (db, *addhost)
    
    }
    
    
    
    
    os.Exit(0)   
    }

    if *delhost != "" {
     
    if *dsn != "" {
		db, err = sql.Open("sqlite3", *dsn)
		if err != nil {
			log.Fatalf("ERROR: opening SQLite DB %q, error: %s", *dsn, err)
		}
    
    
        db_del_dest (db, *delhost)
    
    }
    
    os.Exit(0)          
    }
    
    
	log.Println("starting pingwatch")

	end := make(chan os.Signal)
	signal.Notify(end, syscall.SIGINT, syscall.SIGTERM, syscall.SIGPIPE)

	// SIGUSR1 dumps goroutines on stdout
	usr1Chan := make(chan os.Signal, 1)
	signal.Notify(usr1Chan, syscall.SIGUSR1)
	go func() {
		for {
			<-usr1Chan
			dump_goroutines()
		}
	}()

    
    
	if *dsn != "" {
		db, err = sql.Open("sqlite3", *dsn)
		if err != nil {
			log.Fatalf("ERROR: opening SQLite DB %q, error: %s", *dsn, err)
		}
		defer db.Close()
		db_init(db)
		go start_pinger(db, *interval)
		go webui_worker(db)
		<-end
	}
}

func dump_goroutines() {
	log.Println("DUMPING GOROUTINES")
	buf := make([]byte, 1<<24)
	bytes := runtime.Stack(buf, true)
	fmt.Printf("%s", buf[:bytes])
}
