package gossip

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNodePeersHandlerGet(t *testing.T) {
	//Prepare peer and node
	peer := NewPeer(Addr{"127.0.0.1", 8081})
	n := NewNode("127.0.0.1", 8080)
	n.Peers = append(n.Peers, peer)

	//Send request
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/peers", nil)
	w := httptest.NewRecorder()
	n.peersHandler(w, req)
	res := w.Result()

	//Parse response
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	}

	var pr PeersResponse
	json.NewDecoder(res.Body).Decode(&pr)

	if len(pr.Peers) != 1 {
		t.Errorf("len(pr.Peers) == %d; want %d", len(pr.Peers), 1)
	} else if pr.Peers[0] != peer.Addr {
		t.Errorf("pr.Peers[1] == %v; want %v", pr.Peers[0], peer.Addr)
	}
}

func TestNodePeersHandlerPost(t *testing.T) {
	//Prepare address and node
	addr := Addr{"127.0.0.1", 8081}
	n := NewNode("127.0.0.1", 8080)
	reqBody, _ := json.Marshal(addr)

	//Send request
	req := httptest.NewRequest("POST", "http://127.0.0.1:8080/peers", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	n.peersHandler(w, req)
	res := w.Result()

	//Parse response
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	} else {
		rAddr := <-n.addPeerChan

		if rAddr != addr {
			t.Errorf("rAddr == %v; want %v", rAddr, addr)
		}
	}
}

func TestNodePeersHandlerOptions(t *testing.T) {
	//Prepare peer and node
	n := NewNode("127.0.0.1", 8080)

	//Send request
	req := httptest.NewRequest("OPTIONS", "http://127.0.0.1:8080/peers", nil)
	w := httptest.NewRecorder()
	n.peersHandler(w, req)
	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	}
}

func TestNodeRootHandlerGet(t *testing.T) {
	//Prepare state and node
	state := State{time.Now().UnixNano(), "Test data"}
	n := NewNode("127.0.0.1", 8080)
	n.State = state

	//Send request
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080", nil)
	w := httptest.NewRecorder()
	n.rootHandler(w, req)
	res := w.Result()

	//Parse response
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	}

	var rState State
	json.NewDecoder(res.Body).Decode(&rState)

	if rState != state {
		t.Errorf("rState == %v; want %v", rState, state)
	}
}

func TestNodeRootHandlerPost(t *testing.T) {
	//Prepare state and node
	state := State{time.Now().UnixNano(), "Test data"}
	n := NewNode("127.0.0.1", 8080)
	reqBody, _ := json.Marshal(state)

	//Send request
	req := httptest.NewRequest("POST", "http://127.0.0.1:8080", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()
	n.rootHandler(w, req)
	res := w.Result()

	//Parse response
	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	} else {
		rState := <-n.stateChan

		if rState != state {
			t.Errorf("rState == %v; want %v", rState, state)
		}
	}
}

func TestNodeRootHandlerOptions(t *testing.T) {
	//Prepare peer and node
	n := NewNode("127.0.0.1", 8080)

	//Send request
	req := httptest.NewRequest("OPTIONS", "http://127.0.0.1:8080", nil)
	w := httptest.NewRecorder()
	n.rootHandler(w, req)
	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	}
}

func TestNodeStatusHandlerGet(t *testing.T) {
	//Prepare state and node
	state := State{time.Now().UnixNano(), "Test data"}
	n := NewNode("127.0.0.1", 8080)
	n.State = state

	//Send request
	req := httptest.NewRequest("GET", "http://127.0.0.1:8080/status", nil)
	w := httptest.NewRecorder()
	n.statusHandler(w, req)
	res := w.Result()

	//Parse response
	var sr StatusResponse
	json.NewDecoder(res.Body).Decode(&sr)

	if sr.LastState != state.Timestamp {
		t.Errorf("sr.LastState == %d; want %d", sr.LastState, state.Timestamp)
	}
}

func TestNodeStatusHandlerOptions(t *testing.T) {
	//Prepare peer and node
	n := NewNode("127.0.0.1", 8080)

	//Send request
	req := httptest.NewRequest("OPTIONS", "http://127.0.0.1:8080/status", nil)
	w := httptest.NewRecorder()
	n.statusHandler(w, req)
	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Errorf("res.StatusCode == %d; want %d", res.StatusCode, http.StatusOK)
	}
}
