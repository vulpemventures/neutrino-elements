package node_test

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/neutrino-elements/pkg/node"
	"github.com/vulpemventures/neutrino-elements/pkg/repository/inmemory"
)

func newTestNodeSvc() (node.NodeService, error) {
	return node.New(node.NodeConfig{
		Network:        "nigiri",
		UserAgent:      "neutrino-elements:0.0.1",
		FiltersDB:      inmemory.NewFilterInmemory(),
		BlockHeadersDB: inmemory.NewHeaderInmemory(),
	})
}

func newExplorerSvc() (explorer.Service, error) {
	return esplora.NewService("http://127.0.0.1:3001", 5000)
}

func newTestData() (string, *btcec.PrivateKey, error) {
	key, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return "", nil, err
	}

	p2wpkh := payment.FromPublicKey(
		key.PubKey(),
		&network.Regtest,
		nil,
	)
	addr, err := p2wpkh.WitnessPubKeyHash()
	if err != nil {
		return "", nil, err
	}

	return addr, key, nil
}

func signInput(ptx *pset.Pset, inIndex int, prvkey *btcec.PrivateKey, scriptPubKey []byte, prevoutValue []byte) error {
	updater, err := pset.NewUpdater(ptx)
	if err != nil {
		return err
	}

	pay, err := payment.FromScript(scriptPubKey, nil, nil)
	if err != nil {
		return err
	}

	script := pay.Script
	hashForSignature := ptx.UnsignedTx.HashForWitnessV0(
		inIndex,
		script,
		prevoutValue,
		txscript.SigHashAll,
	)

	signature, err := prvkey.Sign(hashForSignature[:])
	if err != nil {
		return err
	}

	if !signature.Verify(hashForSignature[:], prvkey.PubKey()) {
		return fmt.Errorf(
			"signature verification failed for input %d",
			inIndex,
		)
	}

	sigWithSigHashType := append(signature.Serialize(), byte(txscript.SigHashAll))
	_, err = updater.Sign(
		inIndex,
		sigWithSigHashType,
		prvkey.PubKey().SerializeCompressed(),
		nil,
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

// FinalizeAndExtractTransaction attempts to finalize the provided partial
// transaction and eventually extracts the final transaction and returns
// it in hex string format, along with its transaction id
func finalizeAndExtractTransaction(psetBase64 string) (string, string, error) {
	ptx, err := pset.NewPsetFromBase64(psetBase64)
	if err != nil {
		return "", "", fmt.Errorf("invalid pset: %s", err)
	}

	ok, err := ptx.ValidateAllSignatures()
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", fmt.Errorf("invalid pset: failed to verify all signatures")
	}

	if err := pset.FinalizeAll(ptx); err != nil {
		return "", "", err
	}

	tx, err := pset.Extract(ptx)
	if err != nil {
		return "", "", err
	}
	txHex, err := tx.ToHex()
	if err != nil {
		return "", "", err
	}
	return txHex, tx.TxHash().String(), nil
}
