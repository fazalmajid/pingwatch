package main

import (
	"database/sql"
	"net/http"
)

func webui_worker(db *sql.DB) {
	http.ListenAndServe(":8990", http.FileServer(http.Dir(".")))
}
