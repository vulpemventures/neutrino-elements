package protocol

import (
	"fmt"

	"github.com/vulpemventures/go-elements/block"
)

const (
	BIP157MaxHeightDiff = 1000
)

type MsgGetCFilters struct {
	FilterType  byte
	StartHeight uint32
	StopHash    [hashLen]byte
}

func NewGetCFilters(network Magic, start *block.Header, stop *block.Header) (*Message, error) {
	stopHash, err := stop.Hash()
	if err != nil {
		return nil, err
	}

	if stop.Height < start.Height {
		return nil, fmt.Errorf("getcfilters stopHeight must be greater or equal to startHeight")
	}

	if stop.Height-start.Height >= BIP157MaxHeightDiff {
		return nil, fmt.Errorf("diff (stopHeight-startHeight) must be strictly less than %+v", BIP157MaxHeightDiff)
	}

	getcfilters := &MsgGetCFilters{
		FilterType:  byte(0),
		StartHeight: start.Height,
		StopHash:    stopHash,
	}

	return NewMessage("getcfilters", network, getcfilters)
}
