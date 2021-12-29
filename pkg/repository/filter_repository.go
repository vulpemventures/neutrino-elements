package repository

import (
	"errors"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil/gcs"
)

type FilterType uint8

const (
	// only regular filter is supported for now
	RegularFilter FilterType = iota
)

var ErrFilterNotFound = errors.New("filter not found")

type FilterRepository interface {
	PutFilter(*chainhash.Hash, *gcs.Filter, FilterType) error
	FetchFilter(*chainhash.Hash, FilterType) (*gcs.Filter, error)
	PurgeFilters(FilterType) error
}
