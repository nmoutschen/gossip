package gossip

import (
	"time"
)

//TODO: Switch to dynamic config
const (
	//ControllerMaxPingDelay is the time (in seconds) before the controller will consider a peer as irrecoverable
	ControllerMaxPingDelay int64 = 60 * 60 //1 hour
	//ControllerScanDelay is the delay between two scans from a controller instance
	ControllerScanDelay time.Duration = 60 * time.Second
	/*CorsAllowHeaders is used for the Access-Control-Allow-Headers header for
	HTTP responses.
	*/
	CorsAllowHeaders string = "Accept, Content-Type, Content-Length, Accept-Encoding"
	/*CorsAllowOrigin is used for the Access-Control-Allow-Origin header for
	HTTP responses.
	*/
	CorsAllowOrigin string = "*"
	//PeerBackoffDuration is the base duration before retrying to send a message to a peer
	PeerBackoffDuration time.Duration = 200 * time.Millisecond
	//PeerMaxAttempts is the number of attempts before considering the peer as unreachable
	PeerMaxAttempts int = 5
	//PeerMaxRecipients is the maximum number of peers that could receive a message
	PeerMaxRecipients int = 4
	//PeerMaxRetries is the number of retries before giving up on sending a message to a peer
	PeerMaxRetries int = 3
	//PeerMaxPingDelay is the time (in seconds) before the node will consider a peer as irrecoverable
	PeerMaxPingDelay int64 = 5 * 60 //5 minutes
	//PeerMinPeers is the minimum number of peers that any peer should have
	PeerMinPeers int = 3
	//PingDelay is the delay between two ping attempts
	PingDelay time.Duration = 30 * time.Second
	//Protocol is the protocol to use to send messages to other peers, either http or https
	Protocol string = "http"
)
