package protocol_test

import (
	"log"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func TestHardcodedGenesisBlockHeader(t *testing.T) {
	tests := []struct {
		name               string
		genesisBlockToTest *block.Header
		expectedHash       string
	}{
		{
			name:               "regtest",
			genesisBlockToTest: &protocol.RegtestGenesisHeader,
			expectedHash:       protocol.RegtestGenesisBlockHash,
		},
		{
			name:               "testnet",
			genesisBlockToTest: &protocol.LiquidTestnetGenesisHeader,
			expectedHash:       protocol.LiquidTestnetGenesisBlockHash,
		},
		{
			name:               "mainnet",
			genesisBlockToTest: &protocol.LiquidGenesisHeader,
			expectedHash:       protocol.LiquidGenesisBlockHash,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hash, err := chainhash.NewHashFromStr(test.expectedHash)
			if err != nil {
				t.Fatal(err)
			}

			headerHash, err := test.genesisBlockToTest.Hash()
			if err != nil {
				t.Fatal(err)
			}

			if !hash.IsEqual(&headerHash) {
				log.Println(test.genesisBlockToTest)
				t.Fatalf("expected hash %s, got %s", test.expectedHash, headerHash)
			}
		})
	}
}
