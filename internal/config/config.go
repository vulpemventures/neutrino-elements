package config

import (
	"fmt"
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
)

var (
	vip *viper.Viper
)

func LoadConfig() error {
	vip = viper.New()
	vip.SetEnvPrefix("NEUTRINO")
	vip.AutomaticEnv()

	vip.SetDefault(NeutrinoDUrlKey, "localhost:8080")
	vip.SetDefault(ExplorerUrlKey, "http://localhost:3001")
	vip.SetDefault(PeerUrlKey, "localhost:18886")
	vip.SetDefault(NetworkKey, network.Regtest.Name)

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
