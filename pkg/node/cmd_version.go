package node

import (
	"fmt"
	"io"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n Node) handleVersion(header *protocol.MessageHeader, conn net.Conn) error {
	var version protocol.MsgVersion

	lr := io.LimitReader(conn, int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&version); err != nil {
		return err
	}

	peer := Peer{
		Address:    conn.RemoteAddr(),
		Connection: conn,
		PongCh:     make(chan uint64),
		Services:   version.Services,
		UserAgent:  version.UserAgent.String,
		Version:    version.Version,
	}

	n.Peers[peer.ID()] = &peer
	go n.monitorPeer(&peer)
	logrus.Debugf("new peer %s", peer)

	// check if the peer supports compact block filters
	if !version.HasService(protocol.SFNodeCF) {
		return fmt.Errorf("peer %s does not support Compact Filters Service (BIP0158)", peer.ID())
	}

	verack, err := protocol.NewVerackMsg(n.Network)
	if err != nil {
		return err
	}

	if err := n.sendMessage(conn, verack); err != nil {
		return err
	}

	sendHeaders, err := protocol.NewSendHeadersMessage(n.Network)
	if err != nil {
		return err
	}

	if err := n.sendMessage(conn, sendHeaders); err != nil {
		return err
	}

	isSync, _ := n.isSync()
	if !isSync {
		logrus.Infof("%s is not syncing", peer)
		err = n.syncWithPeer(peer.ID())
		if err != nil {
			return err
		}
	}

	return nil
}
