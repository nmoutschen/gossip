package gossip

import (
	"fmt"
)

//State represents a piece of information at a given point in time
type State struct {
	Timestamp int64  `json:"time"`
	Data      string `json:"data"`
}

//String returns a string representation of the state
func (s *State) String() string {
	return fmt.Sprintf("%x", s.Timestamp)
}
