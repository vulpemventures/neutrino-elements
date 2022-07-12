package e2etest

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/btcsuite/btcd/btcec"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	neutrinodtypes "github.com/vulpemventures/neutrino-elements/pkg/neutrinod-types"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
	"github.com/vulpemventures/neutrino-elements/pkg/testutil"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

var (
	neutrinoClients = make(map[int]string)
)

func (e *E2ESuite) TestEnd2EndHttp() {
	clientEvents := make(chan neutrinodtypes.OnChainEventResponse)

	neutrinoClientsServerUrl := make(map[int]*httptest.Server)
	for i := 0; i < 3; i++ {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var subscriptionReq neutrinodtypes.OnChainEventResponse

			err := json.NewDecoder(r.Body).Decode(&subscriptionReq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			clientEvents <- subscriptionReq
		}))

		neutrinoClientsServerUrl[i] = server
		neutrinoClients[i] = server.URL
	}
	defer func() {
		for _, server := range neutrinoClientsServerUrl {
			server.Close()
		}
	}()

	httpRequests := createHttpTxs(e.T())

	wg := sync.WaitGroup{}
	for _, v := range httpRequests {
		wg.Add(1)
		go func(b neutrinodtypes.SubscriptionRequestHttp) {
			invokeNeutrinodHttp(e.T(), &wg, b)
		}(v)
	}
	wg.Wait()

	expectedNumOfEvents := 6
	numberOfEvents := 0

loop:
	for {
		select {
		case event := <-clientEvents:
			numberOfEvents++
			e.T().Logf("Received event: %+v", event)
			if numberOfEvents == expectedNumOfEvents {
				break loop
			}
		case <-time.After(time.Minute):
			break loop
		}
	}

	e.Equal(expectedNumOfEvents, numberOfEvents)
}

func createHttpTxs(t *testing.T) []neutrinodtypes.SubscriptionRequestHttp {
	resp := make([]neutrinodtypes.SubscriptionRequestHttp, 0)
	for i := 0; i < len(neutrinoClients); i++ {
		privkey, err := btcec.NewPrivateKey(btcec.S256())
		if err != nil {
			t.Fatal(err)
		}
		pubkey := privkey.PubKey()
		p2wpkh := payment.FromPublicKey(pubkey, &network.Regtest, nil)
		addr, _ := p2wpkh.WitnessPubKeyHash()
		wpkhWalletDescriptor := fmt.Sprintf("wpkh(%v)", hex.EncodeToString(pubkey.SerializeCompressed()))

		req := neutrinodtypes.SubscriptionRequestHttp{
			ActionType:       "register",
			EventTypes:       []scanner.EventType{scanner.UnspentUtxo},
			DescriptorWallet: wpkhWalletDescriptor,
			StartBlockHeight: 0,
			EndpointUrl:      neutrinoClients[i],
		}
		resp = append(resp, req)

		_, err = testutil.Faucet(addr)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Second * 2)

		_, err = testutil.Faucet(addr)
		if err != nil {
			t.Fatal(err)
		}
	}

	return resp
}

func invokeNeutrinodHttp(
	t *testing.T,
	wg *sync.WaitGroup,
	req neutrinodtypes.SubscriptionRequestHttp,
) {
	defer wg.Done()

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post("http://localhost:8000/neutrino/subscribe/http", "application/json", bytes.NewBuffer(reqBytes))
	if err != nil {
		t.Fatal(err)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(b))

	defer resp.Body.Close()
}
