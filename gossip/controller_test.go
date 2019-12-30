package gossip

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestControllerAddPeerWorker(t *testing.T) {
	config := Config{"127.0.0.1", 8080}
	c := NewController("127.0.0.1", 7080)
	go c.addPeerWorker()

	//Add a peer
	c.addPeerChan <- config
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Peers.Load(config); !ok {
		t.Errorf("Peer %v not found in c.Peers", config)
	}
	var pLength int
	c.Peers.Range(func(_, _ interface{}) bool {
		pLength++
		return true
	})
	if pLength != 1 {
		t.Errorf("pLength == %d; want %d", pLength, 1)
	}

	//Test idempotence, a peer should not be added twice
	c.addPeerChan <- config
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Peers.Load(config); !ok {
		t.Errorf("Peer %v not found in c.Peers", config)
	}
	pLength = 0
	c.Peers.Range(func(_, _ interface{}) bool {
		pLength++
		return true
	})
	if pLength != 1 {
		t.Errorf("pLength == %d; want %d", pLength, 1)
	}
}

func TestControllerFindClustersStar(t *testing.T) {
	//Create peers
	peers := []*Peer{
		NewPeer(Config{"127.0.0.1", 8080}),
		NewPeer(Config{"127.0.0.1", 8081}),
		NewPeer(Config{"127.0.0.1", 8082}),
		NewPeer(Config{"127.0.0.1", 8083}),
		NewPeer(Config{"127.0.0.1", 8084}),
		NewPeer(Config{"127.0.0.1", 8085}),
		NewPeer(Config{"127.0.0.1", 8086}),
	}

	//Create network structure
	peers[3].Peers = []*Peer{
		peers[0],
		peers[1],
		peers[2],
		peers[4],
		peers[5],
		peers[6],
	}
	peers[0].Peers = []*Peer{peers[3]}
	peers[1].Peers = []*Peer{peers[3]}
	peers[2].Peers = []*Peer{peers[3]}
	peers[4].Peers = []*Peer{peers[3]}
	peers[5].Peers = []*Peer{peers[3]}
	peers[6].Peers = []*Peer{peers[3]}

	//Create controller
	c := NewController("127.0.0.1", 7080)
	for _, peer := range peers {
		c.Peers.Store(peer.Config, peer)
	}

	//Analyze clusters
	clusters := c.FindClusters()

	if len(clusters) != 1 {
		t.Errorf("len(clusters) == %d; want %d", len(clusters), 1)
		return
	}

	if len(clusters[0]) != len(peers) {
		t.Errorf("len(clusters[0]) == %d; want %d", len(clusters[0]), len(peers))
	}
}

func TestControllerFindClustersTwoTriangles(t *testing.T) {
	//Create peers
	peers := []*Peer{
		NewPeer(Config{"127.0.0.1", 8080}),
		NewPeer(Config{"127.0.0.1", 8081}),
		NewPeer(Config{"127.0.0.1", 8082}),
		NewPeer(Config{"127.0.0.1", 8083}),
		NewPeer(Config{"127.0.0.1", 8084}),
		NewPeer(Config{"127.0.0.1", 8085}),
	}

	//Create network structure
	peers[0].Peers = []*Peer{peers[1], peers[2]}
	peers[1].Peers = []*Peer{peers[0], peers[2]}
	peers[2].Peers = []*Peer{peers[0], peers[1]}
	peers[3].Peers = []*Peer{peers[4], peers[5]}
	peers[4].Peers = []*Peer{peers[3], peers[5]}
	peers[5].Peers = []*Peer{peers[3], peers[4]}

	//Create controller
	c := NewController("127.0.0.1", 7080)
	for _, peer := range peers {
		c.Peers.Store(peer.Config, peer)
	}

	//Analyze clusters
	clusters := c.FindClusters()

	if len(clusters) != 2 {
		t.Errorf("len(clusters) == %d; want %d", len(clusters), 2)
		return
	}

	if len(clusters[0]) != 3 {
		t.Errorf("len(clusters[0]) == %d; want %d", len(clusters[0]), 3)
	}
	if len(clusters[1]) != 3 {
		t.Errorf("len(clusters[1]) == %d; want %d", len(clusters[1]), 3)
	}

	//Merge clusters
	c.MergeClusters(clusters)
	clusters = c.FindClusters()

	if len(clusters) != 1 {
		t.Errorf("len(clusters) == %d after merge; want %d", len(clusters), 1)
		return
	}
	if len(clusters[0]) != len(peers) {
		t.Errorf("len(clusters[0]) == %d after merge; want %d", len(clusters[0]), len(peers))
	}
}

