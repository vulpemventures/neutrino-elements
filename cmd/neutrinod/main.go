package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/internal/config"
	dbpg "github.com/vulpemventures/neutrino-elements/internal/infrastructure/storage/db/pg"
	neutrinodws "github.com/vulpemventures/neutrino-elements/internal/interface/web-socket"
	"github.com/vulpemventures/neutrino-elements/pkg/blockservice"
	"github.com/vulpemventures/neutrino-elements/pkg/node"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatal(err)
	}

	dbManager, err := dbpg.NewDbService(dbpg.DbConfig{
		DbUser:             config.GetString(config.DbUserKey),
		DbPassword:         config.GetString(config.DbPassKey),
		DbHost:             config.GetString(config.DbHostKey),
		DbPort:             config.GetInt(config.DbPortKey),
		DbName:             config.GetString(config.DbNameKey),
		MigrationSourceURL: config.GetString(config.DbMigrationPath),
		DbInsecure:         config.GetBool(config.DbInsecure),
		AwsRegion:          config.GetString(config.AwsRegion),
	})
	if err != nil {
		log.Fatal(err)
	}

	repoFilter, err := dbpg.NewFilterRepositoryImpl(dbManager)
	if err != nil {
		log.Fatal(err)
	}

	repoHeader, err := dbpg.NewHeaderRepositoryImpl(dbManager)
	if err != nil {
		log.Fatal(err)
	}

	nodeCfg := node.NodeConfig{
		Network:        config.GetString(config.NetworkKey),
		UserAgent:      "neutrino-elements:test",
		FiltersDB:      repoFilter,
		BlockHeadersDB: repoHeader,
	}

	blockSvc := blockservice.NewEsploraBlockService(config.GetString(config.ExplorerUrlKey))

	elementsNeutrinoServer, err := neutrinodws.NewElementsNeutrinoServer(
		nodeCfg,
		blockSvc,
		config.GetString(config.PeerUrlKey),
		config.GetString(config.NeutrinoDUrlKey),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	errC := elementsNeutrinoServer.Start(ctx, stop)
	if err := <-errC; err != nil {
		log.Panicf("neutrinod: neutrino-elements daemon noticed error while running: %s", err)
	}
}
