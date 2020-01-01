package gossip

import (
	"fmt"
)

/*Addr stores the address of a node, which is also a uniquely identifiable
representation of that node.
*/
type Addr struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

//String returns a string representation of the address
func (a Addr) String() string {
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}
