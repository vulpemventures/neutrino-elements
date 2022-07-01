package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/btcsuite/btcutil"
	"github.com/gorilla/websocket"
	"github.com/urfave/cli/v2"
	"io/ioutil"
	"net/url"
	"os"
	"path"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	neutrinoDataDir = btcutil.AppDataDir("neutrino", false)
	statePath       = path.Join(neutrinoDataDir, "state.json")
)

func main() {
	app := cli.NewApp()
	app.Version = formatVersion()
	app.Name = "Neutrino Daemon CLI"
	app.Usage = "Command line interface for neutrinod users"
	app.Commands = append(
		app.Commands,
		&configCmd,
		&subscribeCmd,
	)

	err := app.Run(os.Args)
	if err != nil {
		fatal(err)
	}
}

func formatVersion() string {
	return fmt.Sprintf(
		"\nVersion: %s\nCommit: %s\nDate: %s",
		version, commit, date,
	)
}

type invalidUsageError struct {
	ctx     *cli.Context
	command string
}

func (e *invalidUsageError) Error() string {
	return fmt.Sprintf("invalid usage of command %s", e.command)
}

func fatal(err error) {
	var e *invalidUsageError
	if errors.As(err, &e) {
		_ = cli.ShowCommandHelp(e.ctx, e.command)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "[neutrino] %v\n", err)
	}
	os.Exit(1)
}

func getState() (map[string]string, error) {
	data := map[string]string{}

	file, err := ioutil.ReadFile(statePath)
	if err != nil {
		return nil, errors.New("get config state error: try 'config cmd'")
	}
	json.Unmarshal(file, &data)

	return data, nil
}

func setState(data map[string]string) error {

	if _, err := os.Stat(neutrinoDataDir); os.IsNotExist(err) {
		os.Mkdir(neutrinoDataDir, os.ModeDir|0755)
	}

	file, err := os.OpenFile(statePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	currentData, err := getState()
	if err != nil {
		fmt.Println(err)
		return err
	}

	mergedData := merge(currentData, data)

	jsonString, err := json.Marshal(mergedData)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(statePath, jsonString, 0755)
	if err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}

	return nil
}

func merge(maps ...map[string]string) map[string]string {
	merge := make(map[string]string, 0)
	for _, m := range maps {
		for k, v := range m {
			merge[k] = v
		}
	}
	return merge
}

func getNeutrinodConnection() (*websocket.Conn, func(), error) {
	state, err := getState()
	if err != nil {
		return nil, nil, err
	}

	u := url.URL{Scheme: "ws", Host: state["url"], Path: state["path"]}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() { _ = conn.Close() }

	return conn, cleanup, nil
}
