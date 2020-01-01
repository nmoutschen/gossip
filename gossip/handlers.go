package gossip

import (
	"encoding/json"
	"net/http"
)

//corsHeadersResponse sets up headers for CORS
func corsHeadersResponse(w *http.ResponseWriter, r *http.Request, methods string) {
	(*w).Header().Add("Access-Control-Allow-Origin", CorsAllowOrigin)
	(*w).Header().Add("Access-Control-Allow-Headers", CorsAllowHeaders)
	(*w).Header().Add("Access-Control-Allow-Methods", methods)
}

//optionsResponse sends a CORS pre-flight OPTIONS request
func corsOptionsResponse(w http.ResponseWriter, r *http.Request, methods string) {
	corsHeadersResponse(&w, r, methods)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(""))
}

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
