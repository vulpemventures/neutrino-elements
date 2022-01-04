package node

import (
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n Node) handleVersion(header *protocol.MessageHeader, p peer.Peer) error {
	var version protocol.MsgVersion

	conn := p.Connection()
	lr := io.LimitReader(conn, int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&version); err != nil {
		return err
	}

	// check if the peer supports compact block filters
	if !version.HasService(protocol.SFNodeCF) {
		return fmt.Errorf("peer %s does not support Compact Filters Service (BIP0158)", p.ID())
	}

	verack, err := protocol.NewVerackMsg(n.Network)
	if err != nil {
		return err
	}

	if err := n.sendMessage(conn, verack); err != nil {
		return err
	}

	// notify the peer that we would like to receive block header via headers messages
	sendHeaders, err := protocol.NewSendHeadersMessage(n.Network)
	if err != nil {
		return err
	}

	if err := n.sendMessage(conn, sendHeaders); err != nil {
		return err
	}

	n.addPeer(p)
	go n.monitorPeer(p)
	logrus.Debugf("new peer %s", p.ID())

	return nil
}
