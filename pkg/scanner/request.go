package scanner

type ScanRequest struct {
	startHeight uint32    // nil means scan from genesis block
	item        WatchItem // item to watch
}

type ScanRequestOption func(req *ScanRequest)

func NewScanRequest(options ...ScanRequestOption) *ScanRequest {
	req := &ScanRequest{
		item:        nil,
		startHeight: 0,
	}
	for _, option := range options {
		option(req)
	}
	return req
}

func WithWatchItem(item WatchItem) ScanRequestOption {
	return func(req *ScanRequest) {
		req.item = item
	}
}

func WithStartBlock(blockHeight uint32) ScanRequestOption {
	return func(req *ScanRequest) {
		req.startHeight = blockHeight
	}
}
