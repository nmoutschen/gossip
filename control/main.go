package main

import (
	"fmt"
	"os"

	"github.com/nmoutschen/gossip/gossip"
)

func main() {
	control := gossip.NewController(getIP(), getPort())

	// Manually add a node for testing
	p := &gossip.Peer{
		Config: gossip.Config{
			IP:   "127.0.0.1",
			Port: 8080,
		},
	}
	control.Peers.Store(p.Config, p)

	control.Run()
}

func getIP() string {
	ip, ok := os.LookupEnv("GOSSIP_IP")
	if !ok {
		return "127.0.0.1"
	}
	return ip
}

func getPort() int {
	sPort, ok := os.LookupEnv("GOSSIP_PORT")
	if !ok {
		return 7080
	}
	var port *int
	_, err := fmt.Sscanf(sPort, "%d", port)
	if err != nil {
		return 7080
	}
	return *port
}
