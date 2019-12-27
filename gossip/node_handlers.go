package gossip

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

//peersHandler handles requests to the '/peers' path
func (n *Node) peersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		n.peersGetHandler(w, r)
	} else if r.Method == http.MethodPost {
		n.peersPostHandler(w, r)
	} else {
		methodNotAllowedHandler(w, r)
	}
}

//peersGetHandler handles 'GET /peers' requests
func (n *Node) peersGetHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{"node": n, "func": "peersGetHandler"}).Info("Received GET /peers")
	msg := PeersResponse{}
	for _, peer := range n.Peers {
		msg.Peers = append(msg.Peers, peer.Config)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(msg)
}

//peersPostHandler handles 'POST /peers' requests
func (n *Node) peersPostHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{"node": n, "func": "peersPostHandler"}).Info("Received POST /peers")
	config := &Config{}

	if err := json.NewDecoder(r.Body).Decode(config); err != nil {
		log.WithFields(log.Fields{"node": n, "func": "peersPostHandler"}).Warn("Failed to decode request body")
		response(w, r, http.StatusInternalServerError, "Failed to decode request body")
		return
	}

	n.peerChan <- *config
	response(w, r, http.StatusOK, "Peering request received")
}

//rootHandler handles requests to the '/' path
func (n *Node) rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		n.rootGetHandler(w, r)
	} else if r.Method == http.MethodPost {
		n.rootPostHandler(w, r)
	} else {
		methodNotAllowedHandler(w, r)
	}
}

//rootGetHandler handles 'GET /' requests
func (n *Node) rootGetHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{"node": n, "func": "rootGetHandler"}).Info("Received GET /")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(n.State)
}

//rootPostHandler handles 'POST /' requests
func (n *Node) rootPostHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{"node": n, "func": "rootPostHandler"}).Info("Received POST /")
	state := &State{}

	if err := json.NewDecoder(r.Body).Decode(state); err != nil {
		log.WithFields(log.Fields{"node": n, "func": "rootPostHandler"}).Warn("Failed to decode request body")
		response(w, r, http.StatusInternalServerError, "Failed to decode request body")
		return
	}

	n.stateChan <- *state
	response(w, r, http.StatusOK, "State received")
}

//statusHandler handles requests to '/status'
func (n *Node) statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.WithFields(log.Fields{"node": n, "func": "statusHandler"}).Info("Received GET /status")
		methodNotAllowedHandler(w, r)
		return
	}

	status := StatusResponse{
		LastState: n.State.Timestamp,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
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
