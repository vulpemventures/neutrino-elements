package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	neutrinodtypes "github.com/vulpemventures/neutrino-elements/pkg/neutrinod-types"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
)

var (
	emptyOnChainMsg = neutrinodtypes.WsOnChainEventResponse{}
	emptyGeneralMsg = neutrinodtypes.WsGeneralMessageResponse{}
	emptyErrorMsg   = neutrinodtypes.WsMessageErrorResponse{}
)

var subscribeCmd = cli.Command{
	Name:   "subscribe",
	Usage:  "subscribes to neutrinod events related to provided wallet descriptor",
	Action: subscribeAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "descriptor",
			Usage:    "wallet descriptor",
			Required: true,
		},
		&cli.IntFlag{
			Name:     "block_height",
			Usage:    "block height to watch from",
			Required: true,
		},
		&cli.IntFlag{
			Name: "events",
			Usage: "events to watch for:\n" +
				"	0 -> unspent utxo\n" +
				"	1 -> spent utxo\n",
			Value: 0,
		},
	},
}

func subscribeAction(ctx *cli.Context) error {
	conn, cleanup, err := getNeutrinodConnection()
	if err != nil {
		return err
	}
	defer cleanup()

	descriptor := ctx.String("descriptor")
	blockHeight := ctx.Int("block_height")
	eventType := ctx.Int("events")

	req := neutrinodtypes.WsMessageRequest{
		ActionType:       neutrinodtypes.Register,
		EventTypes:       []scanner.EventType{scanner.EventType(eventType)},
		DescriptorWallet: descriptor,
		StartBlockHeight: blockHeight,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	if err := conn.WriteMessage(websocket.TextMessage, reqBytes); err != nil {
		return err
	}

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		onChainMsg := neutrinodtypes.WsOnChainEventResponse{}
		if err := json.Unmarshal(message, &onChainMsg); err != nil {
			log.Error(err.Error())
			return err
		}

		generalMsg := neutrinodtypes.WsGeneralMessageResponse{}
		if err := json.Unmarshal(message, &generalMsg); err != nil {
			log.Error(err.Error())
			return err
		}

		errorMsg := neutrinodtypes.WsMessageErrorResponse{}
		if err := json.Unmarshal(message, &errorMsg); err != nil {
			log.Error(err.Error())
			return err
		}

		if onChainMsg != emptyOnChainMsg {
			if onChainMsg.TxID != "" {
				log.Infof("tx_id: %v", onChainMsg.TxID)
			}
		}

		if generalMsg != emptyGeneralMsg {
			log.Infoln(generalMsg.Message)
		}

		if errorMsg != emptyErrorMsg {
			log.Errorf("error: %v", errorMsg.ErrorMessage)
		}

	}
}
