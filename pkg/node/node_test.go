package node_test

import (
	"github.com/vulpemventures/neutrino-elements/pkg/node"
	"github.com/vulpemventures/neutrino-elements/pkg/repository/inmemory"
	"testing"
	"time"
)

var (
	peerAddr = "localhost:18886"
)

func TestSendTransaction(t *testing.T) {
	nodeSvc, err := node.New(node.NodeConfig{
		Network:        "nigiri",
		UserAgent:      "neutrino-elements:0.0.1",
		FiltersDB:      inmemory.NewFilterInmemory(),
		BlockHeadersDB: inmemory.NewHeaderInmemory(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := nodeSvc.Start(peerAddr); err != nil {
		t.Fatal(err)
	}

	txHex, txID, err := createTx()
	if err != nil {
		t.Fatal(err)
	}

	err = nodeSvc.SendTransaction(txHex)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 5)
	_, err = getTransactionHex(txID)
	if err != nil {
		t.Fatal(err)
	}
}
