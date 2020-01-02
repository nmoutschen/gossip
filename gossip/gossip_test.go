package gossip

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	DefaultConfig.Peer.MaxRetries = 0

	os.Exit(m.Run())
}
