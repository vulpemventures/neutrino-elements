package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
	"github.com/vulpemventures/neutrino-elements/pkg/repository/inmemory"
)

type State struct {
	filtersDB      repository.FilterRepository
	blockHeadersDB repository.BlockHeaderRepository
}

func main() {
	state := &State{
		filtersDB:      inmemory.NewFilterInmemory(),
		blockHeadersDB: inmemory.NewHeaderInmemory(),
	}

	app := cli.NewApp()
	app.Name = "neutrino-elements"
	app.Version = "0.0.1"
	app.Usage = "elements node + utxos scanner"
	app.Commands = []*cli.Command{
		{
			Name:   "start",
			Usage:  "run neutrino-elements node",
			Action: startAction(state),
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
