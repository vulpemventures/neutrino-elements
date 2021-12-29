package repository

import (
	"errors"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/vulpemventures/go-elements/block"
)

var (
	ErrBlockNotFound   = errors.New("block not found")
	ErrNoBlocksHeaders = errors.New("no block headers in repository")
)

type BlockHeaderRepository interface {
	// chain tip returns the best block header in the store
	ChainTip() (*block.Header, error)
	GetBlock(chainhash.Hash) (*block.Header, error)
	WriteHeaders(...block.Header) error
}
