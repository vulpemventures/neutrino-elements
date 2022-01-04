package peer

import (
	"io"

	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

// PeerID is peer IP address.
type PeerID string

// Peer describes a network's node.
type Peer interface {
	// ID returns peer ID. Must be unique in the network.
	ID() PeerID
	// Connection returns peer connection, using to send and receive Elements messages.
	Connection() io.ReadWriteCloser
	// Returns the Network Address of the peer
	Addr() *protocol.Addr
}
