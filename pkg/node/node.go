package node

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
	"github.com/vulpemventures/neutrino-elements/pkg/repository/inmemory"
)

// PeerID is peer IP address.
type PeerID string

// Node implements a Bitcoin node.
type Node struct {
	Network   protocol.Magic
	Peers     map[PeerID]*Peer
	PingCh    chan peerPing
	PongCh    chan uint64
	DisconCh  chan PeerID
	UserAgent string

	compactFiltersCh chan protocol.MsgCFilter
	blockHeadersCh   chan block.Header
	filtersDb        repository.FilterRepository
	blockHeadersDb   repository.BlockHeaderRepository
}

// New returns a new Node.
func New(network, userAgent string) (*Node, error) {
	networkMagic, ok := protocol.Networks[network]
	if !ok {
		return nil, fmt.Errorf("unsupported network %s", network)
	}

	return &Node{
		Network:   networkMagic,
		Peers:     make(map[PeerID]*Peer),
		PingCh:    make(chan peerPing),
		DisconCh:  make(chan PeerID),
		PongCh:    make(chan uint64),
		UserAgent: userAgent,

		compactFiltersCh: make(chan protocol.MsgCFilter),
		blockHeadersCh:   make(chan block.Header),
		filtersDb:        inmemory.NewFilterInmemory(),
		blockHeadersDb:   inmemory.NewHeaderInmemory(),
	}, nil
}

func (no Node) GetBestPeer() *Peer {
	if len(no.Peers) == 0 {
		return nil
	}

	for _, p := range no.Peers {
		return p
	}

	return nil
}

// Run starts a node.
func (no Node) Run(nodeAddr string) error {
	peerAddr, err := ParseNodeAddr(nodeAddr)
	if err != nil {
		return err
	}

	version, err := no.createNodeVersionMsg(peerAddr)
	if err != nil {
		return err
	}

	conn, err := net.Dial("tcp", nodeAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	err = no.sendMessage(conn, version)
	if err != nil {
		return err
	}

	go no.monitorPeers()
	go no.monitorBlockHeaders()
	go no.monitorCFilters()

	tmp := make([]byte, protocol.MsgHeaderLength)

Loop:
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				return err
			}
			logrus.Errorf(err.Error())
			break Loop
		}

		var msgHeader protocol.MessageHeader
		if err := binary.NewDecoder(bytes.NewReader(tmp[:n])).Decode(&msgHeader); err != nil {
			logrus.Errorf("invalid header: %+v", err)
			continue
		}

		if err := msgHeader.Validate(); err != nil {
			logrus.Error(err)
			continue
		}

		logrus.Debugf("received message: %s", msgHeader.Command)

		switch msgHeader.CommandString() {
		case "version":
			if err := no.handleVersion(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'version': %+v", err)
				continue
			}
		case "verack":
			if err := no.handleVerack(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'verack': %+v", err)
				continue
			}
		case "ping":
			if err := no.handlePing(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'ping': %+v", err)
				continue
			}
		case "pong":
			if err := no.handlePong(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'pong': %+v", err)
				continue
			}
		case "inv":
			if err := no.handleInv(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'inv': %+v", err)
				continue
			}
		case "tx":
			if err := no.handleTx(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'tx': %+v", err)
				continue
			}
		case "block":
			if err := no.handleBlock(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'block': %+v", err)
				continue
			}
		case "sendcmpct":
			if err := no.handleSendCmpct(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'sendcmpct': %+v", err)
				continue
			}
		case "getheaders":
			if err := no.handleGetHeaders(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'getheaders': %+v", err)
				continue
			}
		case "headers":
			if err := no.handleHeaders(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'headers': %+v", err)
				continue
			}
		case "cfilter":
			if err := no.handleCFilter(&msgHeader, conn); err != nil {
				logrus.Errorf("failed to handle 'cfilter': %+v", err)
				continue
			}
		}
	}

	return nil
}

// Returns the services (as serviceFlag) supported by the node.
func (no Node) getServicesFlag() protocol.ServiceFlag {
	return protocol.SFNodeCF
}

// Returns the version message of the node.
func (no Node) createNodeVersionMsg(peerAddr *Addr) (*protocol.Message, error) {
	return protocol.NewVersionMsg(
		no.Network,
		no.UserAgent,
		peerAddr.IP,
		peerAddr.Port,
		no.getServicesFlag(),
	)
}

// sendMessage first Marshal the `msg` arg and then use the `conn` to send it.
func (no *Node) sendMessage(conn io.Writer, msg *protocol.Message) error {
	logrus.Debugf("node sends message: %s", msg.CommandString())
	msgSerialized, err := binary.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = conn.Write(msgSerialized)
	return err
}

// on disconnect, remove the peer from the node.
// and close the connection.
func (no Node) disconnectPeer(peerID PeerID) {
	logrus.Debugf("disconnecting peer %s", peerID)

	peer := no.Peers[peerID]
	if peer == nil {
		return
	}

	peer.Connection.Close()
	delete(no.Peers, peerID)
}

// monitorBlockHeaders monitors new block headers comming from peers.
func (no *Node) monitorBlockHeaders() {
	for newHeader := range no.blockHeadersCh {
		err := no.blockHeadersDb.WriteHeaders(newHeader)
		if err != nil {
			logrus.Error(err)
			continue
		}

		if len(no.Peers) > 0 {
			hash, err := newHeader.Hash()
			if err != nil {
				logrus.Error(err)
				continue
			}

			getcFilter := protocol.MsgGetCFilters{
				FilterType:  0,
				StartHeight: newHeader.Height,
				StopHash:    hash,
			}

			msg, err := protocol.NewMessage("getcfilters", no.Network, &getcFilter)
			if err != nil {
				logrus.Error(err)
				continue
			}

			conn := no.GetBestPeer().Connection
			err = no.sendMessage(conn, msg)
			if err != nil {
				logrus.Error(err)
				continue
			}
		}
	}
}

// monitorCFilters monitors new cfilters comming from peers.
func (no *Node) monitorCFilters() {
	for newCFilterMsg := range no.compactFiltersCh {
		err := no.filtersDb.PutFilter(newCFilterMsg.BlockHash, newCFilterMsg.Filter, repository.FilterType(newCFilterMsg.FilterType))
		if err != nil {
			logrus.Error(err)
			continue
		}
	}
}
