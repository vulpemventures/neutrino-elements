package node

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

const (
	pingIntervalSec = 120
	pingTimeoutSec  = 60

	memPoolExpireTimeout = time.Minute * 5
	expireTxInterval     = time.Minute * 1
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

	memPool MemPool

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
		memPool:          NewMemPool(memPoolExpireTimeout, expireTxInterval, log.InfoLevel),
		quit:             make(chan struct{}),
	}, nil
}

func (no node) GetChainTip() (*block.Header, error) {
	return no.blockHeadersDb.ChainTip(context.Background())
}

// AddOutboundPeer sends a new version message to a new peer
// returns an error if the peer is already connected.
// it also starts a goroutine to monitor the peer's messages.
func (no node) AddOutboundPeer(outbound peer.Peer) error {
	if _, ok := no.Peers[outbound.ID()]; ok {
		return fmt.Errorf("peer already known by node")
	}

	msgVersion, err := no.createNodeVersionMsg(outbound)
	if err != nil {
		return err
	}

	err = no.sendMessage(outbound.Connection(), msgVersion)
	if err != nil {
		return err
	}

	go no.handlePeerMessages(outbound)

	return nil
}

// Run starts a node and add an initial outbound peer.
func (no node) Start(initialOutboundPeerAddr string) error {
	no.quit = make(chan struct{})
	initialPeer, err := peer.NewPeerTCP(initialOutboundPeerAddr)
	if err != nil {
		return err
	}

	err = no.AddOutboundPeer(initialPeer)
	if err != nil {
		return err
	}

	go no.monitorPeers()
	go no.monitorBlockHeaders()
	go no.monitorCFilters()

	no.memPool.Start()

	return nil
}

func (no *node) Stop() error {
	close(no.quit)
	no.memPool.Stop()
	return nil
}

// Return the best peer (now randomly)
// TODO : implement a better way to select the best peer (eg. by latency)
func (no node) getBestPeerForSync() peer.Peer {
	if len(no.Peers) == 0 {
		return nil
	}

	for _, p := range no.Peers {
		return p
	}

	return nil
}

// handlePeerMessages handles messages coming from peers.
func (no node) handlePeerMessages(p peer.Peer) {
	tmp := make([]byte, protocol.MsgHeaderLength)
	conn := p.Connection()

Loop:
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			logrus.Errorf(err.Error())
			no.DisconCh <- p.ID()
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
			if err := no.handleVersion(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'version': %+v", err)
				continue
			}
		case "verack":
			if err := no.handleVerack(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'verack': %+v", err)
				continue
			}
		case "ping":
			if err := no.handlePing(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'ping': %+v", err)
				continue
			}
		case "pong":
			if err := no.handlePong(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'pong': %+v", err)
				continue
			}
		case "inv":
			if err := no.handleInv(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'inv': %+v", err)
				continue
			}
		case "tx":
			if err := no.handleTx(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'tx': %+v", err)
				continue
			}
		case "block":
			if err := no.handleBlock(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'block': %+v", err)
				continue
			}
		case "sendcmpct":
			if err := no.handleSendCmpct(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'sendcmpct': %+v", err)
				continue
			}
		case "getheaders":
			if err := no.handleGetHeaders(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'getheaders': %+v", err)
				continue
			}
		case "headers":
			if err := no.handleHeaders(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'headers': %+v", err)
				continue
			}
		case "cfilter":
			if err := no.handleCFilter(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'cfilter': %+v", err)
				continue
			}
		case "getcfilters":
			if err := no.handleGetCFilters(&msgHeader, p); err != nil {
				logrus.Errorf("failed to handle 'getcfilters': %+v", err)
				continue
			}
		default:
			if err := no.skipMessage(&msgHeader, p); err != nil {
				logrus.Errorf("failed to skip message: %+v", err)
				continue
			}
		}

	}
}

// Returns the services (as serviceFlag) supported by the node.
func (no node) getServicesFlag() protocol.ServiceFlag {
	return protocol.SFNodeCF
}

// Returns the version message of the node.
func (no node) createNodeVersionMsg(p peer.Peer) (*protocol.Message, error) {
	peerAddr := p.Addr()

	return protocol.NewVersionMsg(
		no.Network,
		no.UserAgent,
		peerAddr.IP,
		peerAddr.Port,
		no.getServicesFlag(),
	)
}

// sendMessage first Marshal the `msg` arg and then use the `conn` to send it.
func (no *node) sendMessage(conn io.Writer, msg *protocol.Message) error {
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
func (no node) disconnectPeer(peerID peer.PeerID) {
	logrus.Debugf("disconnecting peer %s", peerID)

	peer := no.Peers[peerID]
	if peer == nil {
		return
	}

	peer.Connection().Close()
	delete(no.Peers, peerID)
}

// monitorBlockHeaders monitors new block headers comming from peers.
func (no *node) monitorBlockHeaders() {
	for {
		select {
		case <-no.quit:
			return
		case newHeader := <-no.blockHeadersCh:
			err := no.blockHeadersDb.WriteHeaders(context.Background(), newHeader)
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

				conn := no.getBestPeerForSync().Connection()
				err = no.sendMessage(conn, msg)
				if err != nil {
					logrus.Error(err)
					continue
				}
			}
		}
	}
}

// monitorCFilters monitors new cfilters comming from peers.
func (no *node) monitorCFilters() {
	for {
		select {
		case <-no.quit:
			return
		case newCFilterMsg := <-no.compactFiltersCh:
			entry, err := repository.NewFilterEntry(repository.FilterKey{
				FilterType: repository.FilterType(newCFilterMsg.FilterType),
				BlockHash:  newCFilterMsg.BlockHash.CloneBytes(),
			}, newCFilterMsg.Filter)
			if err != nil {
				logrus.Error(err)
				continue
			}

			err = no.filtersDb.PutFilter(context.Background(), entry)
			if err != nil {
				logrus.Error(err)
				continue
			}
		}
	}
}

func (no *node) SendTransaction(txhex string) error {
	msgTx, err := protocol.NewMsgTxFromHex(txhex)
	if err != nil {
		return err
	}

	msg, err := protocol.NewMessage("tx", no.Network, &msgTx)
	if err != nil {
		return err
	}

	for _, peer := range no.Peers {
		err = no.sendMessage(peer.Connection(), msg)
		if err != nil {
			return err
		}
	}

	return nil
}
