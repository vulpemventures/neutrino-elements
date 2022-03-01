package scanner

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil/gcs/builder"
	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/neutrino-elements/pkg/blockservice"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
)

type Report struct {
	// Transaction is the transaction that includes the item that was found.
	Transaction *transaction.Transaction

	// BlockHash is the block hash of the block that includes the transaction.
	BlockHash   *chainhash.Hash
	BlockHeight uint32
}

type ScannerService interface {
	// Start runs a go-routine in order to handle incoming requests via Watch
	Start() (<-chan Report, error)
	// Stop the scanner
	Stop() error
	Watch(...ScanRequestOption) error
}

type scannerService struct {
	started       bool
	requestsQueue *scanRequestQueue
	filterDB      repository.FilterRepository
	headerDB      repository.BlockHeaderRepository
	genesisHash   *chainhash.Hash
	blockService  blockservice.BlockService
	quitCh        chan struct{}
}

var _ ScannerService = (*scannerService)(nil)

func New(
	filterDB repository.FilterRepository,
	headerDB repository.BlockHeaderRepository,
	blockSvc blockservice.BlockService,
	genesisHash *chainhash.Hash,
) ScannerService {
	return &scannerService{
		requestsQueue: newScanRequestQueue(),
		filterDB:      filterDB,
		headerDB:      headerDB,
		blockService:  blockSvc,
		quitCh:        make(chan struct{}),
		genesisHash:   genesisHash,
	}
}

func (s *scannerService) Start() (<-chan Report, error) {
	if s.started {
		return nil, fmt.Errorf("utxo scanner already started")
	}

	s.quitCh = make(chan struct{}, 1)
	resultCh := make(chan Report)
	// start the requests manager
	go s.requestsManager(resultCh)

	s.started = true
	return resultCh, nil
}

func (s *scannerService) Stop() error {
	s.quitCh <- struct{}{}
	s.started = false
	return nil
}

func (s *scannerService) Watch(opts ...ScanRequestOption) error {
	req := newScanRequest(opts...)
	s.requestsQueue.enqueue(req)
	return nil
}

// requestsManager is responsible to resolve the requests that are waiting for in the queue.
func (s *scannerService) requestsManager(ch chan<- Report) {
	defer close(s.quitCh)

	for {
		s.requestsQueue.cond.L.Lock()
		for s.requestsQueue.isEmpty() {
			logrus.Debug("scanner queue is empty, waiting for new requests")
			s.requestsQueue.cond.Wait() // wait for new requests

			// check if we should quit the routine
			select {
			case <-s.quitCh:
				s.requestsQueue.cond.L.Unlock()
				return
			default:
			}
		}
		s.requestsQueue.cond.L.Unlock()

		// get the next request without removing it from the queue
		nextRequest := s.requestsQueue.peek()
		err := s.requestWorker(nextRequest.StartHeight, ch)
		if err != nil {
			logrus.Errorf("error while scanning: %v", err)
		}

		// check if we should quit the routine
		select {
		case <-s.quitCh:
			return
		default:
			continue
		}

	}
}

// will check if any blocks has the requested item
// if yes, will extract the transaction that match the item
// TODO handle properly errors (enqueue the unresolved requests ??)
func (s *scannerService) requestWorker(startHeight uint32, ch chan<- Report) error {
	nextBatch := make([]*ScanRequest, 0)
	nextHeight := startHeight

	chainTip, err := s.headerDB.ChainTip()
	if err != nil {
		return err
	}

	for nextHeight <= chainTip.Height {
		// append all the requests with start height = nextHeight
		nextBatch = append(nextBatch, s.requestsQueue.dequeueAtHeight(nextHeight)...)

		itemsBytes := make([][]byte, len(nextBatch))
		for i, req := range nextBatch {
			itemsBytes[i] = req.Item.Bytes()
		}

		// get the block hash for height
		var blockHash *chainhash.Hash
		if nextHeight == 0 {
			blockHash = s.genesisHash
		} else {
			blockHash, err = s.headerDB.GetBlockHashByHeight(nextHeight)
			if err != nil {
				return err
			}
		}

		// check with filterDB if the block has one of the items
		matched, err := s.blockFilterMatches(itemsBytes, blockHash)
		if err != nil {
			return err
		}

		if matched {
			reports, remainReqs, err := s.extractBlockMatches(blockHash, nextBatch)
			if err != nil {
				return err
			}

			// if some requests generate a report, send them to the channel
			for _, report := range reports {
				ch <- report
			}

			// if some requests remain, put them back in the next batch
			// this will remove the resolved requests from the batch
			nextBatch = remainReqs
		}

		// increment the height to scan
		// if nothing was found, we can just continue with same batch and next height
		nextHeight++

		chainTip, err = s.headerDB.ChainTip()
		if err != nil {
			return err
		}
	}

	// enqueue the remaining requests
	for _, req := range nextBatch {
		s.requestsQueue.enqueue(req)
	}

	return nil
}

func (s *scannerService) blockFilterMatches(items [][]byte, blockHash *chainhash.Hash) (bool, error) {
	filterToFetchKey := repository.FilterKey{
		BlockHash:  blockHash.CloneBytes(),
		FilterType: repository.RegularFilter,
	}

	filter, err := s.filterDB.GetFilter(context.Background(), filterToFetchKey)
	if err != nil {
		if err == repository.ErrFilterNotFound {
			return false, nil
		}
		return false, err
	}

	gcsFilter, err := filter.GcsFilter()
	if err != nil {
		return false, err
	}

	key := builder.DeriveKey(blockHash)
	matched, err := gcsFilter.MatchAny(key, items)
	if err != nil {
		return false, err
	}

	return matched, nil
}

func (s *scannerService) extractBlockMatches(blockHash *chainhash.Hash, requests []*ScanRequest) ([]Report, []*ScanRequest, error) {
	block, err := s.blockService.GetBlock(blockHash)
	if err != nil {
		if err == blockservice.ErrorBlockNotFound {
			return nil, requests, nil // skip requests if block svc is not able to find the block
		}

		return nil, nil, err
	}

	results := make([]Report, 0)

	remainRequests := make([]*ScanRequest, 0)

	for _, req := range requests {
		reqMatchedAtLeastOneTime := false
		for _, tx := range block.TransactionsData.Transactions {
			if req.Item.Match(tx) {
				reqMatchedAtLeastOneTime = true
				results = append(results, Report{
					Transaction: tx,
					BlockHash:   blockHash,
					BlockHeight: block.Header.Height,
				})
			}
		}

		if !reqMatchedAtLeastOneTime {
			remainRequests = append(remainRequests, req)
		}
	}

	return results, remainRequests, nil
}
