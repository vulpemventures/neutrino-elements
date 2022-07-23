package node

import (
	"context"
	"fmt"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/sirupsen/logrus"
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

	log.Debugf("node: tipHasAllAncestors: %v", tipHasAllAncestors)
	log.Debugf("node: chainTip: %v", chainTip.Height)
	log.Debugf("node: PeersTip: %v", p.PeersTip())

	if chainTip.Height < p.PeersTip() {
		return false, nil
	}

	return tipHasAllAncestors, nil
}

func (n *node) getGenesisBlockHash() (*chainhash.Hash, error) {
	genesisHexHash := protocol.GetCheckpoints(n.Network)[0]
	return chainhash.NewHashFromStr(genesisHexHash)
}

func (n *node) syncWithPeer(peerID peer.PeerID) error {
	log.Infof("node: syncing block headers with peer: %s ...", peerID)

	p := n.Peers[peerID]

	if p == nil {
		return fmt.Errorf("peer %s not found", peerID)
	}

	locator, err := n.blockHeadersDb.LatestBlockLocator(context.Background())
	if err != nil {
		if err == repository.ErrNoBlocksHeaders {
			genesisHash, err := n.getGenesisBlockHash()
			if err != nil {
				return err
			}

			locator = blockchain.BlockLocator{genesisHash}
		} else {
			return err
		}
	}
	stopHash := zeroHash

	msg, err := protocol.NewMsgGetHeaders(n.Network, stopHash, locator)
	if err != nil {
		return err
	}

	log.Debugf("sending getheaders to p %s", peerID)
	if err := n.sendMessage(p.Connection(), msg); err != nil {
		return err
	}

	return nil
}

func (n *node) checkSyncedInitial(p peer.Peer) {
	for {
		log.Debug("node: checkSynced")
		isSynced, _ := n.synced(p)
		if isSynced {
			n.notifySynced()
			return
		}
		time.Sleep(time.Second * 1)
	}
}

func (n *node) sync(p peer.Peer) {
	if p == nil {
		for _, bestPeer := range n.Peers {
			if bestPeer != nil {
				p = bestPeer
				break
			}
		}
	}

	if err := n.syncWithPeer(p.ID()); err != nil {
		log.Errorf("node: sync block headers with peer: %s failed: %s", p.ID(), err)
	}
}

func (n *node) notifySynced() {
	log.Debug("node: notifySynced")
	n.notifySyncedOnce.Do(
		func() {
			log.Debugf("node: syncing block headers finished")
			n.syncedChan <- struct{}{}
		},
	)
}
