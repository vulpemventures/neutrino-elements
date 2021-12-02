package utxoscanner

import "github.com/btcsuite/btcd/chaincfg/chainhash"

type ScanRequest struct {
	StartBlock *chainhash.Hash // nil means scan from genesis block
	EndBlock   *chainhash.Hash // nil means scan until tip
	WatchItem  []byte          // item to watch
}

type ScanRequestOption func(req *ScanRequest)

func WithAddress(address string) ScanRequestOption {
	return func(req *ScanRequest) {
		// address to script
	}
}

func WithScript(script []byte) ScanRequestOption {
	return func(req *ScanRequest) {
	}
}

func WithStartBlock(blockhash chainhash.Hash) ScanRequestOption {
	return func(req *ScanRequest) {
		req.StartBlock = &blockhash
	}
}

func WithEndBlock(blockhash chainhash.Hash) ScanRequestOption {
	return func(req *ScanRequest) {
		req.EndBlock = &blockhash
	}
}
