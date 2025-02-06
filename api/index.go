package main

import (
	"fmt"
	"net/http"
)

// Handler adalah entry point untuk serverless function
func Handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "Halo, dari API Poliklinik Backend!")
}
