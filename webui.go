package main

import (
	"database/sql"
	"io"
	"log"
	"net/http"

	"github.com/rakyll/statik/fs"
)

//go:generate statik -src . -include index.html,dygraph.min.js -f -p main
//go:generate mv main/statik.go .
//go:generate rmdir main

func webui_worker(db *sql.DB) {
	sfs, err := fs.New()
	if err != nil {
		log.Fatal("could not init statik", err)
	}
	fsh := http.FileServer(sfs)
	http.Handle("/", fsh)
	http.Handle("/dygraph.min.js", fsh)
	http.Handle("/index.html", fsh)
	dataHandler := func(w http.ResponseWriter, _ *http.Request) {
		data := get_data(db)
		for _, row := range data {
			io.WriteString(w, row)
		}
	}
	http.HandleFunc("/data", dataHandler)

	http.ListenAndServe(*port, nil)
}
