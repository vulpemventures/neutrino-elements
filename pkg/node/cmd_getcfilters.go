package node

import (
	"fmt"
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
	"golang.org/x/net/context"
)

func (n node) handleGetCFilters(header *protocol.MessageHeader, p peer.Peer) error {
	var getCFilters protocol.MsgGetCFilters
	lr := io.LimitReader(p.Connection(), int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&getCFilters); err != nil {
		return err
	}

	if getCFilters.FilterType != byte(repository.RegularFilter) {
		return fmt.Errorf("invalid filter type")
	}

	stopHash, err := chainhash.NewHash(getCFilters.StopHash[:])
	if err != nil {
		return err
	}

	endBlockHeader, err := n.blockHeadersDb.GetBlockHeader(*stopHash)
	if err != nil {
		return err
	}

	if endBlockHeader.Height < getCFilters.StartHeight {
		return fmt.Errorf("end height is less than start height")
	}

	if endBlockHeader.Height-getCFilters.StartHeight >= 1000 {
		return fmt.Errorf("end height is too far away from start height")
	}

	for height := endBlockHeader.Height; height > getCFilters.StartHeight; height-- {
		blockHash, err := n.blockHeadersDb.GetBlockHashByHeight(height)
		if err != nil {
			logrus.Error(err)
			continue
		}

		filter, err := n.filtersDb.GetFilter(
			context.Background(), repository.FilterKey{
				BlockHash:  blockHash.CloneBytes(),
				FilterType: repository.RegularFilter,
			})
		if err != nil {
			logrus.Error(err)
			continue
		}

		msgCFilter, err := protocol.NewMsgCFilter(
			n.Network,
			blockHash,
			filter,
		)
		if err != nil {
			logrus.Error(err)
			continue
		}

		if err := n.sendMessage(p.Connection(), msgCFilter); err != nil {
			logrus.Error(err)
			continue
		}
	}

	return nil
}
