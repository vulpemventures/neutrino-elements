package node

import (
	"fmt"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

var zeroHash [32]byte = [32]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

func (no *node) isSync() (bool, error) {
	chainTip, err := no.blockHeadersDb.ChainTip()
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

	tipHasAllAncestors, err := no.blockHeadersDb.HasAllAncestors(tipHash)
	if err != nil {
		return false, err
	}

	return tipHasAllAncestors, nil
}

func (no *node) getGenesisBlockHash() (*chainhash.Hash, error) {
	genesisHexHash := protocol.GetCheckpoints(no.Network)[0]
	return chainhash.NewHashFromStr(genesisHexHash)
}

func (no *node) syncWithPeer(peerID peer.PeerID) error {
	peer := no.Peers[peerID]

	if peer == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}

	locator, err := no.blockHeadersDb.LatestBlockLocator()
	if err != nil {
		genesisHash, err := no.getGenesisBlockHash()
		if err != nil {
			return err
		}

		locator = blockchain.BlockLocator{genesisHash}
	}

	msg, err := protocol.NewMsgGetHeaders(no.Network, zeroHash, locator)
	if err != nil {
		return err
	}

	logrus.Debugf("sending getheaders to peer %s", peerID)
	if err := no.sendMessage(peer.Connection(), msg); err != nil {
		return err
	}

	return nil
}

func (n *node) checkSync(p peer.Peer) {
	if p == nil {
		for _, bestPeer := range n.Peers {
			if bestPeer != nil {
				p = bestPeer
				break
			}
		}
	}

	isSync, _ := n.isSync()
	if !isSync {
		logrus.Infof("start sync block headers with peer: %s", p.ID())
		n.syncWithPeer(p.ID())
	}
}
