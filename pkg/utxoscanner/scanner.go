package utxoscanner

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/neutrino-elements/internal/domain"
)

type SpendReport struct {
	// SpendingTx is the transaction that spent the output that a spend
	// report was requested for.
	//
	// NOTE: This field will only be populated if the target output has
	// been spent.
	SpendingTx *transaction.Transaction

	// SpendingTxIndex is the input index of the transaction above which
	// spends the target output.
	//
	// NOTE: This field will only be populated if the target output has
	// been spent.
	SpendingInputIndex uint32

	// SpendingTxHeight is the hight of the block that included the
	// transaction  above which spent the target output.
	//
	// NOTE: This field will only be populated if the target output has
	// been spent.
	SpendingTxHeight uint32

	// Output is the raw output of the target outpoint.
	//
	// NOTE: This field will only be populated if the target is still
	// unspent.
	Output *transaction.TxOutput

	// BlockHash is the block hash of the block that includes the unspent
	// output.
	//
	// NOTE: This field will only be populated if the target is still
	// unspent.
	BlockHash *chainhash.Hash

	// BlockHeight is the height of the block that includes the unspent output.
	//
	// NOTE: This field will only be populated if the target is still
	// unspent.
	BlockHeight uint32

	// BlockIndex is the index of the output's transaction in its block.
	//
	// NOTE: This field will only be populated if the target is still
	// unspent.
	BlockIndex uint32
}

type UtxoScanner interface {
	// Start runs a go-routine in order to handle incoming requests via Watch
	Start() (<-chan SpendReport, error)
	// Stop the scanner
	Stop() error
	Watch(...ScanRequestOption) error
}

type utxoScanner struct {
	filterRepository domain.FilterRepository
	headerRepository domain.BlockHeaderRepository
}

// var _ UtxoScanner = (*utxoScanner)(nil)
