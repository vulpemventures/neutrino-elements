package e2etest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/gorilla/websocket"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	neutrinodtypes "github.com/vulpemventures/neutrino-elements/pkg/neutrinod-types"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
	"github.com/vulpemventures/neutrino-elements/pkg/testutil"
	"net/url"
	"sync"
	"testing"
	"time"
)

// TestEnd2End tests the neutrino daemon server which listens for new
//websocket connections where it is expected to receive a request to scan
//wallet descriptor and return new UTXO's for the wallet.
//In bellow test case two transactions are created for 5 descriptor wallet's and
//it is expected that neutrino daemon will return 10 new UTXO's events 2 for each wallet
func (e *E2ESuite) TestEnd2End() {
	wsRequests := createTxs(e.T())

	wg := sync.WaitGroup{}
	i := 0
	for k, v := range wsRequests {
		wg.Add(1)
		go func(a string, b neutrinodtypes.WsMessageRequest, i int) {
			invokeNeutrinoD(i, e.T(), &wg, a, b)
		}(k, v, i)
		i++
	}
	wg.Wait()
}

func createTxs(t *testing.T) map[string]neutrinodtypes.WsMessageRequest {
	resp := make(map[string]neutrinodtypes.WsMessageRequest)
	for i := 0; i < 5; i++ {
		privkey, err := btcec.NewPrivateKey(btcec.S256())
		if err != nil {
			t.Fatal(err)
		}
		pubkey := privkey.PubKey()
		p2wpkh := payment.FromPublicKey(pubkey, &network.Regtest, nil)
		addr, _ := p2wpkh.WitnessPubKeyHash()
		wpkhWalletDescriptor := fmt.Sprintf("wpkh(%v)", hex.EncodeToString(pubkey.SerializeCompressed()))

		req := neutrinodtypes.WsMessageRequest{
			ActionType:       "register",
			EventTypes:       []scanner.EventType{scanner.UnspentUtxo},
			DescriptorWallet: wpkhWalletDescriptor,
			StartBlockHeight: 0,
		}
		resp[addr] = req

		_, err = testutil.Faucet(addr)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second * 2)

		_, err = testutil.Faucet(addr)
		if err != nil {
			t.Fatal(err)
		}
	}

	return resp
}

func invokeNeutrinoD(
	id int,
	t *testing.T,
	wg *sync.WaitGroup,
	addr string,
	req neutrinodtypes.WsMessageRequest,
) {
	defer wg.Done()

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/neutrino"}
	t.Logf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.WriteMessage(websocket.TextMessage, reqBytes); err != nil {
		t.Log(err)
	}

	receivedTxEventMsg := 0
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				return
			}

			msg := neutrinodtypes.WsOnChainEventResponse{}
			if err := json.Unmarshal(message, &msg); err != nil {
				t.Error(err)
			}

			if msg.TxID != "" {
				t.Logf("id: %v, recv: %v", id, msg.TxID)

				receivedTxEventMsg++
			}

		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	now := time.Now()
	for {
		t.Logf("id: %v, count: %v", id, receivedTxEventMsg)
		if time.Since(now) > time.Second*45 {
			t.Fatal("test timeout")
		}

		if receivedTxEventMsg == 2 {
			time.Sleep(time.Second * 2)

			if err := c.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			); err != nil {
				t.Fatal(err)
			}

			break
		}
		time.Sleep(time.Second)
	}
}
