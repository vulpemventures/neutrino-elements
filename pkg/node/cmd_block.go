package node

import (
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

func (no Node) handleBlock(header *protocol.MessageHeader, p peer.Peer) error {
	var block protocol.MsgBlock

	currentChainTip, err := no.blockHeadersDb.ChainTip()
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

	hash, err := block.Header.Hash()
	if err != nil {
		return fmt.Errorf("block.Hash: %+v", err)
	}

	logrus.Debugf("block: %s", hash.String())

	no.blockHeadersCh <- *block.Header

	return nil
}
