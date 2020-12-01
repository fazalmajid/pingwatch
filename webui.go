package main

import (
	"database/sql"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"

	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rakyll/statik/fs"
	"strconv"
)

//go:generate statik -src . -include index.html,dygraph.min.js,initial.js,delta.js -f -p main
//go:generate mv main/statik.go .
//go:generate rmdir main

const preamble = `
`

func open_js_template(sfs http.FileSystem, name string) *template.Template {
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
	// html/template assumes the content is HTML, so fake it
	t, err = t.Parse("<script>" + string(b) + "</script>")
	if err != nil {
		log.Fatal("could not parse embedded template ", name, ": ", err)
	}
	return t
}
func render_js_template(t *template.Template, w http.ResponseWriter, r *http.Request, data map[string]interface{}) {
	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		log.Println("error rendering delta", err)
		return
	}
	// Add Content-Type javascript to please the Content Security Policies
	w.Header().Add("Content-Type", "text/javascript")
	// strip the <script>...</script> tags added in open_js_template
	out := buf.Bytes()
	_, err = w.Write(out[8 : len(out)-9])
	if err != nil {
		log.Println("error writing", t.Name, ":", err)
	}
}

type ErrorMsg struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func Error(w http.ResponseWriter, req *http.Request, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	j, _ := json.Marshal(ErrorMsg{Status: "error", Reason: msg})
	w.Write(j)
	w.Write([]byte{'\n'})
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

	initial := open_js_template(sfs, "initial.js")
	http.HandleFunc("/initial", func(w http.ResponseWriter, r *http.Request) {
		header, ordered, points := get_data(db, 0.0)
		render_js_template(initial, w, r, map[string]interface{}{
			"Header":  header,
			"Ordered": ordered,
			"Points":  points,
			"Delay":   interval.Nanoseconds() / 1000000,
		})
	})

	delta := open_js_template(sfs, "delta.js")
	http.HandleFunc("/delta", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			Error(w, r, "could not parse arguments: %v", err)
			return
		}
		since, err := strconv.ParseFloat(r.Form.Get("since"), 64)
		if err != nil {
			Error(w, r, "invalid value for 'since' \"%v\": %v",
				r.Form.Get("since"), err)
			return
		}

		_, ordered, points := get_data(db, since)
		if *verbose {
			log.Println("found", len(ordered), "points")
		}
		render_js_template(delta, w, r, map[string]interface{}{
			"Ordered": ordered,
			"Points":  points,
			"MinData": display.Seconds() / interval.Seconds(),
		})
	})

	http.ListenAndServe(*port, nil)
}
