package repository

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/gcs"
	"github.com/btcsuite/btcd/btcutil/gcs/builder"
)

type FilterType byte

const (
	// only regular filter is supported for now
	RegularFilter FilterType = iota
)

// FilterKey is the unique key for a filter.
// for each possible key, the repository should store 1 unique filter
type FilterKey struct {
	BlockHash  []byte
	FilterType FilterType
}

func (k FilterKey) String() string {
	hashedKey := btcutil.Hash160(append(k.BlockHash, byte(k.FilterType)))
	return hex.EncodeToString(hashedKey[:6])
}

// FilterEntry is the base filter structure using to strore filter data.
type FilterEntry struct {
	Key    FilterKey
	NBytes []byte
}

func NewFilterEntry(key FilterKey, filter *gcs.Filter) (*FilterEntry, error) {
	nBytes, err := filter.NBytes()
	if err != nil {
		return nil, err
	}

	return &FilterEntry{
		Key:    key,
		NBytes: nBytes,
	}, nil
}

func (f *FilterEntry) GcsFilter() (*gcs.Filter, error) {
	return gcs.FromNBytes(builder.DefaultP, builder.DefaultM, f.NBytes)
}

var ErrFilterNotFound = errors.New("filter not found")

type FilterRepository interface {
	PutFilter(context.Context, *FilterEntry) error
	GetFilter(context.Context, FilterKey) (*FilterEntry, error)
}
