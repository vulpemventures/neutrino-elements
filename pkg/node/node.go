package node

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

const (
	pingIntervalSec = 120
	pingTimeoutSec  = 60
)

type NodeService interface {
	Start(initialOutboundPeerAddr string) error
	Stop() error
	AddOutboundPeer(peer.Peer) error
	SendTransaction(txhex string) error
	GetChainTip() (*block.Header, error)
}

// node implements an Elements full node.
// It aims to sync block headers and compact filters.
type node struct {
	Network     protocol.Magic
	Peers       map[peer.PeerID]peer.Peer
	pingsCh     chan peerPing
	pongsCh     chan uint64
	peersPongCh map[peer.PeerID]chan uint64

	DisconCh  chan peer.PeerID
	UserAgent string

	compactFiltersCh chan protocol.MsgCFilter
	blockHeadersCh   chan block.Header
	filtersDb        repository.FilterRepository
	blockHeadersDb   repository.BlockHeaderRepository

	quit chan struct{}
}

var _ NodeService = (*node)(nil)

type NodeConfig struct {
	Network        string
	UserAgent      string
	FiltersDB      repository.FilterRepository
	BlockHeadersDB repository.BlockHeaderRepository
}

// New returns a new Node.
func New(config NodeConfig) (NodeService, error) {
	networkMagic, ok := protocol.Networks[config.Network]
	if !ok {
		return nil, fmt.Errorf("unsupported network %s", config.Network)
	}

	return &node{
		Network:     networkMagic,
		Peers:       make(map[peer.PeerID]peer.Peer),
		pingsCh:     make(chan peerPing),
		pongsCh:     make(chan uint64),
		peersPongCh: make(map[peer.PeerID]chan uint64),
		DisconCh:    make(chan peer.PeerID),
		UserAgent:   config.UserAgent,

		compactFiltersCh: make(chan protocol.MsgCFilter),
		blockHeadersCh:   make(chan block.Header),
		filtersDb:        config.FiltersDB,
		blockHeadersDb:   config.BlockHeadersDB,
		quit:             make(chan struct{}),
	}, nil
}

func (n node) GetChainTip() (*block.Header, error) {
	return n.blockHeadersDb.ChainTip(context.Background())
}

// AddOutboundPeer sends a new version message to a new peer
// returns an error if the peer is already connected.
// it also starts a goroutine to monitor the peer's messages.
func (n node) AddOutboundPeer(outbound peer.Peer) error {
	if _, ok := n.Peers[outbound.ID()]; ok {
		return fmt.Errorf("peer already known by node")
	}

	msgVersion, err := n.createNodeVersionMsg(outbound)
	if err != nil {
		return err
	}

	err = n.sendMessage(outbound.Connection(), msgVersion)
	if err != nil {
		return err
	}

	go n.handlePeerMessages(outbound)

	return nil
}

// Run starts a node and add an initial outbound peer.
func (n node) Start(initialOutboundPeerAddr string) error {
	n.quit = make(chan struct{})
	initialPeer, err := peer.NewElementsPeer(initialOutboundPeerAddr)
	if err != nil {
		return err
	}

	err = n.AddOutboundPeer(initialPeer)
	if err != nil {
		return err
	}

	go n.monitorPeers()
	go n.monitorBlockHeaders()
	go n.monitorCFilters()

	return nil
}

func (n *node) Stop() error {
	close(n.quit)
	return nil
}

// Return the best peer (now randomly)
// TODO : implement a better way to select the best peer (eg. by latency)
func (n *node) getBestPeerForSync() peer.Peer {
	if len(n.Peers) == 0 {
		return nil
	}

	for _, p := range n.Peers {
		return p
	}

	return nil
}

// handlePeerMessages handles messages coming from peers.
func (n *node) handlePeerMessages(p peer.Peer) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered handlePeerMessages", r)
			n.handlePeerMessages(p)
		}
	}()

	tmp := make([]byte, protocol.MsgHeaderLength)
	conn := p.Connection()

