package dbpg

import (
	"bytes"
	"context"
	"database/sql"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

type headerRepositoryImpl struct {
	db *DbService
}

func NewHeaderRepositoryImpl(db *DbService) (repository.BlockHeaderRepository, error) {
	return &headerRepositoryImpl{
		db: db,
	}, nil
}

type BlockHeader struct {
	Hash        string `db:"hash"`
	Height      uint32 `db:"height"`
	HeaderBytes []byte `db:"header_bytes"`
}

func (h *headerRepositoryImpl) ChainTip(
	ctx context.Context,
) (*block.Header, error) {
	query := `select * from block_header where height in (select max(height) from block_header);`

	blockHeader := &BlockHeader{}
	err := h.db.Db.Get(blockHeader, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNoBlocksHeaders
		}

		return nil, err
	}

	header, err := block.DeserializeHeader(bytes.NewBuffer(blockHeader.HeaderBytes))
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (h *headerRepositoryImpl) GetBlockHeader(
	ctx context.Context,
	hash chainhash.Hash,
) (*block.Header, error) {
	query := `select * from block_header where hash=$1;`

	blockHeader := &BlockHeader{}
	err := h.db.Db.Get(blockHeader, query, hash.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrBlockNotFound
		}

		return nil, err
	}

	header, err := block.DeserializeHeader(bytes.NewBuffer(blockHeader.HeaderBytes))
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (h *headerRepositoryImpl) GetBlockHashByHeight(
	ctx context.Context,
	height uint32,
) (*chainhash.Hash, error) {
	bh, err := h.getBlockHeaderByHeight(height)
	if err != nil {
		return nil, err
	}

	hash, err := bh.Hash()
	if err != nil {
		return nil, err
	}

	return &hash, nil
}

func (h *headerRepositoryImpl) WriteHeaders(
	ctx context.Context,
	header ...block.Header,
) error {
	tx, err := h.db.Db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, v := range header {
		headerBytes, err := v.Serialize()
		if err != nil {
			return err
		}

		hash, err := v.Hash()
		if err != nil {
			return err
		}

		bh := BlockHeader{
			Hash:        hash.String(),
			Height:      v.Height,
			HeaderBytes: headerBytes,
		}

		query := `INSERT INTO block_header (hash, height, header_bytes) VALUES (:hash, :height, :header_bytes);`

		if _, err = tx.NamedExec(query, bh); err != nil {
			if pqErr := err.(*pq.Error); pqErr != nil {
				if pqErr.Code == uniqueViolation {
					log.Warnf("header already exists: %s", hash.String())
					return nil
				}
			}
			return err
		}
	}

	return tx.Commit()
}

func (h *headerRepositoryImpl) LatestBlockLocator(
	ctx context.Context,
) (blockchain.BlockLocator, error) {
	tip, err := h.ChainTip(ctx)
	if err != nil {
		return nil, err
	}

	return h.blockLocatorFromHash(tip)
}

func (h *headerRepositoryImpl) HasAllAncestors(
	ctx context.Context,
	hash chainhash.Hash,
) (bool, error) {
	headers, err := h.getAllBlockHeaders()
	if err != nil {
		return false, err
	}

	var blockHeader *block.Header
	headersMap := make(map[chainhash.Hash]*block.Header)
	for _, v := range headers {
		h, err := v.Hash()
		if err != nil {
			return false, err
		}
		headersMap[h] = v

		if h == hash {
			blockHeader = v
		}
	}

	if blockHeader == nil {
		return false, repository.ErrBlockNotFound
	}

	for blockHeader.Height > 1 {
		currentHash, err := chainhash.NewHash(blockHeader.PrevBlockHash)
		if err != nil {
			return false, err
		}

		blockHeader = headersMap[*currentHash]
		if blockHeader == nil {
			return false, nil
		}
	}

	return true, nil
}

func (h *headerRepositoryImpl) blockLocatorFromHash(blck *block.Header) (blockchain.BlockLocator, error) {
	headers, err := h.getAllBlockHeaders()
	if err != nil {
		return nil, err
	}

	headersMap := make(map[uint32]*block.Header)
	for _, v := range headers {
		headersMap[v.Height] = v
	}

	var locator blockchain.BlockLocator

	if blck == nil {
		return nil, repository.ErrBlockNotFound
	}

	hash, err := blck.Hash()
	if err != nil {
		return nil, err
	}

	// Append the initial hash
	locator = append(locator, &hash)

	if blck.Height == 0 || err != nil {
		return locator, nil
	}

	height := blck.Height
	decrement := uint32(1)
	for height > 0 && len(locator) < wire.MaxBlockLocatorsPerMsg {
		blockHeader, _ := headersMap[height]

		headerHash, err := blockHeader.Hash()
		if err != nil {
			return nil, err
		}

		locator = append(locator, &headerHash)

		if decrement > height {
			height = 0
		} else {
			height -= decrement
		}

		// Decrement by 1 for the first 10 blocks, then double the jump
		// until we get to the genesis hash
		if len(locator) > 10 {
			decrement *= 2
		}
	}

	return locator, nil
}

func (h *headerRepositoryImpl) getBlockHeaderByHeight(height uint32) (*block.Header, error) {
	query := `select * from block_header where height=$1;`

	blockHeader := &BlockHeader{}
	err := h.db.Db.Get(blockHeader, query, height)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNoBlocksHeaders
		}

		return nil, err
	}

	header, err := block.DeserializeHeader(bytes.NewBuffer(blockHeader.HeaderBytes))
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (h *headerRepositoryImpl) getAllBlockHeaders() ([]*block.Header, error) {
	query := `select * from block_header;`

	blockHeaders := []*BlockHeader{}
	err := h.db.Db.Select(&blockHeaders, query)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNoBlocksHeaders
	}

	var headers []*block.Header
	for _, v := range blockHeaders {
		header, err := block.DeserializeHeader(bytes.NewBuffer(v.HeaderBytes))
		if err != nil {
			return nil, err
		}

		headers = append(headers, header)
	}

	return headers, nil
}
