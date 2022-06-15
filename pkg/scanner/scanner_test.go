package scanner_test

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/stretchr/testify/assert"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
	"github.com/vulpemventures/neutrino-elements/pkg/testutil"
	"testing"
	"time"
)

func TestWatch(t *testing.T) {
	const address = "el1qq0mjw2fwsc20vr4q2ypq9w7dslg6436zaahl083qehyghv7td3wnaawhrpxphtjlh4xjwm6mu29tp9uczkl8cxfyatqc3vgms"

	n, s, reportCh := testutil.MakeNigiriTestServices(
		testutil.PeerAddrLocal,
		testutil.EsploraUrlLocal,
		"nigiri",
	)

	watchItem, err := scanner.NewScriptWatchItemFromAddress(address)
	if err != nil {
		t.Fatal(err)
	}

	tip, err := n.GetChainTip()
	if err != nil {
		t.Fatal(err)
	}

	s.Watch(scanner.WithStartBlock(tip.Height+1), scanner.WithWatchItem(watchItem))
	txid, err := testutil.Faucet(address)
	if err != nil {
		t.Fatal(err)
	}

	nextReport := <-reportCh

	if nextReport.Transaction.TxHash().String() != txid {
		t.Fatalf("expected txid %s, got %s", txid, nextReport.Transaction.TxHash().String())
	}

	s.Stop()
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 3)
}

func TestWatchPersistent(t *testing.T) {
	const address = "el1qqfs4ecf5427tyshnsq0x3jy3ad2tqfn03x3fqmxtyn2ycuvmk98urxmh9cdmr5zcqfs42l6a3kpyrk6pkxjx7yuvqsnuuckhp"

	n, s, reportCh := testutil.MakeNigiriTestServices(
		testutil.PeerAddrLocal,
		testutil.EsploraUrlLocal,
		"nigiri",
	)

	watchItem, err := scanner.NewScriptWatchItemFromAddress(address)
	if err != nil {
		t.Fatal(err)
	}

	tip, err := n.GetChainTip()
	if err != nil {
		t.Fatal(err)
	}

	s.Watch(scanner.WithStartBlock(tip.Height+1), scanner.WithWatchItem(watchItem), scanner.WithPersistentWatch())
	txid, err := testutil.Faucet(address)
	if err != nil {
		t.Fatal(err)
	}

	nextReport := <-reportCh

	if nextReport.Transaction.TxHash().String() != txid {
		t.Fatalf("expected txid %s, got %s", txid, nextReport.Transaction.TxHash().String())
	}

	// we test if the watch is persistent by sending a new transaction
	txid, err = testutil.Faucet(address)
	if err != nil {
		t.Fatal(err)
	}

	nextReport = <-reportCh

	if nextReport.Transaction.TxHash().String() != txid {
		t.Fatalf("expected txid %s, got %s", txid, nextReport.Transaction.TxHash().String())
	}

	s.Stop()
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 3)
}

func TestWalletDescriptor(t *testing.T) {
	n, s, reportCh := testutil.MakeNigiriTestServices(
		testutil.PeerAddrLocal,
		testutil.EsploraUrlLocal,
		"nigiri",
	)

	privkey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Fatal(err)
	}
	pubkey := privkey.PubKey()
	p2wpkh := payment.FromPublicKey(pubkey, &network.Regtest, nil)
	addr, _ := p2wpkh.WitnessPubKeyHash()

	wpkhWalletDescriptor := fmt.Sprintf("wpkh(%v)", hex.EncodeToString(pubkey.SerializeCompressed()))

	tip, err := n.GetChainTip()
	if err != nil {
		t.Fatal(err)
	}

	if err := s.WatchDescriptorWallet(
		wpkhWalletDescriptor,
		[]scanner.EventType{scanner.UnspentUtxo},
		int(tip.Height),
	); err != nil {
		t.Fatal(err)
	}

	txID, err := testutil.Faucet(addr)
	if err != nil {
		t.Fatal(err)
	}

	nextReport := <-reportCh

	if nextReport.Transaction.TxHash().String() != txID {
		t.Fatalf("expected txid %s, got %s", txID, nextReport.Transaction.TxHash().String())
	}

	s.Stop()
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second * 3)
}

