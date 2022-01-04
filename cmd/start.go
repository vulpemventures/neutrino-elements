package main

import (
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"github.com/vulpemventures/neutrino-elements/pkg/node"
)

func startAction(state *State) cli.ActionFunc {
	return func(c *cli.Context) error {
		// Create a new peer node.
		node, err := node.New("nigiri", "test")
		if err != nil {
			panic(err)
		}

		logrus.SetLevel(logrus.DebugLevel)

		// err = node.Run("liquid-testnet.blockstream.com:18892")
		// if err != nil {
		// 	panic(err)
		// }

		err = node.Start("localhost:18886")
		if err != nil {
			panic(err)
		}

		signalQuit := make(chan os.Signal, 1)
		signal.Notify(signalQuit, os.Interrupt)

		<-signalQuit
		return nil
	}
}
