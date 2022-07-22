package main

import (
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/btcec/v2"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/neutrino-elements/pkg/testutil"
	"time"
)

func main() {
	privkey, err := btcec.NewPrivateKey()
	if err != nil {
		log.Fatal(err)
	}
	pubkey := privkey.PubKey()
	p2wpkh := payment.FromPublicKey(pubkey, &network.Regtest, nil)
	addr, _ := p2wpkh.WitnessPubKeyHash()
	wpkhWalletDescriptor := fmt.Sprintf("wpkh(%v)", hex.EncodeToString(pubkey.SerializeCompressed()))

	txID1, err := testutil.Faucet(addr)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(time.Second * 2)

	txID2, err := testutil.Faucet(addr)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("addr: %v\n", addr)
	fmt.Printf("wpkh_desc: %v\n", wpkhWalletDescriptor)
	fmt.Printf("tx_id1: %v\n", txID1)
	fmt.Printf("tx_id2: %v\n", txID2)
}
