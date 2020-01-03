package gossip

import (
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

//peersHandler handles requests to '/peers'
func (c *Controller) peersHandler(w http.ResponseWriter, r *http.Request) {
	corsHeadersResponse(&w, r, c.config, "GET, POST")
	if r.Method == http.MethodGet {
		c.peersGetHandler(w, r)
	} else if r.Method == http.MethodPost {
		c.peersPostHandler(w, r)
	} else if r.Method == http.MethodOptions {
		corsOptionsResponse(w, r, c.config, "GET, POST")
	} else {
		methodNotAllowedHandler(w, r)
	}
}

//peersGetHandler handles 'GET /peers' requests
func (c *Controller) peersGetHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{"controller": c, "func": "peersGetHandler"}).Info("Received GET /peers")
	//Create response
	cpr := &CtrlPeersResponse{}
	c.Peers.Range(func(key, value interface{}) bool {
		addr, ok := key.(Addr)
		if !ok {
			log.WithFields(log.Fields{"controller": c, "func": "peersGetHandler", "addr": addr}).Warn("Failed to assert address")
			return true
		}

		peer, ok := value.(*Peer)
		if !ok {
			log.WithFields(log.Fields{"controller": c, "func": "peersGetHandler", "peer": peer}).Warn("Failed to assert peer")
			return true
		}

		n := &CtrlPeerResponse{
			Addr: addr,
		}
		for _, p := range peer.Peers {
			n.Peers = append(n.Peers, p.Addr)
		}

		cpr.Nodes = append(cpr.Nodes, *n)
		return true
	})

	//Send response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(*cpr)
}

//peersPostHandler handles 'POST /peers' requests
func (c *Controller) peersPostHandler(w http.ResponseWriter, r *http.Request) {
	log.WithFields(log.Fields{"controller": c, "func": "peersPostHandler"}).Info("Received POST /peers")
	addr := &Addr{}

	if err := json.NewDecoder(r.Body).Decode(addr); err != nil {
		log.WithFields(log.Fields{"controller": c, "func": "peersPostHandler"}).Warn("Failed to decode request body")
		response(w, r, http.StatusInternalServerError, "Failed to decode request body")
		return
	}

	//Invalid port number
	if (*addr).Port == 0 {
		response(w, r, http.StatusBadRequest, "Required property 'port' is 0 or not present")
		return
	}

	/*Infer that the client node does not know its IP address and use the one
	from the HTTP request instead.
	*/
	if (*addr).IP == "" {
		log.WithFields(log.Fields{"controller": c, "func": "peersPostHandler"}).Infof("Auto-detecting IP address for peer: %s", strings.SplitN(r.RemoteAddr, ":", 2)[0])
		(*addr).IP = strings.SplitN(r.RemoteAddr, ":", 2)[0]
	}

	c.addPeerChan <- *addr
	response(w, r, http.StatusOK, "Peer address received")
}
