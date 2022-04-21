package scanner

type ScanRequest struct {
	StartHeight  uint32    // nil means scan from genesis block
	Item         WatchItem // item to watch
	IsPersistent bool      // if true, the request will be re-added with StartHeight = StartHeiht + 1
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

func newScanRequest(options ...ScanRequestOption) *ScanRequest {
	req := &ScanRequest{}
	for _, option := range options {
		option(req)
	}
	return req
}
