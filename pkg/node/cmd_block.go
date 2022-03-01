package node

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

func (no node) handleBlock(header *protocol.MessageHeader, p peer.Peer) error {
	var block protocol.MsgBlock

	currentChainTip, err := no.blockHeadersDb.ChainTip(context.Background())
	if err != nil && err != repository.ErrNoBlocksHeaders {
		return err
	}

	if currentChainTip != nil {
		logrus.Println(currentChainTip.Height)
	}

	lr := io.LimitReader(p.Connection(), int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&block); err != nil {
		return err
	}

	no.blockHeadersCh <- *block.Header

	return nil
}
