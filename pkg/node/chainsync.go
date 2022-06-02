package node

import (
	"context"
	"fmt"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
	"time"
)

var zeroHash [32]byte = [32]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

func (n *node) synced(p peer.Peer) (bool, error) {
	chainTip, err := n.blockHeadersDb.ChainTip(context.Background())
	if err != nil {
		return false, err
	}

	if chainTip == nil {
		return false, nil
	}

	tipHash, err := chainTip.Hash()
	if err != nil {
		return false, err
	}

	tipHasAllAncestors, err := n.blockHeadersDb.HasAllAncestors(context.Background(), tipHash)
	if err != nil {
		return false, err
	}

	if chainTip.Height != p.StartBlockHeight() {
		return false, nil
	}

	return tipHasAllAncestors, nil
}

func (n *node) getGenesisBlockHash() (*chainhash.Hash, error) {
	genesisHexHash := protocol.GetCheckpoints(n.Network)[0]
	return chainhash.NewHashFromStr(genesisHexHash)
}

func (n *node) syncWithPeer(peerID peer.PeerID) error {
	peer := n.Peers[peerID]

	if peer == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}

	var stopHash [32]byte

	locator, err := n.blockHeadersDb.LatestBlockLocator(context.Background())
	if err != nil {
		if err == repository.ErrNoBlocksHeaders {
			genesisHash, err := n.getGenesisBlockHash()
			if err != nil {
				return err
			}

			locator = blockchain.BlockLocator{genesisHash}
			stopHash = zeroHash
		} else {
			return err
		}
	} else {
		stopHash = *locator[len(locator)-1]
	}

	msg, err := protocol.NewMsgGetHeaders(n.Network, stopHash, locator)
	if err != nil {
		return err
	}

	logrus.Debugf("sending getheaders to peer %s", peerID)
	if err := n.sendMessage(peer.Connection(), msg); err != nil {
		return err
	}

	return nil
}

func (n *node) sync(p peer.Peer) {
	logrus.Infof("node: start sync block headers with peer: %s", p.ID())

	if p == nil {
		for _, bestPeer := range n.Peers {
			if bestPeer != nil {
				p = bestPeer
				break
			}
		}
	}

	//TODO sync needs to be done when receiving headers from peer
	isSynced, _ := n.synced(p)
	for {
		if isSynced {
			logrus.Infof("node: sync block headers with peer: %s is done", p.ID())
			return
		}

		if err := n.syncWithPeer(p.ID()); err != nil {
			logrus.Errorf("node: sync block headers with peer: %s failed: %s", p.ID(), err)
		}

		isSynced, _ = n.synced(p)
		time.Sleep(time.Second * 2)
	}
}