func TestWalletDescriptorRange(t *testing.T) {
	n, s, reportCh := testutil.MakeNigiriTestServices(
		testutil.PeerAddrLocal,
		testutil.EsploraUrlLocal,
		"nigiri",
	)

	masterPrivateKey, err := testutil.GenerateMasterPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	childKey, err := masterPrivateKey.Derive(1)
	if err != nil {
		t.Fatal(err)
	}

	addresses := make([]string, 0)
	for i := 0; i < 10; i++ {
		child, err := childKey.Derive(uint32(i))
		if err != nil {
			t.Fatal(err)
		}

		pubKey, err := child.ECPubKey()
		if err != nil {
			if err != nil {
				t.Fatal(err)
			}
		}

		pk, err := btcec.NewPrivateKey(btcec.S256())
		if err != nil {
			t.Fatal(err)
		}

		p2wpkh := payment.FromPublicKey(pubKey, &network.Regtest, pk.PubKey())
		addr, err := p2wpkh.ConfidentialWitnessPubKeyHash()
		if err != nil {
			t.Fatal(err)
		}
		addresses = append(addresses, addr)
	}

	masterPubKey, err := masterPrivateKey.Neuter()
	if err != nil {
		t.Fatal(err)
	}
	wpkhWalletDescriptor := fmt.Sprintf("wpkh(%v/1/*)", masterPubKey.String())

	tip, err := n.GetChainTip()
	if err != nil {
		t.Fatal(err)
	}

	if err := s.WatchDescriptorWallet(
		wpkhWalletDescriptor,
		[]scanner.EventType{scanner.UnspentUtxo},
		int(tip.Height),
	); err != nil {
		t.Fatal(err)
	}

	go func() {
		for _, v := range addresses {
			time.Sleep(2 * time.Second)

			if _, err := testutil.Faucet(v); err != nil {
				fmt.Println(err)
			}
		}
	}()

	i := 0
loop:
	for {
		select {
		case r := <-reportCh:
			i++
			if i == 10 {
				break loop
			}
			t.Log(r.Transaction.TxHash().String())
		case <-time.After(time.Minute):
			break loop
		}
	}

	assert.Equal(t, 10, i)

	s.Stop()
	if err := n.Stop(); err != nil {
		t.Fatal(err)
	}
}

func TestWalletDescriptorTestNet(t *testing.T) {
	t.SkipNow()
	descInternal := "wpkh(xpub6CLsieBwg2jBNBbfoF7UqA6FnU6RjQLT2BXYRTxwq9BfTsSuMiEemky8jVnoECZSrqiJmyUCZUTg9SXJxFYZzzo66KVqL1Z4fYTb9rF6u3F/0/*)"
	descExternal := "wpkh(xpub6CLsieBwg2jBNBbfoF7UqA6FnU6RjQLT2BXYRTxwq9BfTsSuMiEemky8jVnoECZSrqiJmyUCZUTg9SXJxFYZzzo66KVqL1Z4fYTb9rF6u3F/1/*)"

	n, s, reportCh := testutil.MakeNigiriTestServices(
		"liquid-testnet.sevenlabs.dev:18886",
		"http://blockstream.info/liquidtestnet/api",
		"testnet",
	)

	time.Sleep(time.Minute * 1)

	tip, err := n.GetChainTip()
	if err != nil {
		t.Fatal(err)
	}

	if err := s.WatchDescriptorWallet(
		descInternal,
		[]scanner.EventType{scanner.UnspentUtxo},
		int(tip.Height),
	); err != nil {
		t.Fatal(err)
	}

	if err := s.WatchDescriptorWallet(
		descExternal,
		[]scanner.EventType{scanner.UnspentUtxo},
		int(tip.Height),
	); err != nil {
		t.Fatal(err)
	}

loop:
	for {
		select {
		case r := <-reportCh:
			t.Log(r.Transaction.TxHash().String())
		case <-time.After(time.Minute * 15):
			break loop
		}
	}
}
