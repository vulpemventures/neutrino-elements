package inmemory

import (
	"sync"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil/gcs"
	"github.com/btcsuite/btcutil/gcs/builder"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

type FilterInmemory struct {
	filtersByHash map[string][]byte
	locker        *sync.RWMutex
}

var _ repository.FilterRepository = (*FilterInmemory)(nil)

func NewFilterInmemory() *FilterInmemory {
	return &FilterInmemory{
		filtersByHash: make(map[string][]byte),
		locker:        new(sync.RWMutex),
	}
}

func (f *FilterInmemory) PutFilter(blockHash *chainhash.Hash, filter *gcs.Filter, filterType repository.FilterType) error {
	f.locker.Lock()
	defer f.locker.Unlock()

	filterBytes, err := filter.NBytes()
	if err != nil {
		return err
	}

	f.filtersByHash[blockHash.String()] = filterBytes
	return nil
}

func (f *FilterInmemory) FetchFilter(blockHash *chainhash.Hash, filterType repository.FilterType) (*gcs.Filter, error) {
	f.locker.RLock()
	defer f.locker.RUnlock()

	filter, ok := f.filtersByHash[blockHash.String()]
	if !ok {
		return nil, repository.ErrFilterNotFound
	}

	gcsFilter, err := gcs.FromNBytes(builder.DefaultP, builder.DefaultM, filter)
	if err != nil {
		return nil, err
	}

	return gcsFilter, nil
}
