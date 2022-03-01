package main

import (
	"os"
	"os/signal"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vulpemventures/neutrino-elements/pkg/blockservice"
	"github.com/vulpemventures/neutrino-elements/pkg/node"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
)

func startAction(state *State) cli.ActionFunc {
	return func(c *cli.Context) error {
		// Create a new peer node.
		node, err := node.New(node.NodeConfig{
			Network:        "nigiri",
			UserAgent:      "neutrino-elements:0.0.1",
			FiltersDB:      state.filtersDB,
			BlockHeadersDB: state.blockHeadersDB,
		})
		if err != nil {
			panic(err)
		}

		// err = node.Run("liquid-testnet.blockstream.com:18892") // testnet
		err = node.Start("localhost:18886") // regtest
		if err != nil {
			panic(err)
		}

		genesisBlockHash := protocol.GetCheckpoints(protocol.MagicNigiri)[0]
		h, err := chainhash.NewHashFromStr(genesisBlockHash)
		if err != nil {
			panic(err)
		}

		blockSvc := blockservice.NewEsploraBlockService("http://localhost:3001")
		scanSvc := scanner.New(state.filtersDB, state.blockHeadersDB, blockSvc, h)
		reportCh, err := scanSvc.Start()
		if err != nil {
			panic(err)
		}

		go func() {
			for report := range reportCh {
				logrus.Infof("SCAN RESOLVE: %+v", report.Transaction.TxHash())
			}
		}()

		// we'll watch if this address receives fund
		watchItem, err := scanner.NewScriptWatchItemFromAddress("el1qq2enu72g3m306antkz6az3r8qklsjt62p2vt3mlfyaxmc9mwg4cl24hvzq5sfkv45ef9ahnyrr6rnr2vr63tzl5l3jpy950z7")
		if err != nil {
			panic(err)
		}

		// let's send the request to the scanner after 10sec
		time.Sleep(time.Second * 3)
		scanSvc.Watch(
			scanner.WithStartBlock(1),
			scanner.WithWatchItem(watchItem),
		)
		if err != nil {
			panic(err)
		}

		signalQuit := make(chan os.Signal, 1)
		signal.Notify(signalQuit, os.Interrupt)
		<-signalQuit
		node.Stop()
		scanSvc.Stop()
		return nil
	}
}
