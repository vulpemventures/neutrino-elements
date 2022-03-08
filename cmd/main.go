package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vulpemventures/neutrino-elements/pkg/node"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
	"github.com/vulpemventures/neutrino-elements/pkg/repository/inmemory"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
)

type State struct {
	filtersDB      repository.FilterRepository
	blockHeadersDB repository.BlockHeaderRepository
	nodeService    node.NodeService
	utxoScanner    scanner.ScannerService
	reportsChan    <-chan scanner.Report
}

func main() {
	// logrus.SetLevel(logrus.DebugLevel)

	state := &State{
		filtersDB:      inmemory.NewFilterInmemory(),
		blockHeadersDB: inmemory.NewHeaderInmemory(),
		nodeService:    nil,
		utxoScanner:    nil,
		reportsChan:    nil,
	}

	app := cli.NewApp()
	app.Name = "neutrino-elements"
	app.Version = "0.0.1"
	app.Usage = "elements node + utxos scanner"
	app.Commands = []*cli.Command{
		{
			Name:  "start",
			Usage: "watch for an address using utxo scanner",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "address",
					Usage:   "address to watch",
					Aliases: []string{"a"},
				},
			},
			Action: startAction(state),
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
