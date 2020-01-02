package gossip

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestControllerAddPeerWorker(t *testing.T) {
	addr := Addr{"127.0.0.1", 8080}
	c := NewController(Addr{"127.0.0.1", 7080}, nil)
	go c.addPeerWorker()

	//Add a peer
	c.addPeerChan <- addr
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Peers.Load(addr); !ok {
		t.Errorf("Peer %v not found in c.Peers", addr)
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
	c.addPeerChan <- addr
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Peers.Load(addr); !ok {
		t.Errorf("Peer %v not found in c.Peers", addr)
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
		NewPeer(Addr{"127.0.0.1", 8080}, nil),
		NewPeer(Addr{"127.0.0.1", 8081}, nil),
		NewPeer(Addr{"127.0.0.1", 8082}, nil),
		NewPeer(Addr{"127.0.0.1", 8083}, nil),
		NewPeer(Addr{"127.0.0.1", 8084}, nil),
		NewPeer(Addr{"127.0.0.1", 8085}, nil),
		NewPeer(Addr{"127.0.0.1", 8086}, nil),
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
	c := NewController(Addr{"127.0.0.1", 7080}, nil)
	for _, peer := range peers {
		c.Peers.Store(peer.Addr, peer)
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
		NewPeer(Addr{"127.0.0.1", 8080}, nil),
		NewPeer(Addr{"127.0.0.1", 8081}, nil),
		NewPeer(Addr{"127.0.0.1", 8082}, nil),
		NewPeer(Addr{"127.0.0.1", 8083}, nil),
		NewPeer(Addr{"127.0.0.1", 8084}, nil),
		NewPeer(Addr{"127.0.0.1", 8085}, nil),
	}

	//Create network structure
	peers[0].Peers = []*Peer{peers[1], peers[2]}
	peers[1].Peers = []*Peer{peers[0], peers[2]}
	peers[2].Peers = []*Peer{peers[0], peers[1]}
	peers[3].Peers = []*Peer{peers[4], peers[5]}
	peers[4].Peers = []*Peer{peers[3], peers[5]}
	peers[5].Peers = []*Peer{peers[3], peers[4]}

	//Create controller
	c := NewController(Addr{"127.0.0.1", 7080}, nil)
	for _, peer := range peers {
		c.Peers.Store(peer.Addr, peer)
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

func TestControllerFindLowPeersLine(t *testing.T) {
	//Create peers
	peers := []*Peer{
		NewPeer(Addr{"127.0.0.1", 8080}, nil),
		NewPeer(Addr{"127.0.0.1", 8081}, nil),
		NewPeer(Addr{"127.0.0.1", 8082}, nil),
		NewPeer(Addr{"127.0.0.1", 8083}, nil),
		NewPeer(Addr{"127.0.0.1", 8084}, nil),
	}

	//Create network structure
	peers[0].Peers = []*Peer{peers[1]}
	peers[1].Peers = []*Peer{peers[0], peers[2]}
	peers[2].Peers = []*Peer{peers[1], peers[3]}
	peers[3].Peers = []*Peer{peers[2], peers[4]}
	peers[4].Peers = []*Peer{peers[3]}

	//Create controller
	c := NewController(Addr{"127.0.0.1", 7080}, nil)
	for _, peer := range peers {
		c.Peers.Store(peer.Addr, peer)
	}

	//Analyze peers
	lcPeers := c.FindLowPeers()

	if len(lcPeers) != len(peers) {
		t.Errorf("len(lcPeers) == %d; want %d", len(lcPeers), len(peers))
	}

	//Connect peers
	c.ConnectLowPeers()

	/*Check connectivity

	Due to the random nature of ConnectLowPeers(), it's possible that this will
	fail. Therefore, log instead of mark as error.
	*/
	for _, peer := range peers {
		if len(peer.Peers) < DefaultConfig.Controller.MinPeers {
			t.Logf("len(peer.Peers) == %d; want >= %d", len(peer.Peers), DefaultConfig.Controller.MinPeers)
		}
	}
}

func TestControllerFindLowPeersStar(t *testing.T) {
	//Create peers
	peers := []*Peer{
		NewPeer(Addr{"127.0.0.1", 8080}, nil),
		NewPeer(Addr{"127.0.0.1", 8081}, nil),
		NewPeer(Addr{"127.0.0.1", 8082}, nil),
		NewPeer(Addr{"127.0.0.1", 8083}, nil),
		NewPeer(Addr{"127.0.0.1", 8084}, nil),
	}

	//Create network structure
	peers[0].Peers = []*Peer{peers[1], peers[2], peers[3], peers[4]}
	peers[1].Peers = []*Peer{peers[0]}
	peers[2].Peers = []*Peer{peers[0]}
	peers[3].Peers = []*Peer{peers[0]}
	peers[4].Peers = []*Peer{peers[0]}

	//Create controller
	c := NewController(Addr{"127.0.0.1", 7080}, nil)
	for _, peer := range peers {
		c.Peers.Store(peer.Addr, peer)
	}

	//Analyze peers
	lcPeers := c.FindLowPeers()

	if len(lcPeers) != len(peers)-1 {
		t.Errorf("len(lcPeers) == %d; want %d", len(lcPeers), len(peers)-1)
	}
	for _, peer := range lcPeers {
		if peer.Addr == peers[0].Addr {
			t.Errorf("Found %v in lcPeers", peer)
		}
	}

	//Connect peers
	c.ConnectLowPeers()

	/*Check connectivity

	Due to the random nature of ConnectLowPeers(), it's possible that this will
	fail. Therefore, log instead of mark as error.
	*/
	for _, peer := range peers {
		if len(peer.Peers) < DefaultConfig.Controller.MinPeers {
			t.Logf("len(peer.Peers) == %d; want >= %d", len(peer.Peers), DefaultConfig.Controller.MinPeers)
		}
	}
}

func TestControllerFindLowPeersFull(t *testing.T) {
	//Create peers
	peers := []*Peer{
		NewPeer(Addr{"127.0.0.1", 8080}, nil),
		NewPeer(Addr{"127.0.0.1", 8081}, nil),
		NewPeer(Addr{"127.0.0.1", 8082}, nil),
		NewPeer(Addr{"127.0.0.1", 8083}, nil),
	}

	//Create network structure
	peers[0].Peers = []*Peer{peers[1], peers[2], peers[3]}
	peers[1].Peers = []*Peer{peers[0], peers[2], peers[3]}
	peers[2].Peers = []*Peer{peers[0], peers[1], peers[3]}
	peers[3].Peers = []*Peer{peers[0], peers[1], peers[2]}

	//Create controller
	c := NewController(Addr{"127.0.0.1", 7080}, nil)
	for _, peer := range peers {
		c.Peers.Store(peer.Addr, peer)
	}

	//Analyze peers
	lcPeers := c.FindLowPeers()

	if len(lcPeers) != 0 {
		t.Errorf("len(lcPeers) == %d; want %d", len(lcPeers), 0)
	}

	//Connect peers, this should not do anything
	c.ConnectLowPeers()

	/*Check connectivity

	Due to the random nature of ConnectLowPeers(), it's possible that this will
	fail. Therefore, log instead of mark as error.
	*/
	for _, peer := range peers {
		if len(peer.Peers) < DefaultConfig.Controller.MinPeers {
			t.Logf("len(peer.Peers) == %d; want >= %d", len(peer.Peers), DefaultConfig.Controller.MinPeers)
		}
	}
}

func TestControllerRemovePeerWorker(t *testing.T) {
	addr := Addr{"127.0.0.1", 8080}
	c := NewController(Addr{"127.0.0.1", 7080}, nil)
	removePeerChan := make(chan Addr)
	defer close(removePeerChan)
	c.Peers.Store(addr, NewPeer(addr, nil))

	go c.removePeerWorker(removePeerChan)
	removePeerChan <- addr
	//TODO: Find a better solution to handle asynchronous operations.
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.Peers.Load(addr); ok {
		t.Errorf("Peer %v found in c.Peers", addr)
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

func TestControllerScanPeers(t *testing.T) {
	//Setup
	peer := NewPeer(Addr{"127.0.0.1", 80}, nil)
	c := NewController(Addr{"127.0.0.1", 7080}, nil)
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
			Peers: []Addr{peer.Addr},
		})
	}))
	defer func() { testServer.Close() }()
	peer.Addr = parseURL(testServer.URL)
	c.Peers.Store(peer.Addr, peer)

	c.ScanPeers()

	if !received {
		t.Errorf("HTTP Server never received a request")
	}

	if len(peer.Peers) != 1 {
		t.Errorf("len(peer.Peers) == %d; want %d", len(peer.Peers), 1)
	}
	var lenPeers int
	c.Peers.Range(func(_, _ interface{}) bool {
		lenPeers++
		return true
	})
	if lenPeers != 1 {
		t.Errorf("lenPeers == %d; want %d", lenPeers, 1)
	}
}
