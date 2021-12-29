package node

import (
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

	sendHeaders, err := protocol.NewSendHeadersMessage(n.Network)
	if err != nil {
		return err
	}

	if err := n.sendMessage(conn, sendHeaders); err != nil {
		return err
	}

	verack, err := protocol.NewVerackMsg(n.Network)
	if err != nil {
		return err
	}

	if err := n.sendMessage(conn, verack); err != nil {
		return err
	}

	return nil
}
