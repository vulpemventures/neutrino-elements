package protocol

import (
	"io"

	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/binary"
)

type MsgHeaders struct {
	Headers []*block.Header
}

var _ binary.Unmarshaler = (*MsgHeaders)(nil)

func (msgHeaders *MsgHeaders) UnmarshalBinary(r io.Reader) error {
	d := binary.NewDecoder(r)

	var count VarInt
	if err := d.Decode(&count); err != nil {
		return err
	}

	numberOfHeaders, err := count.Int()
	if err != nil {
		return err
	}

	headersBytes, err := d.ReadUntilEOF()
	if err != nil {
		return err
	}

	headers := make([]*block.Header, numberOfHeaders)

	for i := 0; i < numberOfHeaders; i++ {
		header, err := block.DeserializeHeader(&headersBytes)
		if err != nil {
			return err
		}

		headers[i] = header

		// assume that we read a "block" with zero transactions
		var tmpLen VarInt
		err = binary.NewDecoder(&headersBytes).Decode(&tmpLen)
		if err != nil {
			return err
		}
	}

	msgHeaders.Headers = headers
	return nil
}
