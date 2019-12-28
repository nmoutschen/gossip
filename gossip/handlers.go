package gossip

import (
	"encoding/json"
	"net/http"
)

//methodNotAllowedHandler handles requests with unsupported request methods
func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	response(w, r, http.StatusMethodNotAllowed, "Method Not Allowed")
}

//response sends basic responses back to the requester
func response(w http.ResponseWriter, r *http.Request, statusCode int, msg string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Message: msg,
	})
}
