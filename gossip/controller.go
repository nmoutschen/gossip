package gossip

import (
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

//Controller represents a controller instance
type Controller struct {
	Config Config
	Peers  *sync.Map

	addPeerChan chan Config
}

//NewController creates a new control instance
func NewController(ip string, port int) *Controller {
	c := &Controller{
		Config: Config{
			IP:   ip,
			Port: port,
		},
		Peers: &sync.Map{},

		addPeerChan: make(chan Config),
	}

	log.WithFields(log.Fields{"controller": c, "func": "NewController"}).Info("Initializing controller")

	return c
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

	//Discovery phase
	wg := &sync.WaitGroup{}
	scanned := &sync.Map{}
	c.Peers.Range(func(_ interface{}, value interface{}) bool {
		peer, ok := value.(*Peer)
		if !ok {
			log.WithFields(log.Fields{"controller": c, "func": "ScanPeers", "peer": peer}).Warn("Failed to assert peer")
			return true
		}

		//Remove irrecoverable peer
		if peer.IsIrrecoverable() {
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

	//Closing the channel will automatically stop the worker
	close(removePeerChan)
}

//Run starts the control instance
func (c *Controller) Run() {
	//Start workers
	go c.addPeerWorker()
	go c.scanWorker()

	//Register handlers
	http.HandleFunc("/nodes", c.peersHandler)

	//Run HTTP server
	log.WithFields(log.Fields{"controller": c, "func": "NewController"}).Info("Starting controller")
	log.WithFields(log.Fields{"controller": c, "func": "NewController"}).Fatal(http.ListenAndServe(c.Config.String(), nil))
}

//addPeerWorker listens on the addPeerChan channel for new peers
func (c *Controller) addPeerWorker() {
	for {
		config := <-c.addPeerChan
		log.WithFields(log.Fields{"controller": c, "func": "addPeerWorker", "config": config}).Info("Received peering request")

		//Skip known peers
		if _, known := c.Peers.Load(config); known {
			log.WithFields(log.Fields{"controller": c, "func": "addPeerWorker", "config": config}).Debug("Skip known peer")
			continue
		}

		//Add peers to the list of known peers
		peer := &Peer{
			Config: config,
		}
		c.Peers.Store(config, peer)
	}
}

//removePeerWorker is a temporary worker to remove irrecoverable peers
func (c *Controller) removePeerWorker(removePeerChan chan Config) {
	for {
		select {
		case config := <-removePeerChan:
			if _, ok := c.Peers.Load(config); ok {
				log.WithFields(log.Fields{"controller": c, "func": "removePeerWorker", "config": config}).Info("Removing peer")
				c.Peers.Delete(config)
			} else {
				log.WithFields(log.Fields{"controller": c, "func": "removePeerWorker", "config": config}).Debug("Ignore duplicate peer removal message")
			}
		case <-removePeerChan:
			log.WithFields(log.Fields{"controller": c, "func": "removePeerWorker"}).Debug("Stopping worker")
			return
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

		/* TODO:
		* find peers with less than PeerMinPeers peers
		* find clusters
		* add peerings to repair separate clusters and peers with less
		  than PeerMinPeers peers
		*/
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
