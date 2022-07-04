package neutrinodws

import (
	"context"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/internal/core/application"
	"github.com/vulpemventures/neutrino-elements/internal/interface/web-socket/handler"
	"github.com/vulpemventures/neutrino-elements/internal/interface/web-socket/middleware"
	"github.com/vulpemventures/neutrino-elements/pkg/blockservice"
	"github.com/vulpemventures/neutrino-elements/pkg/node"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
	"net/http"
	"time"
)

const (
	shutdownTimeout = 2 * time.Second
)

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

	log.Infoln("neutrinod: waiting for node to sync with peer...")
	if err := n.nodeSvc.Start(n.peerUrl); err != nil {
		errC <- err
	}
	log.Infoln("neutrinod: node is synced with peer")

	genesisBlockHashStr := protocol.GetCheckpoints(protocol.MagicRegtest)[0]
	genesisBlockHash, err := chainhash.NewHashFromStr(genesisBlockHashStr)
	if err != nil {
		errC <- err
	}

	scannerSvc := scanner.New(
		n.nodeCfg.FiltersDB,
		n.nodeCfg.BlockHeadersDB,
		n.blockSvc,
		genesisBlockHash,
	)

	notificationSvc := application.NewNotificationService(scannerSvc)

	if err := notificationSvc.Start(); err != nil {
		errC <- err
	}

	descriptorWalletNotifierSvc := handler.NewDescriptorWalletNotifierHandler(notificationSvc)
	descriptorWalletNotifierSvc.Start()

	middlewareSvc := middleware.NewMiddlewareService()
	middlewares := []middleware.Middleware{
		middleware.LoggingMiddleware,
		middleware.PanicRecovery,
	}

	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc(
		"/neutrino/subscribe/ws",
		middlewareSvc.WrapHandlerWithMiddlewares(
			descriptorWalletNotifierSvc.HandleSubscriptionRequestWs, middlewares...),
	)

	muxRouter.HandleFunc(
		"/neutrino/subscribe/http",
		middlewareSvc.WrapHandlerWithMiddlewares(
			descriptorWalletNotifierSvc.HandleSubscriptionRequestHttp, middlewares...),
	)

	httpServer := &http.Server{
		Addr:    n.serverAddress,
		Handler: muxRouter,
	}

	go func() {
		<-ctx.Done()

		log.Info("neutrinod: shutdown signal received")

		ctxTimeout, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

		defer func() {
			descriptorWalletNotifierSvc.Stop()
			scannerSvc.Stop()

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

		log.Info("neutrinod: neutrino-elements daemon graceful shutdown completed")
	}()

	// start http server
	go func() {
		log.Infof(
			"neutrinod: neutrino-elements daemon listening and serving at: %v",
			n.serverAddress)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errC <- err
		}
	}()

	return errC
}
