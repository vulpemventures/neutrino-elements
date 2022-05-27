package node

import (
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (no node) handleTx(header *protocol.MessageHeader, p peer.Peer) error {
	var tx protocol.MsgTx

	lr := io.LimitReader(p.Connection(), int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&tx); err != nil {
		return err
	}

	logrus.Debugf("transaction: %x", tx.HashStr())

	no.memPool.AddTx(tx)

	return nil
}
