package gossip

import (
	"time"
)

//ControllerConfig represents the configuration properties for controllers
type ControllerConfig struct {
	/*MaxPingDelay is the time (in ms) before the controller will consider a
	peer as irrecoverable*/
	MaxScanDelay time.Duration `json:"maxScanDelay" yaml:"maxScanDelay" default:"1h"`
	//MinPeers is the minimum number of peers that any peer should have
	MinPeers int `json:"minPeers" yaml:"minPeers" default:"3"`
	/*ScanInterval is the delay (in ms) between two scans from a controller
	instance*/
	ScanInterval time.Duration `json:"scanInterval" yaml:"scanInterval" default:"60s"`
	//IP address of the controller
	IP string `json:"ip" yaml:"ip" default:"127.0.0.1"`
	//Port for the HTTP server on the controller
	Port int `json:"port" yaml:"port" default:"7080"`
}

//CorsConfig represents the configuration properties for CORS
type CorsConfig struct {
	/*AllowHeaders is used for the Access-Control-Allow-Headers header for HTTP
	responses.*/
	AllowHeaders string `json:"allowHeaders" yaml:"allowHeaders" default:"Accept, Content-Type, Content-Length, Accept-Encoding"`
	/*AllowOrigin is used for the Access-Control-Allow-Origin header for HTTP
	responses.*/
	AllowOrigin string `json:"allowOrigin" yaml:"allowOrigin" default:"*"`
}

//NodeConfig represents the configuration properties for nodes
type NodeConfig struct {
	/*MaxRecipients is the maximum number of peers to a node that could receive
	a message*/
	MaxRecipients int `json:"maxRecipients" yaml:"maxRecipients" default:"4"`
	/*MaxPingDelay is the time before the node will consider a
	peer as irrecoverable*/
	MaxPingDelay time.Duration `json:"maxPingDelay" yaml:"maxPingDelay" default:"5m"`
	//ScanInterval is the delay between two pings from a node instance
	PingInterval time.Duration `json:"pingInterval" yaml:"pingInterval" default:"30s"`
	//IP address of the node
	IP string `json:"ip" yaml:"ip" default:"127.0.0.1"`
	//Port for the HTTP server on the node
	Port int `json:"port" yaml:"port" default:"8080"`
}

//PeerConfig represents the configuration properties for peers
type PeerConfig struct {
	/*BackoffDuration is the base duration before retrying to send a
	message to a peer*/
	BackoffDuration time.Duration `json:"backoffDuration" yaml:"backoffDuration" default:"200ms"`
	/*MaxAttempts is the number of attempts before considering the peer as
	unreachable*/
	MaxAttempts int `json:"maxAttempts" yaml:"maxAttempts" default:"5"`
	/*MaxRetries is the number of retries before giving up on sending a message
	to a peer*/
	MaxRetries int `json:"maxRetries" yaml:"maxRetries" default:"3"`
}

//Config represents all configuration properties
type Config struct {
	Controller ControllerConfig `json:"controller" yaml:"controller"`
	Cors       CorsConfig       `json:"cors" yaml:"cors"`
	Node       NodeConfig       `json:"node" yaml:"node"`
	Peer       PeerConfig       `json:"peer" yaml:"peer"`
	/*Protocol is the protocol to use to send messages to other peers, either
	http or https*/
	Protocol string `json:"protocol" yaml:"protocol" default:"http"`
}

//DefaultConfig is the default configuration for controllers, nodes and peers.
var DefaultConfig *Config = &Config{
	Controller: ControllerConfig{
		MaxScanDelay: 3600000, //1 hour (3 600 000 ms)
		MinPeers:     3,
		ScanInterval: 60 * time.Second, //1 minute (60 000 ms)
		IP:           "127.0.0.1",
		Port:         7080,
	},
	Cors: CorsConfig{
		AllowHeaders: "Accept, Content-Type, Content-Length, Accept-Encoding",
		AllowOrigin:  "*",
	},
	Node: NodeConfig{
		MaxRecipients: 4,
		MaxPingDelay:  300000,           //5 minutes (300 000 ms)
		PingInterval:  30 * time.Second, //30 seconds (30 000 ms)
		IP:            "127.0.0.1",
		Port:          8080,
	},
	Peer: PeerConfig{
		BackoffDuration: 200 * time.Millisecond, //200 ms
		MaxAttempts:     5,
		MaxRetries:      3,
	},
	Protocol: "http",
}
