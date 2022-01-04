package node

import (
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n Node) handleSendCmpct(header *protocol.MessageHeader, p peer.Peer) error {
	var sendCmpct protocol.MsgSendCmpct
	lr := io.LimitReader(p.Connection(), int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&sendCmpct); err != nil {
		return err
	}

	logrus.Debugf("sendcmpct: %+v", sendCmpct.LowBandwitdhType)

	return nil
}
