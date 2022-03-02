package node

import (
	"io"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (no node) handleInv(header *protocol.MessageHeader, p peer.Peer) error {
	var inv protocol.MsgInv

	conn := p.Connection()
	lr := io.LimitReader(conn, int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&inv); err != nil {
		return err
	}

	var getData protocol.MsgGetData
	getData.Inventory = inv.Inventory
	getData.Count = inv.Count

	getDataMsg, err := protocol.NewMessage("getdata", no.Network, getData)
	if err != nil {
		return err
	}

	msg, err := binary.Marshal(getDataMsg)
	if err != nil {
		return err
	}

	if _, err := conn.Write(msg); err != nil {
		return err
	}

	return nil
}
