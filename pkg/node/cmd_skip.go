package node

import (
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

// skipMessage is using to skip unhandled messages coming from peers
func (n node) skipMessage(header *protocol.MessageHeader, p peer.Peer) error {
	logrus.Debugf("skipping message: %s", header.Command)

	lr := io.LimitReader(p.Connection(), int64(header.Length))
	if _, err := binary.NewDecoder(lr).ReadUntilEOF(); err != nil {
		return err
	}

	return nil
}
