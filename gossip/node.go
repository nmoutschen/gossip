package gossip

import (
	"math/rand"
	"net/http"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
)

//Node represents the management unit for this node
type Node struct {
	Addr  Addr    `json:"addr"`
	Peers []*Peer `json:"peers"`
	State State   `json:"state"`

	//fetchStateChan is a channel to force fetch updates from other peers
	fetchStateChan chan *Peer
	//addPeerChan is a channel to receive peering requests
	addPeerChan chan Addr
	//peerStateChan is a channel to receive state updates that need to be propagated to peers
	peerStateChan chan State
	//stateChan is a channel to receive state updates
	stateChan chan State
}

//NewNode creates a new Node
func NewNode(ip string, port int) *Node {
	n := &Node{
		Addr: Addr{
			IP:   ip,
			Port: port,
		},

		fetchStateChan: make(chan *Peer, 8),
		addPeerChan:    make(chan Addr, 8),
		peerStateChan:  make(chan State, 8),
		stateChan:      make(chan State, 8),
	}

	log.WithFields(log.Fields{"node": n, "func": "NewNode"}).Info("Initializing node")

	return n
}

//AddPeer adds a new peer if there are no known peers with the same Addr
func (n *Node) AddPeer(addr Addr) {
	log.WithFields(log.Fields{"node": n, "addr": addr, "func": "addPeerWorker"}).Info("Received peering request")

	//Skip if self.
	if addr == n.Addr {
		log.WithFields(log.Fields{"node": n, "addr": addr, "func": "addPeerWorker"}).Info("Skip self-peering request")
		return
	}

	//Skip if already known.
	if _, found := n.FindPeer(addr); found {
		log.WithFields(log.Fields{"node": n, "addr": addr, "func": "addPeerWorker"}).Info("Skip known peer")
		return
	}

	//Add the peer to the list of known peers.
	peer := &Peer{
		Addr: addr,
	}
	n.Peers = append(n.Peers, peer)

	//Send a peering request.
	go peer.SendPeeringRequest(n.Addr)
}

/*FindPeer looks up known peers and returns if there is a peer matching the
Addr provided.
*/
func (n *Node) FindPeer(addr Addr) (int, bool) {
	for pos, peer := range n.Peers {
		if peer.Addr == addr {
			return pos, true
		}
	}
	return -1, false
}

/*PeerSendState sends a state to peers.

If the number of peers known to this node is greater than PeerMaxRecipients,
this function takes PeerMaxRecipients peers at random and sends the state to
only those peers.
*/
func (n *Node) PeerSendState(state State) int {
	var peers []*Peer
	//If there are too many peers, need to limit to PeerMaxRecipients peers
	//chosen randomly.
	if len(n.Peers) > PeerMaxRecipients {
		for _, i := range rand.Perm(len(n.Peers)) {
			peers = append(peers, n.Peers[i])
			if len(peers) >= PeerMaxRecipients {
				break
			}
		}
	} else {
		peers = n.Peers
	}
	log.WithFields(log.Fields{"node": n, "state": state, "func": "peerSendStateWorker"}).Infof("Sending state update to %d/%d peers", len(peers), len(n.Peers))

	for _, peer := range peers {
		go peer.Send(state)
	}

	return len(peers)
}

//PingPeers ping all peers known to the node
func (n *Node) PingPeers() {
	var peersToRemove []int
	for i, peer := range n.Peers {
		/*If the peer is irrecoverable, mark it for removal from the list
		of known peers.
		*/
		if peer.IsIrrecoverable() {
			peersToRemove = append(peersToRemove, i)
			continue
		}

		/*Despite running a goroutine in a loop, we don't need to wait for
		completion here as this routine doesn't have any side effect to the
		pingWorker.

		peersToRemove is filled before this routine, therefore it is
		already ready to be processed, without having to wait for the
		goroutines to finish their execution.
		*/
		go func(n *Node, peer *Peer) {
			peer.Ping()
			if peer.LastState > n.State.Timestamp {
				n.fetchStateChan <- peer
			}
		}(n, peer)
	}

	/* Every time we remove a peer 'i' from n.Peers, the new index of peers
	greater than i is reduced by 1. If peersToRemove is not sorted, an item
	in the middle could be remove, leaving items on the left and right in
	to be removed. If that's the case, it will be hard to know if we need
	to decrease the index by the number of items remove.

	By sorting the slice, we ensure consistent behavior (always needing to
	substract the position by the number of items removed).
	*/
	sort.Ints(peersToRemove)

	//Process irrecoverable peers.
	for c, i := range peersToRemove {
		log.WithFields(log.Fields{"node": n, "func": "PingPeers", "peer": n.Peers[i]}).Info("Removing irrecoverable peer")
		n.Peers = append(n.Peers[:i-c], n.Peers[i-c+1:]...)
	}
}

