package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/gorilla/websocket"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/neutrino-elements/pkg/testutil"
	"log"
	"net/url"
	"testing"
	"time"
)

// TestNeutrinoDaemon tests the neutrino daemon which listens for new
//websocket connections where it is expected to receive a request to scan
//wallet descriptor and return new UTXO's for the wallet.
//In bellow test case two transactions are created for wallet descriptor and
//it is expected that neutrino daemon will return two new UTXO's
func TestNeutrinoDaemon(t *testing.T) {
	neutrinod, err := testutil.RunCommandDetached("../../bin/neutrinod")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := neutrinod.Process.Kill(); err != nil {
			t.Fatal(err)
		}
	}()

	time.Sleep(time.Second * 3)

	privkey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Fatal(err)
	}
	pubkey := privkey.PubKey()
	p2wpkh := payment.FromPublicKey(pubkey, &network.Regtest, nil)
	addr, _ := p2wpkh.WitnessPubKeyHash()
	wpkhWalletDescriptor := fmt.Sprintf("wpkh(%v)", hex.EncodeToString(pubkey.SerializeCompressed()))

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/neutrino"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	req := WsMessageReq{
		EventType:        "UNSPENT",
		DescriptorWallet: wpkhWalletDescriptor,
		StartBlockHeight: 0,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.WriteMessage(websocket.TextMessage, reqBytes); err != nil {
		t.Log(err)
	}

	receivedMsgs := 0
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				return
			}
			log.Printf("recv: %s", message)

			receivedMsgs++
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	go func() {
		_, err := testutil.Faucet(addr)
		if err != nil {
			fmt.Println(err)
		}

		time.Sleep(time.Second * 2)

		_, err = testutil.Faucet(addr)
		if err != nil {
			fmt.Println(err)
		}
	}()

	now := time.Now()
	for {
		if time.Since(now) > time.Second*45 {
			t.Fatal("test timeout")
		}

		if receivedMsgs == 2 {
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
