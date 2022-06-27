package scanner

type ScanRequest struct {
	// ID of the request
	ID int
	// StartHeight from which scan should be performed, nil means scan from genesis block
	StartHeight uint32
	// Item to watch
	Item WatchItem
	// IsPersistent if true, the request will be re-added with StartHeight = StartHeiht + 1
	IsPersistent bool
}

type ScanRequestOption func(req *ScanRequest)

func WithWatchItem(item WatchItem) ScanRequestOption {
	return func(req *ScanRequest) {
		req.Item = item
	}
}

func WithStartBlock(blockHeight uint32) ScanRequestOption {
	return func(req *ScanRequest) {
		req.StartHeight = blockHeight
	}
}

func WithPersistentWatch() ScanRequestOption {
	return func(req *ScanRequest) {
		req.IsPersistent = true
	}
}

func WithRequestID(id int) ScanRequestOption {
	return func(req *ScanRequest) {
		req.ID = id
	}
}

func newScanRequest(options ...ScanRequestOption) *ScanRequest {
	req := &ScanRequest{}
	for _, option := range options {
		option(req)
	}
	return req
}
