package node

import (
	"io"
	"net"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n Node) handleHeaders(msgHeader *protocol.MessageHeader, conn net.Conn) error {
	var headers protocol.MsgHeaders

	lr := io.LimitReader(conn, int64(msgHeader.Length))

	if err := binary.NewDecoder(lr).Decode(&headers); err != nil {
		return err
	}

	for _, header := range headers.Headers {
		n.blockHeadersCh <- *header
	}

	return nil
}
