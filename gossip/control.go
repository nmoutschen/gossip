package gossip

import (
	log "github.com/sirupsen/logrus"
)

//Controller represents a controller instance
type Controller struct {
	Peers []*Peer
}

//NewControl creates a new control instance
func NewControl() *Controller {
	c := &Controller{}

	log.WithFields(log.Fields{"controller": c, "func": "NewControl"}).Info("Initializing controller")

	return c
}

//Run starts the control instance
func (c *Controller) Run() {
	//TODO
	log.WithFields(log.Fields{"controller": c, "func": "NewControl"}).Info("Starting controller")
}
