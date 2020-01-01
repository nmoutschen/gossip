package main

import (
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/nmoutschen/gossip/gossip"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	node := gossip.NewNode(gossip.Addr{
		IP:   getIP(),
		Port: getPort(),
	})
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
	port, err := strconv.Atoi(sPort)
	if err != nil {
		return 8080
	}
	return port
}
