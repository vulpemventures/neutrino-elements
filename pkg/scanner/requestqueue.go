package scanner

import (
	"sync"
)

type scanRequestQueue struct {
	requests []*ScanRequest
	locker   sync.Locker
	cond     sync.Cond
}

func newScanRequestQueue() *scanRequestQueue {
	return &scanRequestQueue{
		requests: make([]*ScanRequest, 0),
		locker:   new(sync.Mutex),
		cond:     sync.Cond{L: &sync.Mutex{}},
	}
}

func (queue *scanRequestQueue) dequeueAtHeight(height uint32) []*ScanRequest {
	queue.locker.Lock()
	defer queue.locker.Unlock()

	var selected []*ScanRequest // requests starting at hash
	var remain []*ScanRequest   // remaining requests

	for _, req := range queue.requests {
		if req.StartHeight == height {
			selected = append(selected, req)
		} else {
			remain = append(remain, req)
		}
	}
	queue.requests = remain
	return selected
}

func (queue *scanRequestQueue) enqueue(req *ScanRequest) {
	queue.locker.Lock()
	defer queue.locker.Unlock()
	defer queue.cond.Signal()

	queue.requests = append(queue.requests, req)
}

func (queue *scanRequestQueue) peek() *ScanRequest {
	queue.locker.Lock()
	defer queue.locker.Unlock()

	if len(queue.requests) == 0 {
		return nil
	}

	return queue.requests[0]
}

func (queue *scanRequestQueue) isEmpty() bool {
	queue.locker.Lock()
	defer queue.locker.Unlock()

	return len(queue.requests) == 0
}
