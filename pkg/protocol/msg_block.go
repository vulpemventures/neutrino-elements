package protocol

import (
	"io"

	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
)

// MsgBlock represents 'block' message.
type MsgBlock struct {
	block.Block
}

var _ binary.Unmarshaler = (*MsgBlock)(nil)

// UnmarshalBinary implements binary.Unmarshaler
func (blck *MsgBlock) UnmarshalBinary(r io.Reader) error {
	d := binary.NewDecoder(r)
	bytes, err := d.ReadUntilEOF()
	if err != nil {
		return err
	}

	decodedBlock, err := block.NewFromBuffer(&bytes)
	if err != nil {
		return err
	}

	blck.Header = decodedBlock.Header
	blck.TransactionsData = &block.Transactions{
		Transactions: decodedBlock.TransactionsData.Transactions,
	}

	logrus.Debug(blck.TransactionsData.Transactions)
	return nil
}
