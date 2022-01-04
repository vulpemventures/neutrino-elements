package node

import (
	"io"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n Node) handlePong(header *protocol.MessageHeader, p peer.Peer) error {
	var pong protocol.MsgPing

	lr := io.LimitReader(p.Connection(), int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&pong); err != nil {
		return err
	}

	n.pongsCh <- pong.Nonce

	return nil
}
