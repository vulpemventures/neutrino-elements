package protocol_test

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func TestMessageSerialization(t *testing.T) {
	version := protocol.MsgVersion{
		Version:   protocol.Version,
		Services:  uint64(protocol.SFNodeCF),
		Timestamp: time.Date(2019, 11, 11, 0, 0, 0, 0, time.UTC).Unix(),
		AddrRecv: protocol.VersionNetAddr{
			Services: uint64(protocol.SFNodeCF),
			IP:       protocol.NewIPv4(127, 0, 0, 1),
			Port:     9333,
		},
		AddrFrom: protocol.VersionNetAddr{
			Services: uint64(protocol.SFNodeCF),
			IP:       protocol.NewIPv4(127, 0, 0, 1),
			Port:     9334,
		},
		Nonce:       31337,
		UserAgent:   protocol.NewUserAgent("/test:0.1.0/"),
		StartHeight: -1,
		Relay:       true,
	}
	msg, err := protocol.NewMessage("version", protocol.MagicNigiri, version)
	if err != nil {
		t.Errorf("unexpected error: %+v", err)
		return
	}

	msgSerialized, err := binary.Marshal(msg)
	if err != nil {
		t.Errorf("unexpected error: %+v", err)
		return
	}

	actual := hex.EncodeToString(msgSerialized)
	expected := "1234567876657273696f6e000000000062000000ada9766e80110100400000000000000080a4c85d00000000400000000000000000000000000000000000ffff7f0000012475400000000000000000000000000000000000ffff7f0000012476697a0000000000000c2f746573743a302e312e302fffffffff01"
	if actual != expected {
		t.Errorf("expected: %s, actual: %s", expected, actual)
	}

}

func TestHasValidCommand(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"version", true},
		{"invalid", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			var packed [12]byte
			buf := make([]byte, 12-len(test.name))
			copy(packed[:], append([]byte(test.name), buf...)[:])

			mh := protocol.MessageHeader{
				Command: packed,
			}
			actual := mh.HasValidCommand()

			if actual != test.expected {
				t.Errorf("expected: %v, actual: %v", test.expected, actual)
			}
		})
	}
}

func TestHasValidMagic(t *testing.T) {
	magicMainnet := protocol.MagicLiquid
	magicSimnet := protocol.MagicNigiri
	magicTestnet := protocol.MagicLiquidTestnet

	tests := []struct {
		name     string
		magic    [4]byte
		expected bool
	}{
		{"liquid", magicMainnet, true},
		{"nigiri", magicSimnet, true},
		{"liquid testnet", magicTestnet, true},
		{"invalid", [4]byte{0xde, 0xad, 0xbe, 0xef}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mh := protocol.MessageHeader{
				Magic: test.magic,
			}
			actual := mh.HasValidMagic()

			if actual != test.expected {
				t.Errorf("expected: %v, actual: %v", test.expected, actual)
			}
		})
	}
}
