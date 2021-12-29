package inmemory

import (
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil/gcs"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

type FilterInmemory struct {
	filtersByHash map[chainhash.Hash]gcs.Filter
	locker        *sync.RWMutex
}

var _ repository.FilterRepository = (*FilterInmemory)(nil)

func NewFilterInmemory() *FilterInmemory {
	return &FilterInmemory{
		filtersByHash: make(map[chainhash.Hash]gcs.Filter),
		locker:        new(sync.RWMutex),
	}
}

func (f *FilterInmemory) PutFilter(blockHash *chainhash.Hash, filter *gcs.Filter, filterType repository.FilterType) error {
	f.locker.Lock()
	defer f.locker.Unlock()

	f.filtersByHash[*blockHash] = *filter
	return nil
}

func (f *FilterInmemory) FetchFilter(blockHash *chainhash.Hash, filterType repository.FilterType) (*gcs.Filter, error) {
	f.locker.RLock()
	defer f.locker.RUnlock()

	filter, ok := f.filtersByHash[*blockHash]
	if !ok {
		return nil, repository.ErrFilterNotFound
	}

	return &filter, nil
}

func (f *FilterInmemory) PurgeFilters(filterType repository.FilterType) error {
	f.locker.Lock()
	defer f.locker.Unlock()

	f.filtersByHash = make(map[chainhash.Hash]gcs.Filter)
	return nil
}
