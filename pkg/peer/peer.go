package peer

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/vulpemventures/go-elements/block"
)

type Peer interface {
	GetBestBlockHeader() (*block.Header, error)
	GetBlockHeaderByHeight(height uint32) (*block.Header, error)
	GetCFilter(chainhash.Hash, wire.FilterType) (*chainhash.Hash, error)
}