package node

import (
	"io"

	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n Node) handleVerack(header *protocol.MessageHeader, conn io.ReadWriter) error {
	return nil
}
