package main

import (
	"encoding/json"
	"net/http"
)

// AuthDiscoverHandler returns a JSON document describing the currently configured authentication
// service.
func AuthDiscoverHandler(c *Context, w http.ResponseWriter, r *http.Request) {
	type response struct {
		Address string `json:"address"`
		Style   string `json:"style"`
	}

	resp := response{
		Address: c.Settings.AuthService,
		Style:   "local",
	}

	json.NewEncoder(w).Encode(resp)
}
