package protocol

import (
	"crypto/sha256"
	"fmt"
	"io"
	"sort"

	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
)

// MsgTx represents 'tx' message.
type MsgTx struct {
	transaction.Transaction
}

// Hash returns transaction ID.
func (tx MsgTx) Hash() ([]byte, error) {
	serialized, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("tx.MarshalBinary: %+v", err)
	}

	hash := sha256.Sum256(serialized)
	hash = sha256.Sum256(hash[:])

	txid := hash[:]

	sort.SliceStable(txid, func(i, j int) bool {
		return true
	})

	return txid, nil
}

// MarshalBinary implements binary.Marshaler interface.
func (tx MsgTx) MarshalBinary() ([]byte, error) {
	return tx.Serialize()
}

// UnmarshalBinary implements binary.Unmarshaler
func (tx *MsgTx) UnmarshalBinary(r io.Reader) error {
	d := binary.NewDecoder(r)
	bytes, err := d.ReadUntilEOF()
	if err != nil {
		return err
	}

	decodedTx, err := transaction.NewTxFromBuffer(&bytes)
	if err != nil {
		return err
	}

	tx.Transaction = *decodedTx

	return nil
}
