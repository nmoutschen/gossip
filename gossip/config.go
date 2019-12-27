package gossip

import (
	"fmt"
)

//Config stores the configuration of a node, such as its IP address and port
type Config struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

//String returns a string representation of the configuration
func (c Config) String() string {
	return fmt.Sprintf("%s:%d", c.IP, c.Port)
}
