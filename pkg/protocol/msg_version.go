package protocol

import (
	"math/rand"
	"time"
)

// MsgVersion ...
type MsgVersion struct {
	Version     int32
	Services    uint64
	Timestamp   int64
	AddrRecv    VersionNetAddr
	AddrFrom    VersionNetAddr
	Nonce       uint64
	UserAgent   VarStr
	StartHeight int32
	Relay       bool
}

// NewVersionMsg returns a new MsgVersion.
func NewVersionMsg(network Magic, userAgent string, peerIP IPv4, peerPort uint16, Services ServiceFlag) (*Message, error) {
	payload := MsgVersion{
		Version:   Version,
		Services:  uint64(Services),
		Timestamp: time.Now().UTC().Unix(),
		AddrRecv: VersionNetAddr{
			Services: uint64(Services) | uint64(SFNodeNetwork),
			IP:       peerIP,
			Port:     peerPort,
		},
		AddrFrom: VersionNetAddr{
			Services: uint64(Services),
			IP:       NewIPv4(127, 0, 0, 1),
			Port:     9334,
		},
		Nonce:       rand.Uint64(),
		UserAgent:   NewUserAgent(userAgent),
		StartHeight: -1,
		Relay:       true,
	}

	msg, err := NewMessage("version", network, payload)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (msg MsgVersion) HasService(service ServiceFlag) bool {
	return msg.Services&uint64(service) == uint64(service)
}
