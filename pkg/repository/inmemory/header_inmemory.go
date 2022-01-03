package inmemory

import (
	"sync"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

type headerInmemory struct {
	headers map[chainhash.Hash]*block.Header
	locker  *sync.RWMutex
}

var _ repository.BlockHeaderRepository = (*headerInmemory)(nil)

func NewHeaderInmemory() *headerInmemory {
	return &headerInmemory{
		headers: make(map[chainhash.Hash]*block.Header),
		locker:  new(sync.RWMutex),
	}
}

func (h *headerInmemory) ChainTip() (*block.Header, error) {
	h.locker.RLock()
	defer h.locker.RUnlock()

	if len(h.headers) == 0 {
		return nil, repository.ErrNoBlocksHeaders
	}

	var tip *block.Header = nil
	for _, val := range h.headers {
		if tip == nil || val.Height > tip.Height {
			tip = val
		}
	}

	return tip, nil
}

func (h *headerInmemory) GetBlockHeader(hash chainhash.Hash) (*block.Header, error) {
	h.locker.RLock()
	defer h.locker.RUnlock()

	blockHeader, ok := h.headers[hash]
	if !ok {
		return nil, repository.ErrBlockNotFound
	}

	return blockHeader, nil
}

func (h *headerInmemory) WriteHeaders(headers ...block.Header) error {
	h.locker.Lock()
	defer h.locker.Unlock()

	for _, header := range headers {
		hash, err := header.Hash()
		if err != nil {
			logrus.Error(err)
			continue
		}
		h.headers[hash] = &header
	}

	return nil
}

func (h *headerInmemory) LatestBlockLocator() (blockchain.BlockLocator, error) {
	tip, err := h.ChainTip()
	if err != nil {
		return nil, err
	}

	return h.blockLocatorFromHash(tip)
}

func (h *headerInmemory) getBlockHeaderByHeight(height uint32) (*block.Header, error) {
	h.locker.RLock()
	defer h.locker.RUnlock()

	for _, header := range h.headers {
		if header.Height == height {
			return header, nil
		}
	}

	return nil, repository.ErrBlockNotFound
}

func (h *headerInmemory) blockLocatorFromHash(chainTip *block.Header) (blockchain.BlockLocator, error) {
	var locator blockchain.BlockLocator

	if chainTip == nil {
		return nil, repository.ErrNoBlocksHeaders
	}

	hash, err := chainTip.Hash()
	if err != nil {
		return nil, err
	}

	// Append the initial hash
	locator = append(locator, &hash)

	if chainTip.Height == 0 || err != nil {
		return locator, nil
	}

	height := chainTip.Height
	decrement := uint32(1)
	for height > 0 && len(locator) < wire.MaxBlockLocatorsPerMsg {
		// Decrement by 1 for the first 10 blocks, then double the jump
		// until we get to the genesis hash
		if len(locator) > 10 {
			decrement *= 2
		}

		if decrement > height {
			height = 0
		} else {
			height -= decrement
		}

		blockHeader, err := h.getBlockHeaderByHeight(height)
		if err != nil {
			return locator, err
		}

		headerHash, err := blockHeader.Hash()
		if err != nil {
			return locator, err
		}

		locator = append(locator, &headerHash)
	}

	return locator, nil
}

func (h *headerInmemory) HasAllAncestors(hash chainhash.Hash) (bool, error) {
	h.locker.RLock()
	defer h.locker.RUnlock()

	if len(h.headers) == 0 {
		return false, repository.ErrNoBlocksHeaders
	}

	blockHeader := h.headers[hash]

	for blockHeader.Height > 1 {
		currentHash, err := chainhash.NewHash(blockHeader.PrevBlockHash)
		if err != nil {
			return false, err
		}

		blockHeader = h.headers[*currentHash]
		if blockHeader == nil {
			return false, nil
		}
	}

	return true, nil
}
