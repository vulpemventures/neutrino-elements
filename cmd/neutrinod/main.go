package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/blockservice"
	"github.com/vulpemventures/neutrino-elements/pkg/node"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository/inmemory"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
	"os"
	"os/signal"
	"syscall"

	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const (
	pongWait        = 60 * time.Second
	maxMessageSize  = 512
	shutdownTimeout = 2 * time.Second

	unspents EventType = "UNSPENT"

	nigiriUrl    = "localhost:18886"
	neutrinodUrl = "localhost:8080"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func main() {
	repoFilter := inmemory.NewFilterInmemory()
	repoHeader := inmemory.NewHeaderInmemory()

	nodeCfg := node.NodeConfig{
		Network:        "nigiri",
		UserAgent:      "neutrino-elements:test",
		FiltersDB:      repoFilter,
		BlockHeadersDB: repoHeader,
	}

	blockSvc := blockservice.NewEsploraBlockService("http://localhost:3001")

	peerUrl := nigiriUrl
	if os.Getenv("PEER_URL") != "" {
		peerUrl = os.Getenv("PEER_URL")
	}

	serverAddress := neutrinodUrl
	if os.Getenv("NEUTRINOD_URL") != "" {
		serverAddress = os.Getenv("NEUTRINOD_URL")
	}

	elementsNeutrinoDaemon, err := NewElementsNeutrinoServer(
		nodeCfg,
		blockSvc,
		peerUrl,
		serverAddress,
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	errC := elementsNeutrinoDaemon.Start(ctx, stop)
	if err := <-errC; err != nil {
		log.Panicf("neutrino-elements daemon server noticed error while running: %s", err)
	}
}

type NeutrinoServer struct {
	nodeSvc       node.NodeService
	nodeCfg       node.NodeConfig
	blockSvc      blockservice.BlockService
	peerUrl       string
	serverAddress string
}

func NewElementsNeutrinoServer(
	nodeCfg node.NodeConfig,
	blockSvc blockservice.BlockService,
	peerUrl string,
	serverAddress string,
) (*NeutrinoServer, error) {
	nodeSvc, err := node.New(nodeCfg)
	if err != nil {
		return nil, err
	}

	return &NeutrinoServer{
		nodeSvc:       nodeSvc,
		nodeCfg:       nodeCfg,
		blockSvc:      blockSvc,
		peerUrl:       peerUrl,
		serverAddress: serverAddress,
	}, nil
}

func (n *NeutrinoServer) Start(ctx context.Context, stop context.CancelFunc) <-chan error {
	errC := make(chan error, 1)

	if err := n.nodeSvc.Start(n.peerUrl); err != nil {
		errC <- err
	}

	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/neutrino", n.wsHandler)

	httpServer := &http.Server{
		Addr:    n.serverAddress,
		Handler: muxRouter,
	}

	go func() {
		<-ctx.Done()

		log.Info("shutdown signal received")

		ctxTimeout, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

		defer func() {
			if err := n.nodeSvc.Stop(); err != nil {
				errC <- err
			}

			stop()
			cancel()
			close(errC)
		}()

		httpServer.SetKeepAlivesEnabled(false)
		if err := httpServer.Shutdown(ctxTimeout); err != nil {
			errC <- err
		}

		log.Info("neutrino-elements daemon server graceful shutdown completed")
	}()

	// start http server
	go func() {
		log.Infof(
			"neutrino-elements daemon listening and serving at: %v",
			n.serverAddress)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errC <- err
		}
	}()

	return errC
}

func (n *NeutrinoServer) wsHandler(w http.ResponseWriter, r *http.Request) {
	wsUpgrader := websocket.Upgrader{}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("upgrading error: %#v\n", err)
		return
	}

	go n.handleRequest(conn)
}

func (n *NeutrinoServer) handleRequest(conn *websocket.Conn) {
	defer func() {
		conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		if err := conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
			log.Errorf("Error writing close message: %#v\n", err)
		}

		log.Error(err)
		return
	}

	conn.SetPongHandler(
		func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		},
	)

	log.Info("new connection")

	done := make(chan struct{})
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if e, ok := err.(*websocket.CloseError); ok {
				if e.Code != websocket.CloseNormalClosure {
					log.Errorf(
						"Error reading message: %v, error code: %v\n",
						e.Text,
						e.Code,
					)
				}
			} else {
				log.Errorf("Error reading message: %v\n", err)
			}

			done <- struct{}{}

			return
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		wsMsg := &WsMessageReq{}
		if err = json.Unmarshal(message, wsMsg); err != nil {
			log.Error(err)
			return
		}

		log.Infof(
			"new descriptor wallet request: %v, for event: %v\n",
			wsMsg.DescriptorWallet,
			wsMsg.EventType,
		)

		go func(msg WsMessageReq) {
			switch EventType(msg.EventType) {
			case unspents:
				genesisBlockHashStr := protocol.GetCheckpoints(protocol.MagicNigiri)[0]
				genesisBlockHash, err := chainhash.NewHashFromStr(genesisBlockHashStr)
				if err != nil {
					log.Error(err)
					return
				}

				scannerSvc := scanner.New(
					n.nodeCfg.FiltersDB,
					n.nodeCfg.BlockHeadersDB,
					n.blockSvc,
					genesisBlockHash,
				)

				if err := scannerSvc.WatchDescriptorWallet(
					1,
					wsMsg.DescriptorWallet,
					[]scanner.EventType{scanner.UnspentUtxo},
					wsMsg.StartBlockHeight,
				); err != nil {
					log.Error(err)
					return
				}

				reportCh, err := scannerSvc.Start()
				if err != nil {
					log.Error(err)
					return
				}

				for {
					select {
					case <-done:
						scannerSvc.Stop()

						log.Info("connection closed")
						return
					case r := <-reportCh:
						resp, err := json.Marshal(WsMessageRes{
							EventType: string(unspents),
							TxID:      r.Transaction.TxHash().String(),
						})
						if err != nil {
							log.Error(err)
							return
						}

						if err = conn.WriteMessage(websocket.TextMessage, resp); err != nil {
							log.Error(err)
							return
						}
					}
				}
			}
		}(*wsMsg)
	}
}

type EventType string

type WsMessageReq struct {
	EventType        string `json:"eventType"`
	DescriptorWallet string `json:"descriptorWallet"`
	StartBlockHeight int    `json:"startBlockHeight"`
}

type WsMessageRes struct {
	EventType string `json:"eventType"`
	TxID      string `json:"txId"`
}
