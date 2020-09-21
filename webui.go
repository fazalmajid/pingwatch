package main

import (
	"database/sql"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/rakyll/statik/fs"
)

//go:generate statik -src . -include index.html,dygraph.min.js,initial.js -f -p main
//go:generate mv main/statik.go .
//go:generate rmdir main

const preamble = `
`

func open_template(sfs http.FileSystem, name string) *template.Template {
	f, err := sfs.Open("/" + name)
	if err != nil {
		log.Fatal("could not open embedded template", name, ": ", err)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal("could not read embedded template ", name, ": ", err)
	}
	t := template.New(name)
	t, err = t.Parse(string(b))
	if err != nil {
		log.Fatal("could not parse embedded template ", name, ": ", err)
	}
	return t
}

func webui_worker(db *sql.DB) {
	sfs, err := fs.New()
	if err != nil {
		log.Fatal("could not init statik", err)
	}
	fsh := http.FileServer(sfs)
	http.Handle("/", fsh)
	http.Handle("/dygraph.min.js", fsh)
	http.Handle("/index.html", fsh)

	initial := open_template(sfs, "initial.js")
	http.HandleFunc("/initial", func(w http.ResponseWriter, r *http.Request) {
		header, ordered, points := get_data(db)
		initial.Execute(w, map[string]interface{}{
			"Header":  header,
			"Ordered": ordered,
			"Points":  points,
			"Req":     r,
		})
	})

	http.ListenAndServe(*port, nil)
}
