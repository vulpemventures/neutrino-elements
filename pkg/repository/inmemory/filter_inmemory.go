package inmemory

import (
	"context"
	"encoding/hex"
	"sync"

	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

func makeUniqueKey(key repository.FilterKey) string {
	return hex.EncodeToString(append(key.BlockHash, byte(key.FilterType)))
}

type FilterInmemory struct {
	filtersByHash map[string][]byte
	locker        *sync.RWMutex
}

func NewFilterInmemory() repository.FilterRepository {
	return &FilterInmemory{
		filtersByHash: make(map[string][]byte),
		locker:        new(sync.RWMutex),
	}
}

func (f *FilterInmemory) PutFilter(_ context.Context, entry *repository.FilterEntry) error {
	f.locker.Lock()
	defer f.locker.Unlock()

	key := makeUniqueKey(entry.Key)
	f.filtersByHash[key] = entry.NBytes
	return nil
}

func (f *FilterInmemory) GetFilter(_ context.Context, key repository.FilterKey) (*repository.FilterEntry, error) {
	f.locker.RLock()
	defer f.locker.RUnlock()

	k := makeUniqueKey(key)
	filter, ok := f.filtersByHash[k]
	if !ok {
		return nil, repository.ErrFilterNotFound
	}

	return &repository.FilterEntry{
		Key:    key,
		NBytes: filter,
	}, nil
}
