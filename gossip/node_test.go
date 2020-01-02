package gossip

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewNode(t *testing.T) {
	n := NewNode(nil)

	if n.IP != DefaultConfig.Node.IP {
		t.Errorf("n.IP == %s; want %s", n.IP, DefaultConfig.Node.IP)
	}
	if n.Port != DefaultConfig.Node.Port {
		t.Errorf("n.Port == %d; want %d", n.Port, DefaultConfig.Node.Port)
	}
}

func TestNodeAddPeer(t *testing.T) {
	n := NewNode(nil)
	addr := Addr{"127.0.0.1", 8081}

	// Test if the node skips itself
	n.AddPeer(Addr{"127.0.0.1", 8080})

	if len(n.Peers) != 0 {
		t.Errorf("len(n.Peers) == %d; want %d", len(n.Peers), 0)
	}

	//Adding one peer addr
	n.AddPeer(addr)

	//If all goes well, there should be one peer in n.Peers
	if len(n.Peers) != 1 {
		t.Errorf("len(n.Peers) == %d; want %d", len(n.Peers), 1)
	}
	if n.Peers[0].Addr != addr {
		t.Errorf("n.Peers[0].Addr == %v; want %v", n.Peers[0].Addr, addr)
	}

	//Adding the same peer address
	n.AddPeer(addr)

	//Since we are sending the same addr, only one peer should be there
	if len(n.Peers) != 1 {
		t.Errorf("len(n.Peers) == %d; want %d", len(n.Peers), 1)
	}
	if n.Peers[0].Addr != addr {
		t.Errorf("n.Peers[0].Addr == %v; want %v", n.Peers[0].Addr, addr)
	}
}

func TestNodeFindPeer(t *testing.T) {
	n := NewNode(nil)

	peer1 := NewPeer(Addr{"127.0.0.1", 8081}, nil)
	peer2 := NewPeer(Addr{"127.0.0.1", 8082}, nil)
	peer3 := NewPeer(Addr{"127.0.0.1", 8083}, nil)
	peer4 := NewPeer(Addr{"127.0.0.1", 8084}, nil)

	testCases := []struct {
		Peers []*Peer
		Addr  Addr
		Pos   int
		Found bool
	}{
		{[]*Peer{}, peer1.Addr, -1, false},
		{[]*Peer{peer1}, peer1.Addr, 0, true},
		{[]*Peer{peer1}, peer2.Addr, -1, false},
		{[]*Peer{peer1, peer2, peer3}, peer1.Addr, 0, true},
		{[]*Peer{peer1, peer2, peer3}, peer2.Addr, 1, true},
		{[]*Peer{peer1, peer2, peer3}, peer3.Addr, 2, true},
		{[]*Peer{peer1, peer2, peer3}, peer4.Addr, -1, false},
	}

	for i, testCase := range testCases {
		n.Peers = testCase.Peers
		pos, found := n.FindPeer(testCase.Addr)
		if pos != testCase.Pos {
			t.Errorf("pos == %d for test case %d; want %d", pos, i, testCase.Pos)
		}
		if found != testCase.Found {
			t.Errorf("found == %t for test case %d; want %t", found, i, testCase.Found)
		}
	}
}

func TestNodeFetchStateWorker(t *testing.T) {
	var received bool
	state := State{time.Now().UnixNano(), "Test data"}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/")
		}
		received = true
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(state)
	}))
	defer func() { testServer.Close() }()

	peer := NewPeer(parseURL(testServer.URL), nil)
	peer.LastState = state.Timestamp

	n := NewNode(nil)
	n.Peers = append(n.Peers, peer)

	go n.fetchStateWorker()
	n.fetchStateChan <- peer
	newState := <-n.stateChan

	if newState != state {
		t.Errorf("newState == %v; want %v", newState, state)
	}

	if !received {
		t.Errorf("HTTP Server never received a request")
	}
}

