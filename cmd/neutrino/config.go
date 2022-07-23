package main

import (
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"
)

var configCmd = cli.Command{
	Name:   "config",
	Usage:  "Configures gate cli",
	Action: configure,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "url",
			Usage: "neutrino daemon address host:port",
			Value: "localhost:8000",
		},
		&cli.StringFlag{
			Name:  "path",
			Usage: "neutrino daemon url path",
			Value: "/neutrino/subscribe/ws",
		},
	},
	Subcommands: []*cli.Command{
		{
			Name:   "set",
			Usage:  "set individual <key> <value> in the local state",
			Action: configSetAction,
		},
		{
			Name:   "print",
			Usage:  "Print local configuration of the neutrino CLI",
			Action: configList,
		},
	},
}

func configure(ctx *cli.Context) error {
	configState := make(map[string]string)
	configState["url"] = ctx.String("url")
	configState["path"] = ctx.String("path")

	return setState(configState)
}

func configSetAction(c *cli.Context) error {

	if c.NArg() < 2 {
		return errors.New("key and value are missing")
	}

	key := c.Args().Get(0)
	value := c.Args().Get(1)

	err := setState(map[string]string{key: value})
	if err != nil {
		return err
	}

	fmt.Printf("%s %s has been set\n", key, value)

	return nil
}

func configList(ctx *cli.Context) error {

	state, err := getState()
	if err != nil {
		return err
	}

	for key, value := range state {
		fmt.Println(key + ": " + value)
	}

	return nil
}