Loop:
	for {
		nn, err := conn.Read(tmp)
		if err != nil {
			logrus.Errorf(err.Error())
			n.DisconCh <- p.ID()
			break Loop
		}

		var msgHeader protocol.MessageHeader
		if err := binary.NewDecoder(bytes.NewReader(tmp[:nn])).Decode(&msgHeader); err != nil {
			logrus.Debugf("decoding header failed: %+v", err)
			continue
		}

		if err := msgHeader.Validate(); err != nil {
			logrus.Debugln(err)
			logrus.Debugf("validate header failed: %+v", err)
			continue
		}

		logrus.Debugf("received message: %s", msgHeader.Command)

		switch msgHeader.CommandString() {
		case "version":
			if err := n.handleVersion(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'version': %+v", err)
				continue
			}
		case "verack":
			if err := n.handleVerack(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'verack': %+v", err)
				continue
			}
		case "ping":
			if err := n.handlePing(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'ping': %+v", err)
				continue
			}
		case "pong":
			if err := n.handlePong(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'pong': %+v", err)
				continue
			}
		case "inv":
			if err := n.handleInv(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'inv': %+v", err)
				continue
			}
		case "tx":
			if err := n.handleTx(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'tx': %+v", err)
				continue
			}
		case "block":
			if err := n.handleBlock(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'block': %+v", err)
				continue
			}
		case "sendcmpct":
			if err := n.handleSendCmpct(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'sendcmpct': %+v", err)
				continue
			}
		case "getheaders":
			if err := n.handleGetHeaders(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'getheaders': %+v", err)
				continue
			}
		case "headers":
			if err := n.handleHeaders(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'headers': %+v", err)
				continue
			}
		case "cfilter":
			if err := n.handleCFilter(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'cfilter': %+v", err)
				continue
			}
		case "getcfilters":
			if err := n.handleGetCFilters(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'getcfilters': %+v", err)
				continue
			}
		default:
			if err := n.skipMessage(&msgHeader, p); err != nil {
				logrus.Errorf("failed to skip message: %+v", err)
				continue
			}
		}

	}
}

// Returns the services (as serviceFlag) supported by the node.
func (n *node) getServicesFlag() protocol.ServiceFlag {
	return protocol.SFNodeCF
}

// Returns the version message of the node.
func (n *node) createNodeVersionMsg(p peer.Peer) (*protocol.Message, error) {
	peerAddr := p.Addr()

	return protocol.NewVersionMsg(
		n.Network,
		n.UserAgent,
		peerAddr.IP,
		peerAddr.Port,
		n.getServicesFlag(),
	)
}

// sendMessage first Marshal the `msg` arg and then use the `conn` to send it.
func (n *node) sendMessage(conn io.Writer, msg *protocol.Message) error {
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
func (n *node) disconnectPeer(peerID peer.PeerID) {
	logrus.Debugf("disconnecting peer %s", peerID)

	peer := n.Peers[peerID]
	if peer == nil {
		return
	}

	peer.Connection().Close()
	delete(n.Peers, peerID)
}

// monitorBlockHeaders monitors new block headers comming from peers.
func (n *node) monitorBlockHeaders() {
	for {
		select {
		case <-n.quit:
			return
		case newHeader := <-n.blockHeadersCh:
			err := n.blockHeadersDb.WriteHeaders(context.Background(), newHeader)
			if err != nil {
				logrus.Error(err)
				continue
			}

			fmt.Printf("new block: %v\n", newHeader.Height)

			if len(n.Peers) > 0 {
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

				msg, err := protocol.NewMessage("getcfilters", n.Network, &getcFilter)
				if err != nil {
					logrus.Error(err)
					continue
				}

				conn := n.getBestPeerForSync().Connection()
				err = n.sendMessage(conn, msg)
				if err != nil {
					logrus.Error(err)
					continue
				}
			}
		}
	}
}

// monitorCFilters monitors new cfilters comming from peers.
func (n *node) monitorCFilters() {
	for {
		select {
		case <-n.quit:
			return
		case newCFilterMsg := <-n.compactFiltersCh:
			entry, err := repository.NewFilterEntry(repository.FilterKey{
				FilterType: repository.FilterType(newCFilterMsg.FilterType),
				BlockHash:  newCFilterMsg.BlockHash.CloneBytes(),
			}, newCFilterMsg.Filter)
			if err != nil {
				logrus.Error(err)
				continue
			}

			err = n.filtersDb.PutFilter(context.Background(), entry)
			if err != nil {
				logrus.Error(err)
				continue
			}
		}
	}
}

func (n *node) SendTransaction(txhex string) error {
	msgTx, err := protocol.NewMsgTxFromHex(txhex)
	if err != nil {
		return err
	}

	msg, err := protocol.NewMessage("tx", n.Network, &msgTx)
	if err != nil {
		return err
	}

	for _, peer := range n.Peers {
		err = n.sendMessage(peer.Connection(), msg)
		if err != nil {
			return err
		}
	}

	return nil
}
