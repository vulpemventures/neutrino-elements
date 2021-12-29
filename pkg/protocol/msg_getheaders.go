package protocol

import (
	"io"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
)

type MsgGetHeaders struct {
	Version            uint32
	BlockLocatorHashes [][32]byte
	HashStop           [32]byte
}

func (msg *MsgGetHeaders) UnmarshalBinary(r io.Reader) error {
	d := binary.NewDecoder(r)
	if err := d.Decode(&msg.Version); err != nil {
		return err
	}

	var count VarInt
	if err := d.Decode(&count); err != nil {
		return err
	}

	numberOfBlockLocatorHashes, err := count.Int()
	if err != nil {
		return err
	}

	for i := 0; i < numberOfBlockLocatorHashes; i++ {
		var hash [32]byte
		if err := d.Decode(&hash); err != nil {
			return err
		}
		msg.BlockLocatorHashes = append(msg.BlockLocatorHashes, hash)
	}

	if err := d.Decode(&msg.HashStop); err != nil {
		return err
	}

	return nil
}
