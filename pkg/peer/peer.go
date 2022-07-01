package peer

import (
	"io"
	"net"
	"sync"

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
	// Addr returns the Network Address of the peer
	Addr() *protocol.Addr
	// PeersTip returns current tip block height
	PeersTip() uint32
	// SetPeersTip sets the block height tip of the peer
	SetPeersTip(startBlockHeight uint32)
}

type elementsPeer struct {
	networkAddress   *protocol.Addr
	tcpConnection    net.Conn
	startBlockHeight uint32

	//used to synchronize access to the peers startBlockHeight
	m *sync.RWMutex
}

func NewElementsPeer(peerAddr string) (Peer, error) {
	conn, err := net.Dial("tcp", peerAddr)
	if err != nil {
		return nil, err
	}

	netAddress, err := protocol.ParseNodeAddr(peerAddr)
	if err != nil {
		return nil, err
	}

	return &elementsPeer{
		networkAddress: netAddress,
		tcpConnection:  conn,
		m:              new(sync.RWMutex),
	}, nil
}

func (e *elementsPeer) ID() PeerID {
	return PeerID(e.tcpConnection.LocalAddr().String())
}

func (e *elementsPeer) Connection() io.ReadWriteCloser {
	return e.tcpConnection
}

func (e *elementsPeer) Addr() *protocol.Addr {
	return e.networkAddress
}

func (e *elementsPeer) PeersTip() uint32 {
	e.m.RLock()
	defer e.m.RUnlock()

	return e.startBlockHeight
}

func (e *elementsPeer) SetPeersTip(startBlockHeight uint32) {
	e.m.Lock()
	defer e.m.Unlock()

	e.startBlockHeight = startBlockHeight
}

func (e *elementsPeer) String() string {
	return e.tcpConnection.RemoteAddr().String()
}
