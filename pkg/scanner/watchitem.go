package scanner

import (
	"bytes"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/transaction"
)

// WatchItem is an interface containing the common methods using by Scanner to watch specific item.
type WatchItem interface {
	// Bytes returns the element search in the block filter
	Bytes() []byte
	// Match is used to check if a transaction matches the watch item
	Match(tx *transaction.Transaction) bool
	// EventType returns the type of event that will be reported by the scanner
	EventType() EventType
}

// SpentWatchItem is used to watch for spent utxos.
type SpentWatchItem struct {
	hash         *chainhash.Hash // tx hash of the outpoint
	index        uint32          // index of the outpoint
	outputScript []byte          // the way the outpoint should be spent
}

func NewSpentWatchItemFromInput(input *transaction.TxInput, prevoutScript []byte) (WatchItem, error) {
	h, err := chainhash.NewHash(input.Hash)
	if err != nil {
		return nil, err
	}

	return &SpentWatchItem{
		hash:         h,
		index:        input.Index,
		outputScript: prevoutScript,
	}, nil
}

func (o *SpentWatchItem) Bytes() []byte {
	return o.outputScript
}

func (o *SpentWatchItem) Match(tx *transaction.Transaction) bool {
	for _, txInput := range tx.Inputs {
		chainHashTxInput, err := chainhash.NewHash(txInput.Hash)
		if err != nil {
			continue
		}

		if o.hash.IsEqual(chainHashTxInput) && o.index == txInput.Index {
			return true
		}
	}
	return false
}

func (o *SpentWatchItem) EventType() EventType {
	return SpentUtxo
}

// UnspentWatchItem is used to recognise new unspent output related to a specific address/script
type UnspentWatchItem struct {
	outputScript []byte
}

func NewUnspentWatchItemFromAddress(addr string) (WatchItem, error) {
	script, err := address.ToOutputScript(addr)
	if err != nil {
		return nil, err
	}

	return &UnspentWatchItem{
		outputScript: script,
	}, nil
}

func (u *UnspentWatchItem) Bytes() []byte {
	return u.outputScript
}

func (u *UnspentWatchItem) Match(tx *transaction.Transaction) bool {
	for _, txOutput := range tx.Outputs {
		if bytes.Equal(u.outputScript, txOutput.Script) {
			return true
		}
	}
	return false
}

func (u *UnspentWatchItem) EventType() EventType {
	return UnspentUtxo
}
