package gossip

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

//peersHandler handles requests to the '/peers' path
func (n *Node) peersHandler(w http.ResponseWriter, r *http.Request) {
	corsHeadersResponse(&w, r, n.config, "GET, POST")
	if r.Method == http.MethodGet {
		n.peersGetHandler(w, r)
	} else if r.Method == http.MethodPost {
		n.peersPostHandler(w, r)
	} else if r.Method == http.MethodOptions {
		corsOptionsResponse(w, r, n.config, "GET, POST")
	} else {
		methodNotAllowedHandler(w, r)
	}
}

//peersGetHandler handles 'GET /peers' requests
func (n *Node) peersGetHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{"node": n, "func": "peersGetHandler"}).Info("Received GET /peers")
	msg := PeersResponse{}
	for _, peer := range n.Peers {
		msg.Peers = append(msg.Peers, peer.Addr)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(msg)
}

//peersPostHandler handles 'POST /peers' requests
func (n *Node) peersPostHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{"node": n, "func": "peersPostHandler"}).Info("Received POST /peers")
	addr := &Addr{}

	if err := json.NewDecoder(r.Body).Decode(addr); err != nil {
		log.WithFields(log.Fields{"node": n, "func": "peersPostHandler"}).Warn("Failed to decode request body")
		response(w, r, http.StatusInternalServerError, "Failed to decode request body")
		return
	}

	n.addPeerChan <- *addr
	response(w, r, http.StatusOK, "Peering request received")
}

//rootHandler handles requests to the '/' path
func (n *Node) rootHandler(w http.ResponseWriter, r *http.Request) {
	corsHeadersResponse(&w, r, n.config, "GET, POST")
	if r.Method == http.MethodGet {
		n.rootGetHandler(w, r)
	} else if r.Method == http.MethodPost {
		n.rootPostHandler(w, r)
	} else if r.Method == http.MethodOptions {
		corsOptionsResponse(w, r, n.config, "GET, POST")
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
		log.WithFields(log.Fields{"node": n, "func": "rootPostHandler"}).Warnf("Failed to decode request body: %s", err.Error())
		response(w, r, http.StatusInternalServerError, "Failed to decode request body")
		return
	}

	n.stateChan <- *state
	response(w, r, http.StatusOK, "State received")
}

//statusHandler handles requests to '/status'
func (n *Node) statusHandler(w http.ResponseWriter, r *http.Request) {
	corsHeadersResponse(&w, r, n.config, "GET")
	if r.Method == http.MethodOptions {
		corsOptionsResponse(w, r, n.config, "GET")
		return
	} else if r.Method != http.MethodGet {
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
