package node

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

type peerPing struct {
	nonce  uint64
	peerID peer.PeerID
}

func (n *Node) addPeer(peer peer.Peer) error {
	if _, found := n.Peers[peer.ID()]; found {
		return fmt.Errorf("peer already known: %s", peer.ID())
	}

	id := peer.ID()
	n.Peers[id] = peer
	n.peersPongCh[id] = make(chan uint64)

	if len(n.Peers) == 1 {
		n.checkSync(peer)
	}

	return nil
}

func (n *Node) monitorPeers() {
	peerPings := make(map[uint64]peer.PeerID)

	for {
		select {
		case nonce := <-n.pongsCh:
			peerID := peerPings[nonce]
			if peerID == "" {
				break
			}
			peer := n.Peers[peerID]
			if peer == nil {
				break
			}

			n.peersPongCh[peerID] <- nonce
			delete(peerPings, nonce)

		case pp := <-n.pingsCh:
			peerPings[pp.nonce] = pp.peerID

		case peerID := <-n.DisconCh:
			n.disconnectPeer(peerID)

			for k, v := range peerPings {
				if v == peerID {
					delete(peerPings, k)
					break
				}
			}
		}
	}
}

// monitors the pings/pongs for a peer
func (n *Node) monitorPeer(peer peer.Peer) {
	for {
		time.Sleep(pingIntervalSec * time.Second)

		ping, nonce, err := protocol.NewPingMsg(n.Network)
		if err != nil {
			logrus.Fatalf("monitorPeer, NewPingMsg: %v", err)
		}

		msg, err := binary.Marshal(ping)
		if err != nil {
			logrus.Fatalf("monitorPeer, binary.Marshal: %v", err)
		}

		if _, err := peer.Connection().Write(msg); err != nil {
			n.disconnectPeer(peer.ID())
		}

		logrus.Debugf("sent 'ping' to %s", peer)

		n.pingsCh <- peerPing{
			nonce:  nonce,
			peerID: peer.ID(),
		}

		t := time.NewTimer(pingTimeoutSec * time.Second)

		select {
		case pn := <-n.peersPongCh[peer.ID()]:
			if pn != nonce {
				logrus.Errorf("nonce doesn't match for %s: want %d, got %d", peer, nonce, pn)
				n.DisconCh <- peer.ID()
				return
			}
			logrus.Debugf("got 'pong' from %s", peer)
		case <-t.C:
			n.DisconCh <- peer.ID()
			return
		}

		t.Stop()
	}
}