func TestControllerFindLowConnectedPeersLine(t *testing.T) {
	//Create peers
	peers := []*Peer{
		NewPeer(Config{"127.0.0.1", 8080}),
		NewPeer(Config{"127.0.0.1", 8081}),
		NewPeer(Config{"127.0.0.1", 8082}),
		NewPeer(Config{"127.0.0.1", 8083}),
		NewPeer(Config{"127.0.0.1", 8084}),
	}

	//Create network structure
	peers[0].Peers = []*Peer{peers[1]}
	peers[1].Peers = []*Peer{peers[0], peers[2]}
	peers[2].Peers = []*Peer{peers[1], peers[3]}
	peers[3].Peers = []*Peer{peers[2], peers[4]}
	peers[4].Peers = []*Peer{peers[3]}

	//Create controller
	c := NewController("127.0.0.1", 7080)
	for _, peer := range peers {
		c.Peers.Store(peer.Config, peer)
	}

	//Analyze peers
	lcPeers := c.FindLowConnectedPeers()

	if len(lcPeers) != len(peers) {
		t.Errorf("len(lcPeers) == %d; want %d", len(lcPeers), len(peers))
	}
}

func TestControllerFindLowConnectedPeersStar(t *testing.T) {
	//Create peers
	peers := []*Peer{
		NewPeer(Config{"127.0.0.1", 8080}),
		NewPeer(Config{"127.0.0.1", 8081}),
		NewPeer(Config{"127.0.0.1", 8082}),
		NewPeer(Config{"127.0.0.1", 8083}),
		NewPeer(Config{"127.0.0.1", 8084}),
	}

	//Create network structure
	peers[0].Peers = []*Peer{peers[1], peers[2], peers[3], peers[4]}
	peers[1].Peers = []*Peer{peers[0]}
	peers[2].Peers = []*Peer{peers[0]}
	peers[3].Peers = []*Peer{peers[0]}
	peers[4].Peers = []*Peer{peers[0]}

	//Create controller
	c := NewController("127.0.0.1", 7080)
	for _, peer := range peers {
		c.Peers.Store(peer.Config, peer)
	}

	//Analyze peers
	lcPeers := c.FindLowConnectedPeers()

	if len(lcPeers) != len(peers)-1 {
		t.Errorf("len(lcPeers) == %d; want %d", len(lcPeers), len(peers)-1)
	}
	for _, peer := range lcPeers {
		if peer.Config == peers[0].Config {
			t.Errorf("Found %v in lcPeers", peer)
		}
	}
}

func TestControllerFindLowConnectedPeersFull(t *testing.T) {
	//Create peers
	peers := []*Peer{
		NewPeer(Config{"127.0.0.1", 8080}),
		NewPeer(Config{"127.0.0.1", 8081}),
		NewPeer(Config{"127.0.0.1", 8082}),
		NewPeer(Config{"127.0.0.1", 8083}),
	}

	//Create network structure
	peers[0].Peers = []*Peer{peers[1], peers[2], peers[3]}
	peers[1].Peers = []*Peer{peers[0], peers[2], peers[3]}
	peers[2].Peers = []*Peer{peers[0], peers[1], peers[3]}
	peers[3].Peers = []*Peer{peers[0], peers[1], peers[2]}

	//Create controller
	c := NewController("127.0.0.1", 7080)
	for _, peer := range peers {
		c.Peers.Store(peer.Config, peer)
	}

	//Analyze peers
	lcPeers := c.FindLowConnectedPeers()

	if len(lcPeers) != 0 {
		t.Errorf("len(lcPeers) == %d; want %d", len(lcPeers), 0)
	}
}

func TestControllerRemovePeerWorker(t *testing.T) {
	config := Config{"127.0.0.1", 8080}
	c := NewController("127.0.0.1", 7080)
	removePeerChan := make(chan Config)
	defer close(removePeerChan)
	c.Peers.Store(config, NewPeer(config))

	go c.removePeerWorker(removePeerChan)
	removePeerChan <- config
	//TODO: Find a better solution to handle asynchronous operations.
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Peers.Load(config); ok {
		t.Errorf("Peer %v found in c.Peers", config)
	}
	var pLength int
	c.Peers.Range(func(_, _ interface{}) bool {
		pLength++
		return true
	})
	if pLength != 0 {
		t.Errorf("pLength == %d; want %d", pLength, 0)
	}
}

func TestControllerScanPeer(t *testing.T) {
	//Setup
	peer := NewPeer(Config{"127.0.0.1", 80})
	c := NewController("127.0.0.1", 7080)
	scanned := &sync.Map{}
	wg := &sync.WaitGroup{}
	var received bool
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("r.Method == %s; want %s", r.Method, "GET")
		}
		if r.URL.Path != "/peers" {
			t.Errorf("r.URL.PATH == %s; want %s", r.URL.Path, "/peers")
		}
		received = true
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(PeersResponse{
			Peers: []Config{peer.Config},
		})
	}))
	defer func() { testServer.Close() }()
	peer.Config = parseURL(testServer.URL)
	c.Peers.Store(peer.Config, peer)

	wg.Add(1)
	go c.scanPeer(peer, scanned, wg)
	wg.Wait()

	if !received {
		t.Errorf("HTTP Server never received a request")
	}
	if _, ok := scanned.Load(peer.Config); !ok {
		t.Errorf("Peer %v not scanned", peer)
	}

	if len(peer.Peers) != 1 {
		t.Errorf("len(peer.Peers) == %d; want %d", len(peer.Peers), 1)
	}
}
