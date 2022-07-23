package pgtest

import (
	"encoding/hex"
	"github.com/btcsuite/btcd/btcutil/gcs"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

func (s *PgDbTestSuite) TestGetFilter() {
	blockHash := "db262c78cff2454b5dbc811d1e4782e7bd22c04a4a328ad561513dbaab12373a"
	blockHashBytes, err := hex.DecodeString(blockHash)
	if err != nil {
		s.FailNow(err.Error())
	}
	key := repository.FilterKey{
		BlockHash:  blockHashBytes,
		FilterType: repository.RegularFilter,
	}

	filter, err := filterRepo.GetFilter(ctx, key)
	if err != nil {
		s.FailNow(err.Error())
	}

	s.Equal("2df74a01a958", filter.Key.String())
	s.Equal("04cf4244501198a056b38200", string(filter.NBytes))
}

func (s *PgDbTestSuite) TestPutFilter() {
	var k [gcs.KeySize]byte
	contents := [][]byte{
		[]byte("dummy"),
	}
	filter, err := gcs.BuildGCSFilter(uint8(19), 784931, k, contents)
	if err != nil {
		s.FailNow(err.Error())
	}

	blockHash := "ab262c78cff2454b5dbc811d1e4782e7bd22c04a4a328ad561513dbaab12373a"
	blockHashBytes, err := hex.DecodeString(blockHash)
	if err != nil {
		s.FailNow(err.Error())
	}

	key := repository.FilterKey{
		BlockHash:  blockHashBytes,
		FilterType: repository.RegularFilter,
	}

	entry, err := repository.NewFilterEntry(key, filter)
	if err != nil {
		s.FailNow(err.Error())
	}

	if err := filterRepo.PutFilter(ctx, entry); err != nil {
		s.FailNow(err.Error())
	}

	f, err := filterRepo.GetFilter(ctx, key)
	if err != nil {
		s.FailNow(err.Error())
	}

	s.Equal(key.String(), f.Key.String())
}
