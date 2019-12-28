package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/nmoutschen/gossip/gossip"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	node := gossip.NewNode(getIP(), getPort())
	node.Run()
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
		return 8080
	}
	var port int
	_, err := fmt.Sscanf(sPort, "%d", &port)
	if err != nil {
		return 8080
	}
	return port
}
