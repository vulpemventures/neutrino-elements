package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
)

var errInvalidVarIntValue = errors.New("invalid varint value")

// VarInt is variable length integer.
type VarInt struct {
	Value interface{}
}

var _ binary.Unmarshaler = (*VarInt)(nil)
var _ binary.Marshaler = (*VarInt)(nil)

func newFromInt(i int) VarInt {
	if i < 0 {
		return VarInt{Value: uint8(0)}
	}

	if i <= 0xFC {
		return VarInt{Value: uint8(i)}
	}

	if i <= 0xFFFF {
		return VarInt{Value: uint16(i)}
	}

	if i <= 0xFFFFFFFF {
		return VarInt{Value: uint32(i)}
	}

	return VarInt{Value: uint64(i)}
}

func NewVarint(u interface{}) (VarInt, error) {
	switch v := u.(type) {
	case int:
		return newFromInt(v), nil
	case uint8, uint16, uint32:
		return VarInt{Value: v}, nil
	case uint64:
		var err error = nil
		if v > math.MaxInt64 {
			v = math.MaxInt64
		}

		return VarInt{Value: v}, err
	}

	return VarInt{Value: nil}, errInvalidVarIntValue
}

// Int returns returns value as 'int'.
func (vi VarInt) Int() (int, error) {
	switch v := vi.Value.(type) {
	case uint8:
		return int(v), nil

	case uint16:
		return int(v), nil

	case uint32:
		return int(v), nil

	case uint64:
		// Assume we'll never get value more than MaxInt64.
		if v > math.MaxInt64 {
			return math.MaxInt64, nil
		}

		return int(v), nil
	}

	return 0, errInvalidVarIntValue
}

func (vi *VarInt) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})

	switch vi.Value.(type) {
	case uint16:
		if err := buf.WriteByte(0xFD); err != nil {
			return nil, err
		}
	case uint32:
		if err := buf.WriteByte(0xFE); err != nil {
			return nil, err
		}
	case uint64:
		if err := buf.WriteByte(0xFF); err != nil {
			return nil, err
		}
	case uint8:
		break
	default:
		return nil, fmt.Errorf("unknown type of varint value: %+v", vi.Value)
	}

	b, err := binary.MarshalForVarint(vi.Value)
	if err != nil {
		return nil, err
	}

	if _, err := buf.Write(b); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary implements binary.Unmarshaler interface.
func (vi *VarInt) UnmarshalBinary(r io.Reader) error {
	var b uint8

	lr := io.LimitReader(r, 1)
	if err := binary.NewDecoder(lr).Decode(&b); err != nil {
		return err
	}

	if b < 0xFD {
		vi.Value = b
		return nil
	}

	if b == 0xFD {
		lr := io.LimitReader(r, 2)
		v, err := binary.NewDecoder(lr).DecodeUint16ForVarint()
		if err != nil {
			return err
		}

		vi.Value = v
		return nil
	}

	if b == 0xFE {
		var v uint32
		lr := io.LimitReader(r, 4)
		if err := binary.NewDecoder(lr).Decode(&v); err != nil {
			return err
		}

		vi.Value = v
		return nil
	}

	if b == 0xFF {
		var v uint64
		lr := io.LimitReader(r, 8)
		if err := binary.NewDecoder(lr).Decode(&v); err != nil {
			return err
		}

		vi.Value = v
		return nil
	}

	return nil
}
