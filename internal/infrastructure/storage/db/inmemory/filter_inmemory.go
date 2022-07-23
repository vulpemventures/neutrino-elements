package inmemory

import (
	"context"
	"sync"

	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

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

	f.filtersByHash[entry.Key.String()] = entry.NBytes
	return nil
}

func (f *FilterInmemory) GetFilter(_ context.Context, key repository.FilterKey) (*repository.FilterEntry, error) {
	f.locker.RLock()
	defer f.locker.RUnlock()

	filter, ok := f.filtersByHash[key.String()]
	if !ok {
		return nil, repository.ErrFilterNotFound
	}

	return &repository.FilterEntry{
		Key:    key,
		NBytes: filter,
	}, nil
}
