package protocol

import (
	"fmt"
	"io"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil/gcs"
	"github.com/btcsuite/btcutil/gcs/builder"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
)

type MsgCFilter struct {
	FilterType uint8
	BlockHash  *chainhash.Hash
	Filter     *gcs.Filter
}

var _ binary.Unmarshaler = (*MsgCFilter)(nil)

func (msg *MsgCFilter) UnmarshalBinary(r io.Reader) error {
	d := binary.NewDecoder(r)

	if err := d.Decode(&msg.FilterType); err != nil {
		return err
	}

	// invalid filter type
	if msg.FilterType == 255 {
		return fmt.Errorf("invalid filter type")
	}

	var blockHeaderHash [hashLen]byte
	if err := d.Decode(&blockHeaderHash); err != nil {
		return err
	}

	hash, err := chainhash.NewHash(blockHeaderHash[:])
	if err != nil {
		return err
	}

	msg.BlockHash = hash

	var lenFilter VarInt
	if err := d.Decode(&lenFilter); err != nil {
		return err
	}

	len, err := lenFilter.Int()
	if err != nil {
		return err
	}

	bytesFilter, err := d.ReadUntilEOF()
	if err != nil {
		return err
	}

	filterEncoded := bytesFilter.Bytes()[:len]

	gcsFilter, err := gcs.FromNBytes(
		builder.DefaultP, builder.DefaultM, filterEncoded,
	)

	if err != nil {
		return err
	}

	msg.Filter = gcsFilter

	return nil
}
