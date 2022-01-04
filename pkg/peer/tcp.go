package peer

import (
	"io"
	"net"

	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

// peerTCP implements Peer interface.
// it lets to connect to an elements node via TCP	conn
type peerTCP struct {
	networkAddress *protocol.Addr
	tcpConnection  net.Conn
}

var _ Peer = (*peerTCP)(nil)

func NewPeerTCP(peerAddr string) (Peer, error) {
	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		return nil, err
	}

	netAddress, err := protocol.ParseNodeAddr(peerAddr)
	if err != nil {
		return nil, err
	}

	return &peerTCP{
		networkAddress: netAddress,
		tcpConnection:  conn,
	}, nil
}

// ID returns peer ID. Must be unique in the network.
func (p peerTCP) ID() PeerID {
	return PeerID(p.tcpConnection.LocalAddr().String())
}

// ReadWriteCloser interface using to communicate with the peer.
func (p *peerTCP) Connection() io.ReadWriteCloser {
	return p.tcpConnection
}

func (p *peerTCP) Addr() *protocol.Addr {
	return p.networkAddress
}

func (p *peerTCP) String() string {
	return p.tcpConnection.RemoteAddr().String()
}
