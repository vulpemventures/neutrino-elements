package scanner_test

import (
	"bytes"
	"testing"

	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
)

type fakeWatchItem struct {
	bytes []byte
}

var _ scanner.WatchItem = (*fakeWatchItem)(nil)

func (f *fakeWatchItem) Bytes() []byte {
	return f.bytes
}

func (f *fakeWatchItem) Match(*transaction.Transaction) bool {
	return true
}

func newFakeWatchItem(bytes []byte) scanner.WatchItem {
	return &fakeWatchItem{bytes: bytes}
}

var initialWatchItem = newFakeWatchItem([]byte{0x01})
var initialReq = scanner.ScanRequest{
	StartHeight: 0,
	Item:        initialWatchItem,
}

func TestRequestOptions(t *testing.T) {
	tests := []struct {
		name     string
		option   scanner.ScanRequestOption
		expected scanner.ScanRequest
	}{
		{
			name:   "WithWatchItem",
			option: scanner.WithWatchItem(newFakeWatchItem([]byte("fake"))),
			expected: scanner.ScanRequest{
				Item: &fakeWatchItem{
					bytes: []byte("fake"),
				},
				StartHeight: 0,
			},
		},
		{
			name:   "WithStartBlock",
			option: scanner.WithStartBlock(666),
			expected: scanner.ScanRequest{
				Item:        initialWatchItem,
				StartHeight: 666,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			req := initialReq
			test.option(&req)

			if !bytes.Equal(req.Item.Bytes(), test.expected.Item.Bytes()) {
				tt.Errorf("expected: %+v, got: %+v", test.expected.Item, req.Item)
			}

			if req.StartHeight != test.expected.StartHeight {
				tt.Errorf("expected: %d, got: %d", test.expected.StartHeight, req.StartHeight)
			}
		})
	}
}
