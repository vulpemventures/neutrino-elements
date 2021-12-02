package repository

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/vulpemventures/go-elements/block"
)

type BlockHeaderRepository interface {
	// chain tip returns the best block header in the store
	ChainTip() (*block.Header, error)
	GetBlock(chainhash.Hash) (*block.Header, error)
	WriteHeaders(...block.Header) error
}
