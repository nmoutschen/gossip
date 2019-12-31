package gossip

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestControllerPeersHandlerGet(t *testing.T) {
	//Prepare peer and controller
	peer := NewPeer(Config{"127.0.0.1", 8080})
	c := NewController("127.0.0.1", 7080)
	c.Peers.Store(peer.Config, peer)
	peer.Peers = []*Peer{peer}

	//Send request
	req := httptest.NewRequest("GET", "http://127.0.0.1:7080/peers", nil)
	w := httptest.NewRecorder()
	c.peersHandler(w, req)
	res := w.Result()

	//Parse response
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	}

	var cpr CtrlPeersResponse
	json.NewDecoder(res.Body).Decode(&cpr)

	if len(cpr.Nodes) != 1 {
		t.Errorf("len(cpr.Nodes) == %d; want %d", len(cpr.Nodes), 1)
		return
	}

	if cpr.Nodes[0].Config != peer.Config {
		t.Errorf("cpr.Nodes[0].Config == %v; want %v", cpr.Nodes[0].Config, peer.Config)
	}

	if len(cpr.Nodes[0].Peers) != 1 {
		t.Errorf("len(cpr.Nodes[0].Peers) == %d; want %d", len(cpr.Nodes[0].Peers), 1)
		return
	}

	if cpr.Nodes[0].Peers[0] != peer.Config {
		t.Errorf("cpr.Nodes[0].Peers[0] == %v; want %v", cpr.Nodes[0].Peers[0], peer.Config)
	}
}

func TestControllerPeersHandlerPost(t *testing.T) {
	//Prepare config and controller
	config := Config{"127.0.0.1", 8080}
	c := NewController("127.0.0.1", 7080)
	reqBody, _ := json.Marshal(config)

	//Send request
	req := httptest.NewRequest("POST", "http://127.0.0.1:7080/peers", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	c.peersHandler(w, req)
	res := w.Result()

	//Parse response
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	}

	rConfig := <-c.addPeerChan

	if rConfig != config {
		t.Errorf("rConfig == %v; want %v", rConfig, config)
	}
}
