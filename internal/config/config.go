package config

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/vulpemventures/go-elements/network"
	"time"
)

const (
	// NeutrinoDUrlKey is the key for the neutrino url
	NeutrinoDUrlKey = "NEUTRINOD_URL"
	// ExplorerUrlKey is the URL of the Liquid network
	ExplorerUrlKey = "EXPLORER_URL"
	//PeerUrlKey is the URL of the peer node
	PeerUrlKey = "PEER_URL"
	// NetworkKey is the network to use. Either liquid, testnet or regtest
	NetworkKey = "NETWORK"
	// LogLevelKey are the different logging levels. For reference on the values https://godoc.org/github.com/sirupsen/logrus#Level
	LogLevelKey = "LOG_LEVEL"
	// DbUserKey is user used to connect to db
	DbUserKey = "DB_USER"
	// DbPassKey is password used to connect to db
	DbPassKey = "DB_PASS"
	// DbHostKey is host where db is installed
	DbHostKey = "DB_HOST"
	// DbPortKey is port on which db is listening
	DbPortKey = "DB_PORT"
	// DbNameKey is name of database
	DbNameKey = "DB_NAME"
	// DbMigrationPath is the path to migration files
	DbMigrationPath = "DB_MIGRATION_PATH"
	// DbInsecure is used to define db connection url
	DbInsecure = "DB_INSECURE"
	// AwsRegion is AWS region in which RDS is running
	AwsRegion = "AWSREGION"
)

var (
	vip *viper.Viper
)

func LoadConfig() error {
	vip = viper.New()
	vip.SetEnvPrefix("NEUTRINO_ELEMENTS")
	vip.AutomaticEnv()

	vip.SetDefault(NeutrinoDUrlKey, "localhost:8000")
	vip.SetDefault(ExplorerUrlKey, "http://localhost:3001")
	vip.SetDefault(PeerUrlKey, "localhost:18886")
	vip.SetDefault(NetworkKey, network.Regtest.Name)
	vip.SetDefault(LogLevelKey, int(log.InfoLevel))
	vip.SetDefault(DbUserKey, "root")
	vip.SetDefault(DbPassKey, "secret")
	vip.SetDefault(DbHostKey, "127.0.0.1")
	vip.SetDefault(DbPortKey, 5432)
	vip.SetDefault(DbNameKey, "neutrino-elements")
	vip.SetDefault(DbMigrationPath, "file://internal/infrastructure/storage/db/pg/migration")
	vip.SetDefault(DbInsecure, true)
	vip.SetDefault(AwsRegion, "eu-central-1")

	networkName := GetString(NetworkKey)
	if networkName != network.Liquid.Name &&
		networkName != network.Testnet.Name &&
		networkName != network.Regtest.Name {
		return fmt.Errorf(
			"network must be either %v, %v or %v",
			network.Liquid.Name,
			network.Testnet.Name,
			network.Regtest.Name,
		)
	}

	log.SetLevel(log.Level(GetInt(LogLevelKey)))

	return nil
}

func GetString(key string) string {
	return vip.GetString(key)
}

func GetInt(key string) int {
	return vip.GetInt(key)
}

func GetFloat(key string) float64 {
	return vip.GetFloat64(key)
}

func GetDuration(key string) time.Duration {
	return vip.GetDuration(key)
}

func GetBool(key string) bool {
	return vip.GetBool(key)
}

func GetStringSlice(key string) []string {
	return vip.GetStringSlice(key)
}
