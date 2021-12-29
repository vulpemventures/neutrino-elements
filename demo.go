package main

import (
	"github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/node"
)

func main() {
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

	err = node.Run("localhost:18886")
	if err != nil {
		panic(err)
	}

}
