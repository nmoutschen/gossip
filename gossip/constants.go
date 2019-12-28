package gossip

import (
	"time"
)

//TODO: Switch to dynamic config
const (
	//ControllerScanDelay is the delay between two scans from a controller instance
	ControllerScanDelay time.Duration = 60 * time.Second
	//PeerBackoffDuration is the base duration before retrying to send a message to a peer
	PeerBackoffDuration time.Duration = 200 * time.Millisecond
	//PeerMaxAttempts is the number of attempts before considering the peer as unreachable
	PeerMaxAttempts int = 5
	//PeerMaxRecipients is the maximum number of peers that could receive a message
	PeerMaxRecipients int = 4
	//PeerMaxRetries is the number of retries before giving up on sending a message to a peer
	PeerMaxRetries int = 3
	//PeerMaxPingDelay is the time (in seconds) before the node will consider a peer as irrecoverable
	PeerMaxPingDelay int64 = 300
	//PeerMinPeers is the minimum number of peers that any peer should have
	PeerMinPeers int = 3
	//PingDelay is the delay between two ping attempts
	PingDelay time.Duration = 30 * time.Second
	//Protocol is the protocol to use to send messages to other peers, either http or https
	Protocol string = "http"
)