//Run start the workers and run an HTTP server
func (n *Node) Run() {
	//Start workers
	go n.addPeerWorker()
	go n.fetchStateWorker()
	go n.peerSendStateWorker()
	go n.pingWorker()
	go n.stateWorker()

	//Register handlers
	http.HandleFunc("/", n.rootHandler)
	http.HandleFunc("/status", n.statusHandler)
	http.HandleFunc("/peers", n.peersHandler)

	//Run HTTP server
	log.WithFields(log.Fields{"node": n, "func": "Run"}).Info("Starting node")
	log.WithFields(log.Fields{"node": n, "func": "Run"}).Fatal(http.ListenAndServe(n.Addr.String(), nil))
}

//String returns a string representation of the address of the node
func (n Node) String() string {
	return n.Addr.String()
}

/*UpdateState updates the internal state if it is older than the proposed
state.

This returns true if the internal state has been updated.
*/
func (n *Node) UpdateState(state State) bool {
	//New state received from the end-user
	if state.Timestamp == 0 {
		state.Timestamp = time.Now().UnixNano()
	}

	switch {
	case state.Timestamp < n.State.Timestamp:
		log.WithFields(log.Fields{"node": n, "state": state, "func": "stateWorker"}).Info("Received obsolete state")
	case state.Timestamp == n.State.Timestamp:
		log.WithFields(log.Fields{"node": n, "state": state, "func": "stateWorker"}).Info("Received known state")
	case state.Timestamp > n.State.Timestamp:
		log.WithFields(log.Fields{"node": n, "state": state, "func": "stateWorker"}).Info("Received new state")
		n.State = state
		return true
	}

	return false
}

/*fetchStateWorker waits for peers on the n.fetchStateChan channel and
retrieves the last state from those peers, then sends the state to the
n.stateChan channel.
*/
func (n *Node) fetchStateWorker() {
	for {
		peer := <-n.fetchStateChan
		log.WithFields(log.Fields{"node": n, "peer": peer, "func": "fetchStateWorker"}).Info("Fetching latest state")

		/*It's possible that we have already fetched the latest state from the
		peer. If that's the case, ignore, as this would generate a useless
		GET request to the peer.
		*/
		if peer.LastState <= n.State.Timestamp {
			log.WithFields(log.Fields{"node": n, "peer": peer, "func": "fetchStateWorker"}).Info("Skip fetching state")
			continue
		}

		if state, err := peer.Get(); err == nil {
			n.stateChan <- state
		}
	}
}

/*addPeerWorker waits for new Addrs on the n.addPeerChan channel and process
them.

If the node is known, there is no need to do anything, so the message can
safely be ignored.

If the node is new, we add it to the list of known peers and send a peering
request back to the node. Since the node who sent the request knows about this
node, they will ignore the request.

This behavior prevents infinite loops of peering requests between two nodes,
but allow the control plane to send the same request as any other node in the
network.
*/
func (n *Node) addPeerWorker() {
	for {
		addr := <-n.addPeerChan
		n.AddPeer(addr)
	}
}

/*peerSendStateWorker waits for new states on the n.peerStateChan channel and
sends the state to all known peers.
*/
func (n *Node) peerSendStateWorker() {
	for {
		state := <-n.peerStateChan
		n.PeerSendState(state)
	}
}

//pingWorker checks the status of all peers at regular interval.
func (n *Node) pingWorker() {
	for {
		time.Sleep(PingDelay)
		n.PingPeers()
	}
}

/*stateWorker waits for new states on the n.stateChan channel and process
them.
*/
func (n *Node) stateWorker() {
	for {
		state := <-n.stateChan

		if n.UpdateState(state) {
			n.peerStateChan <- state
		}
	}
}
