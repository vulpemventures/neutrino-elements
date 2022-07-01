package node

import (
	"context"
	"github.com/vulpemventures/go-elements/block"
	"io"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

func (n node) handleBlock(header *protocol.MessageHeader, p peer.Peer) error {
	var msgBlock protocol.MsgBlock

	_, err := n.blockHeadersDb.ChainTip(context.Background())
	if err != nil && err != repository.ErrNoBlocksHeaders {
		return err
	}

	lr := io.LimitReader(p.Connection(), int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&msgBlock); err != nil {
		return err
	}

	tip, err := n.blockHeadersDb.ChainTip(context.Background())
	if err != nil {
		if err == repository.ErrNoBlocksHeaders {
			tip = &block.Header{
				Height: 0,
			}
		} else {
			return err
		}
	}

	//receivedTip := msgBlock.Header.Height
	//if receivedTip == tip.Height {
	//	n.notifySynced()
	//}

	//if new block arrives before sync is done and if it is greater than peer
	//start height update height so that we can sync till this height
	//eg. when node starts syncing and peer start height(tip) is 50000,
	//knowing that node can receive max 2000 in one message, it will happen that
	//new blocks are going to be mined before sync is done and in order to sync till
	//the latest block, and not 50000, we need to update the peer tip height to
	//the latest block
	newBlockHeight := msgBlock.Header.Height
	if newBlockHeight != tip.Height+1 {
		if newBlockHeight > p.PeersTip() {
			p.SetPeersTip(newBlockHeight)
			n.sync(p)
		}

		return nil
	}
	n.blockHeadersCh <- *msgBlock.Header
	n.memPool.CheckTxConfirmed(msgBlock.Block)

	return nil
}
