package protocol

import (
	"fmt"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

const (
	maxBlockLocatorsPerMsg = 500
)

type MsgGetHeaders struct {
	Version            uint32
	BlockLocatorHashes BlockLocators
	HashStop           [hashLen]byte
}

func NewMsgGetHeaders(network Magic, hashStop [hashLen]byte, blockLocator blockchain.BlockLocator) (*Message, error) {
	payload := &MsgGetHeaders{
		Version:            Version,
		BlockLocatorHashes: [][hashLen]byte{},
		HashStop:           hashStop,
	}

	for _, hash := range blockLocator {
		err := payload.addBlockLocatorHash(hash)
		if err != nil {
			return nil, err
		}
	}

	return NewMessage("getheaders", network, payload)
}

func (msg *MsgGetHeaders) addBlockLocatorHash(hash *chainhash.Hash) error {
	if len(msg.BlockLocatorHashes)+1 > maxBlockLocatorsPerMsg {
		return fmt.Errorf("too many block locator hashes in getheaders message (max: %d)", maxBlockLocatorsPerMsg)
	}

	msg.BlockLocatorHashes = append(msg.BlockLocatorHashes, *hash)
	return nil
}
