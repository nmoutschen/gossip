package gossip

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

//Peer represents a peer to this node
type Peer struct {
	//Attempts is the number of unsuccessful attempts to reach the peer
	Attempts int
	//Addr is the peer address, such as IP address and port number
	Addr Addr
	//LastState is the identifier for the last known data state for that peer
	LastState int64
	//LastSuccess is the timestamp in seconds when the last successful contact with the peer was made
	LastSuccess time.Time
	//Peers is the list of peers of this peer
	Peers []*Peer

	//config store the configuration for the peer
	config *Config
}

//NewPeer creates a new Peer
func NewPeer(addr Addr, config *Config) *Peer {
	if config == nil {
		config = DefaultConfig
	}

	p := &Peer{
		Addr:        addr,
		LastSuccess: time.Now(),

		config: config,
	}

	return p
}

/*CanPeer returns whether this peer can connect with the target peer.

This will return false if this is the same peer or if they are already peered
to each other.
*/
func (p *Peer) CanPeer(tgt *Peer) bool {
	if p.Addr == tgt.Addr {
		log.WithFields(log.Fields{"peer": p, "func": "CanPeer"}).Info("Cannot peer with itself")
		return false
	}

	for _, subPeer := range p.Peers {
		if subPeer.Addr == tgt.Addr {
			log.WithFields(log.Fields{"peer": p, "func": "CanPeer"}).Infof("Cannot peer with already peered node %v", tgt.Addr)
			return false
		}
	}

	return true
}

//Get retrieves the latest state from the peer
func (p *Peer) Get() (State, error) {
	res, err := http.Get(p.URL())
	if err != nil {
		log.WithFields(log.Fields{"peer": p, "func": "Get"}).Warnf("Failed to retrieve the latest state with error: %s", err.Error())
		p.UpdateStatus(false)
		return State{}, err
	}
	if res.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{"peer": p, "func": "Get"}).Warnf("Failed to retrieve the latest state with status code %d", res.StatusCode)
		p.UpdateStatus(false)
		return State{}, errors.New("Failed to retrieve the latest state")
	}

	state := &State{}
	if err = json.NewDecoder(res.Body).Decode(state); err != nil {
		log.WithFields(log.Fields{"peer": p, "func": "Get"}).Warn("Failed to decode state")
		p.UpdateStatus(false)
		return State{}, errors.New("Failed to decode state")
	}

	log.WithFields(log.Fields{"peer": p, "func": "Get", "state": state}).Info("Retrieved state")
	p.LastState = state.Timestamp
	p.UpdateStatus(true)
	return *state, nil
}

/*GetPeers retrieves the peers of this peer.
 */
func (p *Peer) GetPeers() ([]Addr, error) {
	res, err := http.Get(p.URL() + "/peers")
	if err != nil {
		log.WithFields(log.Fields{"peer": p, "func": "GetPeers"}).Warnf("Failed to retrieve peers with error: %s", err.Error())
		p.UpdateStatus(false)
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{"peer": p, "func": "GetPeers"}).Warnf("Failed to retrieve peers with status code %d", res.StatusCode)
		p.UpdateStatus(false)
		return nil, errors.New("Failed to retrieve peers")
	}

	peersResponse := &PeersResponse{}
	if err = json.NewDecoder(res.Body).Decode(peersResponse); err != nil {
		log.WithFields(log.Fields{"peer": p, "func": "GetPeers"}).Warn("Failed to decode peers")
		p.UpdateStatus(false)
		return nil, errors.New("Failed to decode peers")
	}

	log.WithFields(log.Fields{"peer": p, "func": "GetPeers"}).Info("Retrieved peers")
	p.UpdateStatus(true)
	return peersResponse.Peers, nil
}

//IsIrrecoverable returns if a peer is considered as permanently unreachable
func (p *Peer) IsIrrecoverable() bool {
	return p.LastSuccess.Add(p.config.Node.MaxPingDelay).Before(time.Now())
}

/*IsCtrlIrrecoverable returns if a peer is considered as permanently
unreachable for a controller node.
*/
func (p *Peer) IsCtrlIrrecoverable() bool {
	return p.LastSuccess.Add(p.config.Controller.MaxScanDelay).Before(time.Now())
}

/*IsUnreachable returns if the peer is considered unreachable

If the number of attempts to contact the peer exceeds the PeerMaxAttempts
threshold, the peer is considered unreachable.
*/
func (p *Peer) IsUnreachable() bool {
	return p.Attempts >= p.config.Peer.MaxAttempts
}