func TestNodePeerSendStateWorker(t *testing.T) {
	testCases := []struct {
		Recipients int
		Expected   int
	}{
		{0, 0},
		{1, 1},
		{DefaultConfig.Node.MaxRecipients, DefaultConfig.Node.MaxRecipients},
		{DefaultConfig.Node.MaxRecipients + 1, DefaultConfig.Node.MaxRecipients},
	}

	var receivedCount int
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/")
		}
		receivedCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Message: "State received",
		})
	}))
	peer := NewPeer(parseURL(testServer.URL), nil)

	state := State{time.Now().UnixNano(), "Test data"}
	n := NewNode(nil)

	for i, testCase := range testCases {
		for i := 0; i < testCase.Recipients; i++ {
			n.Peers = append(n.Peers, peer)
		}

		count := n.PeerSendState(state)

		/*Need to wait for asynchronous processing. This should be enough but
		could cause issues.
		*/
		time.Sleep(100 * time.Millisecond)

		if count != testCase.Expected {
			t.Errorf("count == %d in test case %d; want %d", count, i, testCase.Expected)
		}
		if receivedCount != testCase.Expected {
			t.Errorf("receivedCount == %d in test case %d; want %d", receivedCount, i, testCase.Expected)
		}
		receivedCount = 0
	}

	//TODO: Test that testCase.Expected requests are sent to the peers
}

func TestNodePingPeers(t *testing.T) {
	//Initialize peer server
	var received bool
	peer := &Peer{config: DefaultConfig}
	peer.UpdateStatus(true)
	peer.UpdateStatus(false)
	peer.LastState = time.Now().UnixNano()
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/status" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/status")
		}
		received = true
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(StatusResponse{
			LastState: peer.LastState,
		})
	}))
	defer func() { testServer.Close() }()
	peer.Addr = parseURL(testServer.URL)

	//Initialize node
	n := NewNode(nil)
	n.Peers = append(n.Peers, peer)

	//Ping all peers
	go n.PingPeers()

	//Wait for a peer from n.PingPeers()
	rPeer := <-n.fetchStateChan

	//Check results
	if rPeer != peer {
		t.Errorf("rPeer == %v; want %v", rPeer, peer)
	}

	if peer.Attempts != 0 {
		t.Errorf("peer.Attempts == %d; want %d", peer.Attempts, 0)
	}

	if !received {
		t.Errorf("HTTP Server never received a request")
	}
}

func TestNodePingPeersUnreachable(t *testing.T) {
	//Initialize peer
	var received bool
	peer := &Peer{config: DefaultConfig}
	peer.Attempts = 10
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/status" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/status")
		}
		received = true
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(StatusResponse{
			LastState: peer.LastState,
		})
	}))
	defer func() { testServer.Close() }()
	peer.Addr = parseURL(testServer.URL)

	//Initialize node
	n := NewNode(nil)
	n.Peers = append(n.Peers, peer)

	//Ping all peers
	go n.PingPeers()

	/*Need to wait for asynchronous processing. This should be enough but could
	cause issues.
	*/
	time.Sleep(100 * time.Millisecond)

	//Check results
	if len(n.Peers) != 0 {
		t.Errorf("len(n.Peers) == %d; want %d", len(n.Peers), 0)
	}

	if peer.Attempts != 10 {
		t.Errorf("peer.Attempts == %d; want %d", peer.Attempts, 10)
	}

	if received {
		t.Errorf("HTTP Server received a request")
	}
}

func TestNodeStateWorker(t *testing.T) {
	state := State{time.Now().UnixNano(), "Test data"}
	n := NewNode(nil)

	go n.stateWorker()
	n.stateChan <- state

	_ = <-n.peerStateChan

	if n.State != state {
		t.Errorf("n.State == %v; want %v", n.State, state)
	}
}

func TestNodeUpdateState(t *testing.T) {
	origState := State{
		Timestamp: time.Now().UnixNano(),
		Data:      "Test data",
	}

	testCases := []struct {
		State    State
		Expected bool
	}{
		{State{1, ""}, false},
		{origState, false},
		{State{origState.Timestamp + 1, "New state"}, true},
		//This test case may fail due to time.Now() resolution being too low on
		//some systems.
		//{State{0, "Data"}, true},
	}

	for i, testCase := range testCases {
		n := NewNode(nil)
		n.UpdateState(origState)
		updated := n.UpdateState(testCase.State)

		if updated != testCase.Expected {
			t.Errorf("n.UpdateState() == %t for test case %d; want %t", updated, i, testCase.Expected)
		}
	}
}
