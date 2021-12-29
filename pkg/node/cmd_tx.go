package node

import (
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (no Node) handleTx(header *protocol.MessageHeader, conn io.ReadWriter) error {
	var tx protocol.MsgTx

	lr := io.LimitReader(conn, int64(header.Length))
	if err := binary.NewDecoder(lr).Decode(&tx); err != nil {
		return err
	}

	hash, err := tx.Hash()
	if err != nil {
		return fmt.Errorf("tx.Hash: %+v", err)
	}

	logrus.Debugf("transaction: %x", hash)

	return nil
}