//Ping checks if the peer is reachable and retrieves its status
func (p *Peer) Ping() {
	log.WithFields(log.Fields{"peer": p, "func": "Ping"}).Debug("Ping")

	res, err := http.Get(p.URL() + "/status")
	if err != nil {
		log.WithFields(log.Fields{"peer": p, "func": "Ping"}).Warnf("Ping failed with error: %s", err)
		p.UpdateStatus(false)
		return
	}
	if res.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{"peer": p, "func": "Ping"}).Warnf("Ping failed with status code %d", res.StatusCode)
		p.UpdateStatus(false)
		return
	}

	statusResponse := &StatusResponse{}
	if err := json.NewDecoder(res.Body).Decode(statusResponse); err != nil {
		log.WithFields(log.Fields{"peer": p, "func": "Ping"}).Warn("Failed to decode response")
		p.UpdateStatus(false)
		return
	}

	p.UpdateStatus(true)
	p.LastState = statusResponse.LastState
}

//Send sends a message to a peer
func (p *Peer) Send(state State) {
	//Skip unreachable peers
	if p.IsUnreachable() {
		log.WithFields(log.Fields{"peer": p, "func": "Send", "state": state}).Info("Skip sending state to unreachable peer")
		return
	}

	log.WithFields(log.Fields{"peer": p, "func": "Send", "state": state}).Info("Sending state to peer")

	//Create a JSON document from the state
	jsonVal, err := json.Marshal(state)
	if err != nil {
		log.WithFields(log.Fields{"peer": p, "func": "Send", "state": state}).Error("Failed to marshal state")
		return
	}

	//Try to send the state to the peer
	for i := 0; i <= p.config.Peer.MaxRetries; i++ {
		res, err := http.Post(p.URL(), "application/json", bytes.NewBuffer(jsonVal))
		if err == nil && res.StatusCode == http.StatusOK {
			p.UpdateStatus(true)
			return
		}

		//TODO: add jitter
		time.Sleep(p.config.Peer.BackoffDuration * (1 << i))
	}

	/*Set the status as failed for this message.

	This only sets the status as failed at the end of the loop instead of after
	every attempt, otherwise the node would quickly reach the PeerMaxAttempts
	threshold.
	*/
	log.WithFields(log.Fields{"peer": p, "func": "Send", "state": state}).Warn("Failed to send state")
	p.UpdateStatus(false)
}

//SendPeeringRequest sends a request for peering to a peer
func (p *Peer) SendPeeringRequest(addr Addr) {
	log.WithFields(log.Fields{"peer": p, "func": "SendPeeringRequest"}).Infof("Sending peering request with %v", addr)

	jsonVal, err := json.Marshal(addr)
	if err != nil {
		log.WithFields(log.Fields{"peer": p, "func": "SendPeeringRequest", "addr": addr}).Warn("Failed to marshal addr")
		return
	}

	//Try to send a peering request to the peer
	for i := 0; i <= p.config.Peer.MaxRetries; i++ {
		res, err := http.Post(p.URL()+"/peers", "application/json", bytes.NewBuffer(jsonVal))
		if err == nil && res.StatusCode == http.StatusOK {
			p.UpdateStatus(true)
			return
		}

		//TODO: add jitter
		time.Sleep(p.config.Peer.BackoffDuration * (1 >> i))
	}

	log.WithFields(log.Fields{"peer": p, "func": "SendPeeringRequest"}).Warn("Failed to send peering request")
	p.UpdateStatus(false)
}

//String returns a string representation of the peer
func (p Peer) String() string {
	return p.Addr.String()
}

/*UpdateStatus updates the status of the peer following an action

If true, this means that the node reached out to the peer successfully.
Therefore, we can reset the number of attempts and set the timestamp for the
latest attempt to reach it to now.

Otherwise, increase the attempts counter.

If the attempts counter exceeds the PeerMaxAttempts constant, consider the peer
as failed (see Peer.IsUnreachable).
*/
func (p *Peer) UpdateStatus(ok bool) {
	if ok {
		p.Attempts = 0
		p.LastSuccess = time.Now()
	} else {
		p.Attempts++
		log.WithFields(log.Fields{"peer": p, "func": "UpdateStatus"}).Infof("%d unsuccessful attempts", p.Attempts)
	}
}

//URL returns the complete URL for that peer
func (p *Peer) URL() string {
	return fmt.Sprintf("%s://%s:%d", p.config.Protocol, p.Addr.IP, p.Addr.Port)
}
