package scanner

import (
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
	blockService  blockservice.BlockService
	quitCh        chan struct{}
}

var _ ScannerService = (*scannerService)(nil)

func NewUtxoScanner(
	filterDB repository.FilterRepository,
	headerDB repository.BlockHeaderRepository,
	blockSvc blockservice.BlockService,
) ScannerService {
	return &scannerService{
		requestsQueue: newScanRequestQueue(),
		filterDB:      filterDB,
		headerDB:      headerDB,
		blockService:  blockSvc,
		quitCh:        make(chan struct{}),
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
	req := NewScanRequest(opts...)
	s.requestsQueue.enqueue(req)
	return nil
}

// requestsManager is responsible to resolve the requests that are waiting for in the queue.
func (s *scannerService) requestsManager(ch chan<- Report) {
	defer close(s.quitCh)

	for {
		for s.requestsQueue.isEmpty() {
			s.requestsQueue.cond.Wait() // wait for new requests

			// check if we should quit the routine
			select {
			case <-s.quitCh:
				return
			default:
			}
		}

		// get the next request without removing it from the queue
		nextRequest := s.requestsQueue.peek()
		err := s.requestWorker(nextRequest.startHeight, ch)
		if err != nil {
			logrus.Error(err)
		}

		// check if we should quit the routine
		select {
		case <-s.quitCh:
			return
		default:
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
			itemsBytes[i] = req.item.Bytes()
		}

		// get the block for height
		header, err := s.headerDB.GetBlockHeaderByHeight(startHeight)
		if err != nil {
			return err
		}

		// compute the block hash
		blockHash, err := header.Hash()
		if err != nil {
			return err
		}

		// check with filterDB if the block has one of the items
		matched, err := s.blockFilterMatches(itemsBytes, &blockHash)
		if err != nil {
			return err
		}

		if matched {
			reports, remainReqs, err := s.extractBlockMatches(&blockHash, nextBatch)
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
	}

	return nil
}

func (s *scannerService) blockFilterMatches(items [][]byte, blockHash *chainhash.Hash) (bool, error) {
	filter, err := s.filterDB.FetchFilter(blockHash, repository.RegularFilter)
	if err != nil {
		return false, err
	}

	key := builder.DeriveKey(blockHash)
	matched, err := filter.MatchAny(key, items)
	if err != nil {
		return false, err
	}

	return matched, nil
}

func (s *scannerService) extractBlockMatches(blockHash *chainhash.Hash, requests []*ScanRequest) ([]Report, []*ScanRequest, error) {
	block, err := s.blockService.GetBlock(blockHash)
	if err != nil {
		return nil, nil, err
	}

	results := make([]Report, 0)

	remainRequests := make([]*ScanRequest, 0)

	for _, req := range requests {
		reqMatchedAtLeastOneTime := false
		for _, tx := range block.TransactionsData.Transactions {
			if req.item.Match(tx) {
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
