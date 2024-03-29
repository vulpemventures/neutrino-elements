package protocol

import (
	"crypto/sha256"
	"fmt"
	"io"
	"strings"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
)

const (
	checksumLength = 4
	nodeNetwork    = 1
	magicLength    = 4

	// MsgHeaderLength specifies the length of Message in bytes
	MsgHeaderLength = magicLength + commandLength + checksumLength + 4 // 4 - payload length value
)

// MessageHeader ...
type MessageHeader struct {
	Magic    [magicLength]byte
	Command  [commandLength]byte
	Length   uint32
	Checksum [checksumLength]byte
}

// Message ...
type Message struct {
	MessageHeader
	Payload []byte
}

// NewMessage returns a new Message.
func NewMessage(cmd string, networkMagic Magic, payload interface{}) (*Message, error) {
	serializedPayload, err := binary.Marshal(payload)
	if err != nil {
		return nil, err
	}

	command, ok := commands[cmd]
	if !ok {
		return nil, fmt.Errorf("unsupported command %s", cmd)
	}

	msg := Message{
		MessageHeader: MessageHeader{
			Magic:    networkMagic,
			Command:  command,
			Length:   uint32(len(serializedPayload)),
			Checksum: checksum(serializedPayload),
		},
		Payload: serializedPayload,
	}

	return &msg, nil
}

// CommandString returns command as a string with zero bytes removed.
func (mh MessageHeader) CommandString() string {
	return strings.Trim(string(mh.Command[:]), "\x00")
}

// Validate ...
func (mh MessageHeader) Validate() error {
	if !mh.HasValidMagic() {
		return fmt.Errorf("invalid magic: %x", mh.Magic)
	}

	if !mh.HasValidCommand() {
		return fmt.Errorf("invalid command: %s", mh.CommandString())
	}
	return nil
}

// HasValidCommand returns true if the message header contains a supported command.
// Returns false otherwise.
func (mh MessageHeader) HasValidCommand() bool {
	_, ok := commands[mh.CommandString()]
	return ok
}

// HasValidMagic returns true if the message header contains a supported magic.
// Returns false otherwise.
func (mh MessageHeader) HasValidMagic() bool {
	switch mh.Magic {
	case MagicLiquid, MagicLiquidTestnet, MagicRegtest:
		return true
	}

	return false
}

// VarStr ...
type VarStr struct {
	Length uint8
	String string
}

func newVarStr(str string) VarStr {
	return VarStr{
		Length: uint8(len(str)), // TODO: implement var_int
		String: str,
	}
}

// UnmarshalBinary implements the binary.Unmarshaler interface
func (v *VarStr) UnmarshalBinary(r io.Reader) error {
	lengthBuf := make([]byte, 1)
	if _, err := r.Read(lengthBuf); err != nil {
		return fmt.Errorf("varStr.UnmarshalBinary: %+v", err)
	}
	v.Length = uint8(lengthBuf[0])

	stringBuf := make([]byte, v.Length)
	if _, err := r.Read(stringBuf); err != nil {
		return fmt.Errorf("varStr.UnmarshalBinary: %+v", err)
	}
	v.String = string(stringBuf)

	return nil
}

func checksum(data []byte) [checksumLength]byte {
	hash := sha256.Sum256(data)
	hash = sha256.Sum256(hash[:])
	var hashArr [checksumLength]byte
	copy(hashArr[:], hash[0:checksumLength])

	return hashArr
}
