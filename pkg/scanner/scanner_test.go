package scanner_test

import (
	"testing"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tdex-network/tdex-daemon/pkg/explorer/esplora"
	"github.com/vulpemventures/neutrino-elements/pkg/blockservice"
	"github.com/vulpemventures/neutrino-elements/pkg/node"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository/inmemory"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
)

var repoFilter = inmemory.NewFilterInmemory()
var repoHeader = inmemory.NewHeaderInmemory()

func makeNigiriTestServices() (node.NodeService, scanner.ScannerService, <-chan scanner.Report) {
	n, err := node.New(node.NodeConfig{
		Network:        "nigiri",
		UserAgent:      "neutrino-elements:test",
		FiltersDB:      repoFilter,
		BlockHeadersDB: repoHeader,
	})

	if err != nil {
		panic(err)
	}

	err = n.Start("localhost:18886")
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 3) // wait for the node sync the first headers if the repo is empty

	blockSvc := blockservice.NewEsploraBlockService("http://localhost:3001")
	genesisBlockHash := protocol.GetCheckpoints(protocol.MagicNigiri)[0]
	h, err := chainhash.NewHashFromStr(genesisBlockHash)
	if err != nil {
		panic(err)
	}
	s := scanner.New(repoFilter, repoHeader, blockSvc, h)

	reportCh, err := s.Start()
	if err != nil {
		panic(err)
	}

	return n, s, reportCh
}

func faucet(addr string) (string, error) {
	svc, err := esplora.NewService("http://127.0.0.1:3001", 5000)
	if err != nil {
		return "", err
	}

	return svc.Faucet(addr, 1, "5ac9f65c0efcc4775e0baec4ec03abdde22473cd3cf33c0419ca290e0751b225")
}

func TestWatch(t *testing.T) {
	const address = "el1qq0mjw2fwsc20vr4q2ypq9w7dslg6436zaahl083qehyghv7td3wnaawhrpxphtjlh4xjwm6mu29tp9uczkl8cxfyatqc3vgms"

	n, s, reportCh := makeNigiriTestServices()

	watchItem, err := scanner.NewScriptWatchItemFromAddress(address)
	if err != nil {
		t.Fatal(err)
	}

	tip, err := n.GetChainTip()
	if err != nil {
		t.Fatal(err)
	}

	s.Watch(scanner.WithStartBlock(tip.Height+1), scanner.WithWatchItem(watchItem))
	txid, err := faucet(address)
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

	n, s, reportCh := makeNigiriTestServices()

	watchItem, err := scanner.NewScriptWatchItemFromAddress(address)
	if err != nil {
		t.Fatal(err)
	}

	tip, err := n.GetChainTip()
	if err != nil {
		t.Fatal(err)
	}

	s.Watch(scanner.WithStartBlock(tip.Height+1), scanner.WithWatchItem(watchItem), scanner.WithPersistentWatch())
	txid, err := faucet(address)
	if err != nil {
		t.Fatal(err)
	}

	nextReport := <-reportCh

	if nextReport.Transaction.TxHash().String() != txid {
		t.Fatalf("expected txid %s, got %s", txid, nextReport.Transaction.TxHash().String())
	}

	// we test if the watch is persistent by sending a new transaction
	txid, err = faucet(address)
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
