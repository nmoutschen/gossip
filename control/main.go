package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/nmoutschen/gossip/gossip"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	config := getConfig()
	control := gossip.NewController(config)
	control.Run()
}

func getConfig() *gossip.Config {
	config := new(gossip.Config)

	err := envconfig.Process("gossip", config)
	if err != nil {
		log.Fatal(err.Error())
	}

	return config
}
