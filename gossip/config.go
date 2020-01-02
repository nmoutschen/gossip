package gossip

import (
	"time"
)

//ControllerConfig represents the configuration properties for controllers
type ControllerConfig struct {
	//MinPeers is the minimum number of peers that any peer should have
	MinPeers int `json:"minPeers"`
	/*MaxPingDelay is the time (in ms) before the controller will consider a
	peer as irrecoverable*/
	MaxPingDelay int64 `json:"maxPingDelay"`
	/*ScanInterval is the delay (in ms) between two scans from a controller
	instance*/
	ScanInterval time.Duration `json:"scanInterval"`
}

//CorsConfig represents the configuration properties for CORS
type CorsConfig struct {
	/*AllowHeaders is used for the Access-Control-Allow-Headers header for HTTP
	responses.*/
	AllowHeaders string `json:"allowHeaders"`
	/*AllowOrigin is used for the Access-Control-Allow-Origin header for HTTP
	responses.*/
	AllowOrigin string `json:"allowOrigin"`
}

//NodeConfig represents the configuration properties for nodes
type NodeConfig struct {
	/*MaxRecipients is the maximum number of peers to a node that could receive
	a message*/
	MaxRecipients int `json:"maxRecipients"`
	/*MaxPingDelay is the time (in ms) before the node will consider a
	peer as irrecoverable*/
	MaxPingDelay int64 `json:"maxPingDelay"`
	//ScanInterval is the delay (in ms) between two pings from a node instance
	PingInterval time.Duration `json:"pingInterval"`
}

//PeerConfig represents the configuration properties for peers
type PeerConfig struct {
	/*BackoffDuration is the base duration (in ms) before retrying to send a
	message to a peer*/
	BackoffDuration time.Duration `json:"backoffDuration"`
	/*MaxAttempts is the number of attempts before considering the peer as
	unreachable*/
	MaxAttempts int `json:"maxAttempts"`
	/*MaxRetries is the number of retries before giving up on sending a message
	to a peer*/
	MaxRetries int `json:"maxRetries"`
}

//Config represents all configuration properties
type Config struct {
	Controller ControllerConfig `json:"controller"`
	Cors       CorsConfig       `json:"cors"`
	Node       NodeConfig       `json:"node"`
	Peer       PeerConfig       `json:"peer"`
	/*Protocol is the protocol to use to send messages to other peers, either
	http or https*/
	Protocol string `json:"protocol"`
}

//DefaultConfig is the default configuration for controllers, nodes and peers.
var DefaultConfig *Config = &Config{
	Controller: ControllerConfig{
		MaxPingDelay: 3600000, //1 hour (3 600 000 ms)
		MinPeers:     3,
		ScanInterval: 60000, //1 minute (60 000 ms)
	},
	Cors: CorsConfig{
		AllowHeaders: "Accept, Content-Type, Content-Length, Accept-Encoding",
		AllowOrigin:  "*",
	},
	Node: NodeConfig{
		MaxRecipients: 4,
		MaxPingDelay:  300000, //5 minutes (300 000 ms)
		PingInterval:  30000,  //30 seconds (30 000 ms)
	},
	Peer: PeerConfig{
		BackoffDuration: 200, //200 ms
		MaxAttempts:     5,
		MaxRetries:      3,
	},
	Protocol: "http",
}
