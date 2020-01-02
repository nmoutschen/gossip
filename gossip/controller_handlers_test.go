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
	peer := NewPeer(Addr{"127.0.0.1", 8080}, nil)
	c := NewController(nil)
	c.Peers.Store(peer.Addr, peer)
	peer.Peers = []*Peer{peer}

	//Send request
	req := httptest.NewRequest("GET", c.URL()+"/peers", nil)
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

	if cpr.Nodes[0].Addr != peer.Addr {
		t.Errorf("cpr.Nodes[0].Addr == %v; want %v", cpr.Nodes[0].Addr, peer.Addr)
	}

	if len(cpr.Nodes[0].Peers) != 1 {
		t.Errorf("len(cpr.Nodes[0].Peers) == %d; want %d", len(cpr.Nodes[0].Peers), 1)
		return
	}

	if cpr.Nodes[0].Peers[0] != peer.Addr {
		t.Errorf("cpr.Nodes[0].Peers[0] == %v; want %v", cpr.Nodes[0].Peers[0], peer.Addr)
	}
}

func TestControllerPeersHandlerPost(t *testing.T) {
	//Prepare addr and controller
	addr := Addr{"127.0.0.1", 8080}
	c := NewController(nil)
	reqBody, _ := json.Marshal(addr)

	//Send request
	req := httptest.NewRequest("POST", "http://127.0.0.1:7080/peers", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	c.peersHandler(w, req)
	res := w.Result()

	//Parse response
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	}

	rAddr := <-c.addPeerChan

	if rAddr != addr {
		t.Errorf("rAddr == %v; want %v", rAddr, addr)
	}
}
