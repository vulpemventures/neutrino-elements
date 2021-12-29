package node

import (
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n Node) handleGetHeaders(header *protocol.MessageHeader, conn io.ReadWriter) error {
	var getHeaders protocol.MsgGetHeaders
	lr := io.LimitReader(conn, int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&getHeaders); err != nil {
		return err
	}

	logrus.Debugf("getheaders: %+v", getHeaders.HashStop)

	return nil
}
