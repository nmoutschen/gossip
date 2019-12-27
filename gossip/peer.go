package gossip

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

//Peer represents a peer to this node
type Peer struct {
	LastState   int64
	LastSuccess int64
	Attempts    int
	Config      Config
}

//NewPeer creates a new Peer
func NewPeer(config Config) *Peer {
	p := &Peer{
		LastSuccess: time.Now().Unix(),
		Config:      config,
	}

	return p
}

//Addr returns the address of the peer with the protocol
func (p *Peer) Addr() string {
	return fmt.Sprintf("%s://%s:%d", Protocol, p.Config.IP, p.Config.Port)
}

//Get retrieves the latest state from the peer
func (p *Peer) Get() (State, error) {
	res, err := http.Get(p.Addr())
	if err != nil || res.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{"peer": p, "func": "Get"}).Warn("Failed to retrieve the latest state")
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
	return *state, nil
}

//IsIrrecoverable returns if a peer is considered as permanently unreachable
func (p *Peer) IsIrrecoverable() bool {
	return p.LastSuccess+PeerMaxPingDelay < time.Now().Unix()
}

//Ping checks if the peer is reachable and retrieves its status
func (p *Peer) Ping() {
	log.WithFields(log.Fields{"peer": p, "func": "Ping"}).Debug("Ping")

	res, err := http.Get(p.Addr() + "/status")
	if err != nil || res.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{"peer": p, "func": "Ping"}).Warn("Ping failed")
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
	for i := 0; i <= PeerMaxRetries; i++ {
		res, err := http.Post(p.Addr(), "application/json", bytes.NewBuffer(jsonVal))
		if err == nil && res.StatusCode == http.StatusOK {
			p.UpdateStatus(true)
			return
		}

		//TODO: add jitter
		time.Sleep(PeerBackoffDuration * (1 >> i))
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
func (p *Peer) SendPeeringRequest(config Config) {
	log.WithFields(log.Fields{"peer": p, "func": "SendPeeringRequest"}).Info("Sending peering request")

	jsonVal, err := json.Marshal(config)
	if err != nil {
		log.WithFields(log.Fields{"peer": p, "func": "SendPeeringRequest", "config": config}).Warn("Failed to marshal config")
		return
	}

	//Try to send a peering request to the peer
	for i := 0; i <= PeerMaxRetries; i++ {
		res, err := http.Post(p.Addr()+"/peers", "application/json", bytes.NewBuffer(jsonVal))
		if err == nil && res.StatusCode == http.StatusOK {
			p.UpdateStatus(true)
			return
		}

		//TODO: add jitter
		time.Sleep(PeerBackoffDuration * (1 >> i))
	}

	log.WithFields(log.Fields{"peer": p, "func": "SendPeeringRequest"}).Warn("Failed to send peering request")
	p.UpdateStatus(false)
}

//String returns a string representation of the peer
func (p Peer) String() string {
	return p.Config.String()
}

/*IsUnreachable returns if the peer is considered unreachable

If the number of attempts to contact the peer exceeds the PeerMaxAttempts
threshold, the peer is considered unreachable.
*/
func (p *Peer) IsUnreachable() bool {
	return p.Attempts >= PeerMaxAttempts
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
		p.LastSuccess = time.Now().Unix()
	} else {
		p.Attempts++
		log.WithFields(log.Fields{"peer": p, "func": "UpdateStatus"}).Infof("%d unsuccessful attempts", p.Attempts)
	}
}
