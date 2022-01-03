package repository

import (
	"errors"

	"github.com/btcsuite/btcd/blockchain"
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
	GetBlockHeader(chainhash.Hash) (*block.Header, error)
	WriteHeaders(...block.Header) error
	// LatestBlockLocator returns the block locator for the latest known tip as root of the locator
	LatestBlockLocator() (blockchain.BlockLocator, error)
	HasAllAncestors(chainhash.Hash) (bool, error)
}
