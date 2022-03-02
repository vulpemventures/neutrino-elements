package node

import (
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n node) handleGetHeaders(header *protocol.MessageHeader, p peer.Peer) error {
	var getHeaders protocol.MsgGetHeaders
	lr := io.LimitReader(p.Connection(), int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&getHeaders); err != nil {
		return err
	}

	logrus.Debugf("getheaders: %+v", getHeaders.HashStop)

	return nil
}
