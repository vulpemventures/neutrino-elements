package node

import (
	"io"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n node) handleCFilter(header *protocol.MessageHeader, p peer.Peer) error {
	var cfilter protocol.MsgCFilter

	lr := io.LimitReader(p.Connection(), int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&cfilter); err != nil {
		return err
	}

	// send the cfilter to the chan
	n.compactFiltersCh <- cfilter

	return nil
}
