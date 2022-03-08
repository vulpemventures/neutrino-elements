package node_test

import (
	"log"
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tdex-network/tdex-daemon/pkg/bufferutil"
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
)

func TestSendTransaction(t *testing.T) {
	nodeSvc, err := newTestNodeSvc()
	if err != nil {
		t.Fatal(err)
	}

	nodeSvc.Start("localhost:18886")

	esploraSvc, err := newExplorerSvc()
	if err != nil {
		t.Fatal(err)
	}

	addr, privKey, err := newTestData()
	if err != nil {
		t.Fatal(err)
	}

	txid, err := esploraSvc.Faucet(addr, float64(1), "")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 5)
	txhex, err := esploraSvc.GetTransactionHex(txid)
	if err != nil {
		t.Fatal(err)
	}

	faucetTx, err := transaction.NewTxFromHex(txhex)
	if err != nil {
		t.Fatal(err)
	}

	ptx, err := pset.New(nil, nil, 2, 0)
	if err != nil {
		t.Fatal(err)
	}

	updater, err := pset.NewUpdater(ptx)
	if err != nil {
		t.Fatal(err)
	}

	h, _ := chainhash.NewHashFromStr(txid)

	updater.AddInput(&transaction.TxInput{
		Hash:  h.CloneBytes(),
		Index: 0,
	})

	err = updater.AddInNonWitnessUtxo(faucetTx, 0)
	if err != nil {
		t.Fatal(err)
	}

	script, _ := address.ToOutputScript(addr)

	lbtc, _ := bufferutil.AssetHashToBytes(network.Regtest.AssetID)
	sats, _ := elementsutil.SatoshiToElementsValue(100000000 - 10000)
	fees, _ := elementsutil.SatoshiToElementsValue(10000)

	updater.AddOutput(&transaction.TxOutput{
		Asset:  lbtc,
		Value:  sats,
		Script: script,
		Nonce:  []byte{0x00},
	})

	updater.AddOutput(&transaction.TxOutput{
		Asset:  lbtc,
		Value:  fees,
		Script: []byte{},
		Nonce:  []byte{0x00},
	})

	err = signInput(ptx, 0, privKey, faucetTx.Outputs[0].Script, faucetTx.Outputs[0].Value)
	if err != nil {
		t.Fatal(err)
	}

	base64, err := ptx.ToBase64()
	if err != nil {
		t.Fatal(err)
	}

	hex, broadcastedtxid, err := finalizeAndExtractTransaction(base64)
	if err != nil {
		t.Fatal(err)
	}

	err = nodeSvc.SendTransaction(hex)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 5)
	log.Println(broadcastedtxid)
	_, err = esploraSvc.GetTransactionHex(broadcastedtxid)
	if err != nil {
		t.Fatal(err)
	}
}
