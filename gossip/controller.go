package gossip

import (
	"math/rand"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

//Controller represents a controller instance
type Controller struct {
	/*Config contains the configuration (IP address and port) for the
	controller's HTTP server
	*/
	Config Config
	//Peers is a sync.Map[Config]*Peer containing all known peers.
	Peers *sync.Map

	addPeerChan chan Config
}

//NewController creates a new controller instance
func NewController(ip string, port int) *Controller {
	c := &Controller{
		Config: Config{
			IP:   ip,
			Port: port,
		},
		Peers: &sync.Map{},

		addPeerChan: make(chan Config, 8),
	}

	log.WithFields(log.Fields{"controller": c, "func": "NewController"}).Info("Initializing controller")

	return c
}

/*FindClusters look at all peers known to the controller and returns the Config
of peers in separate slices if they are not connected.

Each cluster thus represents a graph of peers that is not connected to the
other clusters. In an ideal scenario, the slice returned should be of length 1.
*/
func (c *Controller) FindClusters() [][]*Peer {
	//Extract all configs from the list of known peers
	var count int
	peers := make(map[*Peer]bool)
	c.Peers.Range(func(_, value interface{}) bool {
		peer, ok := value.(*Peer)
		if !ok {
			log.WithFields(log.Fields{"controller": c, "func": "FindClusters", "peer": peer}).Warn("Failed to assert peer")
			return true
		}

		count++
		peers[peer] = true
		return true
	})

	/*Starting from a random peer, explore peers of peers until we run out of
	peers to explore. In an ideal case, one exploration should traverse all
	peers in the graph, which means that there is only one cluster. However, if
	peers are disconnected (e.g. following a network partition), there could be
	multiple clusters.
	*/
	log.WithFields(log.Fields{"controller": c, "func": "FindClusters"}).Debug("Start cluster discovery")
	var clusters [][]*Peer
	for len(peers) > 0 {
		/*Take the first available peer as a starting point to explore the
		graph.

		As the variable 'peers' is a map, the best way to retrieve a random
		peer is to loop through the keys and stop after one iteration.
		*/
		log.WithFields(log.Fields{"controller": c, "func": "FindClusters"}).Debug("Prepare list of peers to visit")
		var toVisit []*Peer
		for peer := range peers {
			toVisit = append(toVisit, peer)
			delete(peers, peer)
			break
		}
		visited := make(map[*Peer]bool)

		log.WithFields(log.Fields{"controller": c, "func": "FindClusters"}).Debug("Visit peers in cluster")
		for len(toVisit) > 0 {
			/*Take the first available peer in the list of peers to visit.

			Here, we immediately add the peer to the list of visited. As peers
			should have bidirectional relationships: if peer A has peer B in
			its list of peers, then peer B should have peer A in its list of
			peers as well. If we add the peer after scanning its peers, then
			the list of peers to visit will grow until we run out of memory.
			*/
			peer := toVisit[0]
			visited[peer] = true
			delete(peers, peer)
			toVisit = toVisit[1:]

			for _, subPeer := range peer.Peers {
				if _, ok := visited[subPeer]; !ok {
					toVisit = append(toVisit, subPeer)
				}
			}
		}

		/*Extract keys from the map of visited peers and transform it into a
		slice of peers.
		*/
		log.WithFields(log.Fields{"controller": c, "func": "FindClusters"}).Debug("Finalize cluster")
		var cluster []*Peer
		for peer := range visited {
			cluster = append(cluster, peer)
		}
		clusters = append(clusters, cluster)
	}

	return clusters
}

/*FindLowConnectedPeers parse through the list of peers and return those who
have less than PeerMinPeers peers.
*/
func (c *Controller) FindLowConnectedPeers() (lcPeers []*Peer) {
	c.Peers.Range(func(_, value interface{}) bool {
		peer, ok := value.(*Peer)
		if !ok {
			log.WithFields(log.Fields{"controller": c, "func": "FindLowConnectedPeers", "peer": peer}).Warn("Failed to assert peer")
			return true
		}

		if len(peer.Peers) < PeerMinPeers {
			lcPeers = append(lcPeers, peer)
		}
		return true
	})

	return lcPeers
}

/*MergeClusters merge clusters together by sending peering requests to pairs of
nodes across clusters.
*/
func (c *Controller) MergeClusters(clusters [][]*Peer) {
	//Nothing to do if there is zero or one cluster
	if len(clusters) <= 1 {
		log.WithFields(log.Fields{"controller": c, "func": "MergeClusters"}).Debug("No need to merge clusters")
		return
	}

	/*The number of connections cannot be greater than the number of peers in a
	cluster.
	*/
	minPeers := PeerMinPeers
	for _, cluster := range clusters {
		if len(cluster) < minPeers {
			minPeers = len(cluster)
		}
	}

	if minPeers == 0 {
		log.WithFields(log.Fields{"controller": c, "func": "MergeClusters"}).Warn("Minimum number of peers is zero")
		return
	}

	log.WithFields(log.Fields{"controller": c, "func": "MergeClusters"}).Infof("Minimum number of peers is %d", minPeers)

	for oPos := range clusters {
		/*If we're parsing the last cluster, this wraps around to zero. This
		way, we are connecting clusters in a ring.
		*/
		dPos := (oPos + 1) % len(clusters)

		/*Retrieve minPeers random peers from the clusters oPos.

		Here, we use rand.Perm instead of rand.Intn as Intn could produce the
		same number more than once.
		*/
		var origs []*Peer
		for i, o := range rand.Perm(len(clusters[oPos])) {
			if i >= minPeers {
				break
			}
			origs = append(origs, clusters[oPos][o])
		}

		//Send peering requests to oPos on behalf of dPos.
		for i, d := range rand.Perm(len(clusters[dPos])) {
			if i >= minPeers {
				break
			}
			log.WithFields(log.Fields{"controller": c, "func": "MergeClusters"}).Infof("Connecting peers %v and %v", origs[i], clusters[dPos][d])
			origs[i].SendPeeringRequest(clusters[dPos][d].Config)
			/*Manually add the peers together, even though there is no proof
			that the peering was successful at this team. It is necessary to do
			this for the identification of nodes with less than PeerMinPeers
			peers.
			*/
			origs[i].Peers = append(origs[i].Peers, clusters[dPos][d])
			clusters[dPos][d].Peers = append(clusters[dPos][d].Peers, origs[i])
		}
	}
}

/*ScanPeers retrieve the list of peers from peers

When scanning for peers, it's possible that the scanner will discover new
peers. If that's the case, it will add the new peer to the list of peers and
scan it as well.

This function also takes care of removing peers that are irrecoverable.
*/
func (c *Controller) ScanPeers() {
	//Start peer removal temporary worker
	removePeerChan := make(chan Config, 8)
	go c.removePeerWorker(removePeerChan)
	//Closing the channel will automatically stop the worker
	defer close(removePeerChan)

	//Discovery phase
	wg := &sync.WaitGroup{}
	scanned := &sync.Map{}
	c.Peers.Range(func(_, value interface{}) bool {
		peer, ok := value.(*Peer)
		if !ok {
			log.WithFields(log.Fields{"controller": c, "func": "ScanPeers", "peer": peer}).Warn("Failed to assert peer")
			return true
		}

		//Remove irrecoverable peer
		if peer.IsCtrlIrrecoverable() {
			log.WithFields(log.Fields{"controller": c, "func": "ScanPeers", "peer": peer}).Info("Removing irrecoverable peer")
			removePeerChan <- peer.Config
			return true
		}

		//Run scan for the peer
		wg.Add(1)
		go c.scanPeer(peer, scanned, wg)
		return true
	})
	wg.Wait()
}

//Run starts the control instance
func (c *Controller) Run() {
	//Start workers
	go c.addPeerWorker()
	go c.scanWorker()

	//Register handlers
	http.HandleFunc("/peers", c.peersHandler)

	//Run HTTP server
	log.WithFields(log.Fields{"controller": c, "func": "NewController"}).Info("Starting controller")
	log.WithFields(log.Fields{"controller": c, "func": "NewController"}).Fatal(http.ListenAndServe(c.Config.String(), nil))
}

//String returns a string representation of the controller
func (c *Controller) String() string {
	return c.Config.String()
}

//addPeerWorker listens on the addPeerChan channel for new peers
func (c *Controller) addPeerWorker() {
	for {
		config := <-c.addPeerChan
		log.WithFields(log.Fields{"controller": c, "func": "addPeerWorker", "config": config}).Info("Received peering info")

		//Skip known peers
		if _, known := c.Peers.Load(config); known {
			log.WithFields(log.Fields{"controller": c, "func": "addPeerWorker", "config": config}).Debug("Skip known peer")
			continue
		}

		//Add peers to the list of known peers
		peer := NewPeer(config)
		c.Peers.Store(config, peer)
	}
}

//removePeerWorker is a temporary worker to remove irrecoverable peers
func (c *Controller) removePeerWorker(removePeerChan chan Config) {
	for {
		config, open := <-removePeerChan
		if !open {
			log.WithFields(log.Fields{"controller": c, "func": "removePeerWorker"}).Debug("Stopping worker")
			return
		} else if _, ok := c.Peers.Load(config); ok {
			log.WithFields(log.Fields{"controller": c, "func": "removePeerWorker", "config": config}).Info("Removing peer")
			c.Peers.Delete(config)
		} else {
			log.WithFields(log.Fields{"controller": c, "func": "removePeerWorker", "config": config}).Debug("Ignore duplicate peer removal message")
		}
	}
}

//scanWorker periodically scans peers
func (c *Controller) scanWorker() {
	for {
		time.Sleep(ControllerScanDelay)
		log.WithFields(log.Fields{"controller": c, "func": "scanWorker"}).Info("Start scan")

		//Scan all nodes
		c.ScanPeers()

		//Find clusters
		clusters := c.FindClusters()
		if len(clusters) == 1 {
			log.WithFields(log.Fields{"controller": c, "func": "scanWorker"}).Info("Found 1 cluster")
		} else {
			log.WithFields(log.Fields{"controller": c, "func": "scanWorker"}).Warnf("Found %d clusters", len(clusters))
		}

		//Merge clusters
		c.MergeClusters(clusters)

		/*Find peers with less than PeerMinPeers peers and connect them
		together

		To do so, we duplicate peers based on the number of missing
		connections, shuffle the slice, then pair them two by two.

		Peers that did not find a match, either because the number of peers in
		the slice was odd or because they ended up matched with themselves, are
		paired with random peers in the network.
		*/
		var lcPeers []*Peer
		for _, peer := range c.FindLowConnectedPeers() {
			for i := len(peer.Peers); i < PeerMinPeers; i++ {
				lcPeers = append(lcPeers, peer)
			}
		}
		log.WithFields(log.Fields{"controller": c, "func": "scanWorker"}).Infof("Found %d peers with not enough peers", len(lcPeers))
		rand.Shuffle(len(lcPeers), func(i, j int) {
			lcPeers[i], lcPeers[j] = lcPeers[j], lcPeers[i]
		})

		//Left-over peers
		var loPeers []*Peer

		//Make the slice an odd number
		if len(lcPeers)%2 == 1 {
			loPeers = append(loPeers, lcPeers[0])
			lcPeers = lcPeers[1:]
		}

		//Match peers two-by-two
		for i := 0; i < len(lcPeers); i += 2 {
			if lcPeers[i].Config == lcPeers[i+1].Config {
				log.WithFields(log.Fields{"controller": c, "func": "scanWorker", "peer": lcPeers[i]}).Info("Found duplicate peer")
				loPeers = append(loPeers, lcPeers[i], lcPeers[i+1])
			} else {
				log.WithFields(log.Fields{"controller": c, "func": "scanWorker"}).Infof("Connecting peers %v and %v", lcPeers[i], lcPeers[i+1])
				lcPeers[i].SendPeeringRequest(lcPeers[i+1].Config)
			}
		}

		/*Matching left-over peers with a random know peer

		There is still a potential for collision (peer matching with itself),
		however, we voluntarily disregard it at this step.
		*/
		if len(loPeers) > 0 {
			//Transform c.Peers into a slice
			var peers []*Peer
			c.Peers.Range(func(_, value interface{}) bool {
				peer, ok := value.(*Peer)
				if !ok {
					log.WithFields(log.Fields{"controller": c, "func": "ScanPeers", "peer": peer}).Warn("Failed to assert peer")
					return true
				}
				peers = append(peers, peer)
				return true
			})

			for _, peer := range loPeers {
				peer.SendPeeringRequest(peers[rand.Intn(len(peers))].Config)
			}
		}
	}
}

//scanPeer scans a single peer or skip it if it in the scanned map
func (c *Controller) scanPeer(peer *Peer, scanned *sync.Map, wg *sync.WaitGroup) {
	defer wg.Done()

	//This peer is already scanned
	if _, ok := scanned.Load(peer.Config); ok {
		log.WithFields(log.Fields{"controller": c, "func": "scanPeer", "peer": peer}).Debug("Skipping scanned peer")
		return
	}

	//Retrieve the list of peers of this peer
	log.WithFields(log.Fields{"controller": c, "func": "scanPeer", "peer": peer}).Info("Scanning peer")
	peers, err := peer.GetPeers()
	scanned.Store(peer.Config, peer)
	if err != nil {
		log.WithFields(log.Fields{"controller": c, "func": "scanPeer", "peer": peer}).Info("Failed to scan peer")
		return
	}

	//Reset the list of peers of the peer
	peer.Peers = nil

	//Parse peers of the peer
	for _, config := range peers {
		//Load the peer
		iSubPeer, _ := c.Peers.LoadOrStore(config, NewPeer(config))
		subPeer, ok := iSubPeer.(*Peer)
		if !ok {
			log.WithFields(log.Fields{"controller": c, "func": "scanPeer", "peer": peer, "subPeer": subPeer}).Warn("Failed to assert subPeer")
			continue
		}

		//Add the sub-peer to the list of peers
		peer.Peers = append(peer.Peers, subPeer)

		//Schedule the peer for scanning if it hasn't already been scanned
		if _, ok := scanned.Load(config); !ok {
			log.WithFields(log.Fields{"controller": c, "func": "scanPeer", "peer": peer, "subPeer": subPeer}).Info("Adding subPeer for scanning")
			wg.Add(1)
			go c.scanPeer(subPeer, scanned, wg)
		}
	}
}
