package inmemory

import (
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

type HeaderInmemory struct {
	headers map[chainhash.Hash]*block.Header
	locker  *sync.RWMutex
}

var _ repository.BlockHeaderRepository = (*HeaderInmemory)(nil)

func NewHeaderInmemory() *HeaderInmemory {
	return &HeaderInmemory{
		headers: make(map[chainhash.Hash]*block.Header),
		locker:  new(sync.RWMutex),
	}
}

func (h *HeaderInmemory) ChainTip() (*block.Header, error) {
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

func (h *HeaderInmemory) GetBlock(hash chainhash.Hash) (*block.Header, error) {
	h.locker.RLock()
	defer h.locker.RUnlock()

	blockHeader, ok := h.headers[hash]
	if !ok {
		return nil, repository.ErrBlockNotFound
	}

	return blockHeader, nil
}

func (h *HeaderInmemory) WriteHeaders(headers ...block.Header) error {
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
