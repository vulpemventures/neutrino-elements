package scanner

type ScanRequest struct {
	StartHeight uint32    // nil means scan from genesis block
	Item        WatchItem // item to watch
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

func newScanRequest(options ...ScanRequestOption) *ScanRequest {
	req := &ScanRequest{
		Item:        nil,
		StartHeight: 0,
	}
	for _, option := range options {
		option(req)
	}
	return req
}
