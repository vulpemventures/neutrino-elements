package scanner

import (
	"bytes"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/transaction"
)

type WatchItem interface {
	// Bytes returns the element search in the block filter
	Bytes() []byte
	// Match is used to check if a transaction matches the watch item
	Match(tx *transaction.Transaction) bool
}

// OutpointWatchItem can be used to check if an outpoint is spent by transaction inputs
type OutpointWatchItem struct {
	hash         *chainhash.Hash // tx hash of the outpoint
	index        uint32          // index of the outpoint
	outputScript []byte          // the way the outpoint should be spent
}

func NewOutpointWatchItemFromInput(input *transaction.TxInput, prevoutScript []byte) (*OutpointWatchItem, error) {
	h, err := chainhash.NewHash(input.Hash)
	if err != nil {
		return nil, err
	}

	return &OutpointWatchItem{
		hash:         h,
		index:        input.Index,
		outputScript: prevoutScript,
	}, nil
}

func (i *OutpointWatchItem) Bytes() []byte {
	return i.outputScript
}

func (i *OutpointWatchItem) Match(tx *transaction.Transaction) bool {
	for _, txInput := range tx.Inputs {
		chainHashTxInput, err := chainhash.NewHash(txInput.Hash)
		if err != nil {
			continue
		}

		if i.hash.IsEqual(chainHashTxInput) && i.index == txInput.Index {
			return true
		}
	}
	return false
}

// ScriptWatchItem is used to check if a transaction sends funds to a script
type ScriptWatchItem struct {
	outputScript []byte
}

func NewScriptWatchItemFromAddress(addr string) (WatchItem, error) {
	script, err := address.ToOutputScript(addr)
	if err != nil {
		return nil, err
	}

	return &ScriptWatchItem{
		outputScript: script,
	}, nil
}

func (i *ScriptWatchItem) Bytes() []byte {
	return i.outputScript
}

func (i *ScriptWatchItem) Match(tx *transaction.Transaction) bool {
	for _, txOutput := range tx.Outputs {
		if bytes.Equal(i.outputScript, txOutput.Script) {
			return true
		}
	}
	return false
}
