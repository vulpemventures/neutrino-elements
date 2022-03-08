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
		addr := c.String("address")
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
		watchItem, err := scanner.NewScriptWatchItemFromAddress(addr)
		if err != nil {
			panic(err)
		}

		time.Sleep(time.Second * 3)
		scanSvc.Watch(
			scanner.WithStartBlock(0),
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
