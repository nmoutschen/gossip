package gossip

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

//peersHandler handles requests to '/peers'
func (c *Controller) peersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		c.peersGetHandler(w, r)
	} else if r.Method == http.MethodPost {
		c.peersPostHandler(w, r)
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
		config, ok := key.(Config)
		if !ok {
			log.WithFields(log.Fields{"controller": c, "func": "peersGetHandler", "config": config}).Warn("Failed to assert config")
			return true
		}

		peer, ok := value.(*Peer)
		if !ok {
			log.WithFields(log.Fields{"controller": c, "func": "peersGetHandler", "peer": peer}).Warn("Failed to assert peer")
			return true
		}

		n := &CtrlPeerResponse{
			Config: config,
		}
		for _, p := range peer.Peers {
			n.Peers = append(n.Peers, p.Config)
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
	config := &Config{}

	if err := json.NewDecoder(r.Body).Decode(config); err != nil {
		log.WithFields(log.Fields{"controller": c, "func": "peersPostHandler"}).Warn("Failed to decode request body")
		response(w, r, http.StatusInternalServerError, "Failed to decode request body")
		return
	}

	c.addPeerChan <- *config
	response(w, r, http.StatusOK, "Peer configuration received")
}
