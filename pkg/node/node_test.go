package node_test

import (
	"github.com/vulpemventures/neutrino-elements/pkg/node"
	"github.com/vulpemventures/neutrino-elements/pkg/repository/inmemory"
	"testing"
	"time"
)

func TestSendTransaction(t *testing.T) {
	nodeSvc, err := node.New(node.NodeConfig{
		Network:        "testnet",
		UserAgent:      "neutrino-elements:0.0.1",
		FiltersDB:      inmemory.NewFilterInmemory(),
		BlockHeadersDB: inmemory.NewHeaderInmemory(),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := nodeSvc.Start("liquid-testnet.sevenlabs.dev:18886"); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Minute * 15)

	//txHex, txID, err := testutil.CreateTx()
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//err = nodeSvc.SendTransaction(txHex)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//time.Sleep(time.Second * 5)
	//_, err = testutil.GetTransactionHex(txID)
	//if err != nil {
	//	t.Fatal(err)
	//}
}
