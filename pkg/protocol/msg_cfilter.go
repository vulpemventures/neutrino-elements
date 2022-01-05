package protocol

import (
	"bytes"
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
var _ binary.Marshaler = (*MsgCFilter)(nil)

func NewMsgCFilter(network Magic, blockHash *chainhash.Hash, filter *gcs.Filter) (*Message, error) {
	payload := &MsgCFilter{
		FilterType: 0,
		BlockHash:  blockHash,
		Filter:     filter,
	}

	return NewMessage("cfilter", network, payload)
}

func (msg *MsgCFilter) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})

	if err := buf.WriteByte(msg.FilterType); err != nil {
		return nil, err
	}

	b, err := binary.Marshal(msg.BlockHash)
	if err != nil {
		return nil, err
	}

	if _, err := buf.Write(b); err != nil {
		return nil, err
	}

	bytesFilter, err := msg.Filter.NBytes()
	if err != nil {
		return nil, err
	}

	filterLen := newFromInt(len(bytesFilter))
	b, err = binary.Marshal(filterLen)
	if err != nil {
		return nil, err
	}

	if _, err := buf.Write(b); err != nil {
		return nil, err
	}

	if _, err := buf.Write(bytesFilter); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (msg *MsgCFilter) UnmarshalBinary(r io.Reader) error {
	d := binary.NewDecoder(r)

	if err := d.Decode(&msg.FilterType); err != nil {
		return err
	}

	// invalid filter type
	if msg.FilterType != 0 {
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

	bytesFilter, err := d.DecodeBytes(int64(len))
	if err != nil {
		return err
	}

	gcsFilter, err := gcs.FromNBytes(
		builder.DefaultP, builder.DefaultM, bytesFilter,
	)

	if err != nil {
		return err
	}

	msg.Filter = gcsFilter

	return nil
}
