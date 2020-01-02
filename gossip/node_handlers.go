package gossip

import (
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

//peersHandler handles requests to the '/peers' path
func (n *Node) peersHandler(w http.ResponseWriter, r *http.Request) {
	corsHeadersResponse(&w, r, n.config, "GET, POST")
	switch r.Method {
	case http.MethodDelete:
		n.peersDeleteHandler(w, r)
	case http.MethodGet:
		n.peersGetHandler(w, r)
	case http.MethodPost:
		n.peersPostHandler(w, r)
	case http.MethodOptions:
		corsOptionsResponse(w, r, n.config, "GET, POST")
	default:
		methodNotAllowedHandler(w, r)
	}
}

//peersDeleteHandler handles 'DELETE /peers' requests
func (n *Node) peersDeleteHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{"node": n, "func": "peersDeleteHandler"}).Info("Received GET /peers")
	addr := &Addr{}

	if err := json.NewDecoder(r.Body).Decode(addr); err != nil {
		log.WithFields(log.Fields{"node": n, "func": "peersDeleteHandler"}).Warnf("Failed to decode request body: %s", err.Error())
		response(w, r, http.StatusInternalServerError, "Failed to decode request body")
		return
	}

	/*Infer that the client node does not know its IP address and use the one
	from the HTTP request instead.
	*/
	if (*addr).IP == "" {
		log.WithFields(log.Fields{"node": n, "func": "peersDeleteHandler"}).Infof("Auto-detecting IP address for peer: %s", strings.SplitN(r.RemoteAddr, ":", 2)[0])
		(*addr).IP = strings.SplitN(r.RemoteAddr, ":", 2)[0]
	}

	n.deletePeerChan <- *addr
	response(w, r, http.StatusOK, "Peer deletion request received")
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
		log.WithFields(log.Fields{"node": n, "func": "peersPostHandler"}).Warnf("Failed to decode request body: %s", err.Error())
		response(w, r, http.StatusInternalServerError, "Failed to decode request body")
		return
	}

	/*Infer that the client node does not know its IP address and use the one
	from the HTTP request instead.
	*/
	if (*addr).IP == "" {
		log.WithFields(log.Fields{"node": n, "func": "peersPostHandler"}).Infof("Auto-detecting IP address for peer: %s", strings.SplitN(r.RemoteAddr, ":", 2)[0])
		(*addr).IP = strings.SplitN(r.RemoteAddr, ":", 2)[0]
	}

	n.addPeerChan <- *addr
	response(w, r, http.StatusOK, "Peering request received")
}

//rootHandler handles requests to the '/' path
func (n *Node) rootHandler(w http.ResponseWriter, r *http.Request) {
	corsHeadersResponse(&w, r, n.config, "GET, POST")
	switch r.Method {
	case http.MethodGet:
		n.rootGetHandler(w, r)
	case http.MethodPost:
		n.rootPostHandler(w, r)
	case http.MethodOptions:
		corsOptionsResponse(w, r, n.config, "GET, POST")
	default:
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
