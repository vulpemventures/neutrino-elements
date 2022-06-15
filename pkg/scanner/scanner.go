package scanner

import (
	"context"
	"fmt"
	"github.com/vulpemventures/go-elements/descriptor"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil/gcs/builder"
	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/neutrino-elements/pkg/blockservice"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"

	log "github.com/sirupsen/logrus"
)

const (
	UnspentUtxo EventType = iota

	numOsScripts = 100
)

type EventType int

type Report struct {
	// Transaction is the transaction that includes the item that was found.
	Transaction *transaction.Transaction

	// BlockHash is the block hash of the block that includes the transaction.
	BlockHash   *chainhash.Hash
	BlockHeight uint32

	// the request resolved by the report
	Request *ScanRequest
}

type ScannerService interface {
	// Start runs a go-routine in order to handle incoming requests via Watch
	Start() (<-chan Report, error)
	// Stop the scanner
	Stop()
	// Watch add a new request to the queue
	Watch(...ScanRequestOption)
	// WatchDescriptorWallet imports wallet descriptor, generates scripts and which
	//for specific events for those scripts
	WatchDescriptorWallet(
		descriptor string,
		eventType []EventType,
		blockStart int,
	) error
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
	log.Info("starting scanner")

	if s.started {
		return nil, fmt.Errorf("scanner already started")
	}

	s.quitCh = make(chan struct{}, 1)
	resultCh := make(chan Report)
	// start the requests manager
	go s.requestsManager(resultCh)

	s.started = true
	return resultCh, nil
}

func (s *scannerService) Stop() {
	log.Info("stopping scanner")

	s.quitCh <- struct{}{}
	s.started = false
	s.requestsQueue = newScanRequestQueue()
}

func (s *scannerService) Watch(opts ...ScanRequestOption) {
	req := newScanRequest(opts...)
	s.requestsQueue.enqueue(req)
}

func (s *scannerService) WatchDescriptorWallet(
	desc string,
	eventType []EventType,
	blockStart int,
) error {
	for _, v := range eventType {
		switch v {
		case UnspentUtxo:
			wallet, err := descriptor.Parse(desc)
			if err != nil {
				return err
			}

			var scripts []descriptor.ScriptResponse

			if wallet.IsRange() {
				scripts, err = wallet.Script(descriptor.WithRange(numOsScripts))
				if err != nil {
					return err
				}

				for _, v := range scripts {
					s.Watch(
						WithStartBlock(uint32(blockStart)),
						WithWatchItem(&ScriptWatchItem{
							outputScript: v.Script,
						}),
						WithPersistentWatch(),
					)
				}
			} else {
				scripts, err = wallet.Script(nil)
				if err != nil {
					return err
				}

				s.Watch(
					WithStartBlock(uint32(blockStart)),
					WithWatchItem(&ScriptWatchItem{
						outputScript: scripts[0].Script,
					}),
					WithPersistentWatch(),
				)
			}

		}
	}

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
func (s *scannerService) requestWorker(startHeight uint32, reportsChan chan<- Report) error {
	nextBatch := make([]*ScanRequest, 0)
	nextHeight := startHeight

	chainTip, err := s.headerDB.ChainTip(context.Background())
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
			blockHash, err = s.headerDB.GetBlockHashByHeight(context.Background(), nextHeight)
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

			for _, report := range reports {
				// send the report to the output channel
				reportsChan <- report

				// if the request is persistent, the scanner will keep watching the item at the next block height
				if report.Request.IsPersistent {
					s.Watch(WithStartBlock(report.BlockHeight+1), WithWatchItem(report.Request.Item), WithPersistentWatch())
				}
			}

			// if some requests remain, put them back in the next batch
			// this will remove the resolved requests from the batch
			nextBatch = remainReqs
		}

		// increment the height to scan
		// if nothing was found, we can just continue with same batch and next height
		nextHeight++

		chainTip, err = s.headerDB.ChainTip(context.Background())
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
					Request:     req,
				})
			}
		}

		if !reqMatchedAtLeastOneTime {
			remainRequests = append(remainRequests, req)
		}
	}

	return results, remainRequests, nil
}
