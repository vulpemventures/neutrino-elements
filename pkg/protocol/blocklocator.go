package protocol

import (
	"bytes"
	"io"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
)

const (
	hashLen = 32
)

type BlockLocators [][hashLen]byte

var _ binary.Marshaler = (*BlockLocators)(nil)
var _ binary.Unmarshaler = (*BlockLocators)(nil)

func (locators BlockLocators) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})

	numberOfHashes := newFromInt(len(locators))
	// write the len as varint
	b, err := binary.Marshal(numberOfHashes)
	if err != nil {
		return nil, err
	}

	if _, err := buf.Write(b); err != nil {
		return nil, err
	}

	for _, locatorHash := range locators {
		b, err := binary.Marshal(locatorHash)
		if err != nil {
			return nil, err
		}

		if _, err := buf.Write(b); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (locators BlockLocators) UnmarshalBinary(r io.Reader) error {
	d := binary.NewDecoder(r)

	var count VarInt
	if err := d.Decode(&count); err != nil {
		return err
	}

	numberOfBlockLocatorHashes, err := count.Int()
	if err != nil {
		return err
	}

	hashes := make([][32]byte, numberOfBlockLocatorHashes)

	for i := 0; i < numberOfBlockLocatorHashes; i++ {
		var hash [hashLen]byte
		if err := d.Decode(&hash); err != nil {
			return err
		}
		hashes[i] = hash
	}

	locators = hashes
	return nil
}
