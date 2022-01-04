package node

import (
	"io"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n Node) handlePing(header *protocol.MessageHeader, p peer.Peer) error {
	var ping protocol.MsgPing

	conn := p.Connection()
	lr := io.LimitReader(conn, int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&ping); err != nil {
		return err
	}

	pong, err := protocol.NewPongMsg(n.Network, ping.Nonce)
	if err != nil {
		return err
	}

	msg, err := binary.Marshal(pong)
	if err != nil {
		return err
	}

	if _, err := conn.Write(msg); err != nil {
		return err
	}

	return nil
}
