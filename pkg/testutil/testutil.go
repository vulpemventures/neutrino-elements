package testutil

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/vulpemventures/go-bip39"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/neutrino-elements/pkg/blockservice"
	"github.com/vulpemventures/neutrino-elements/pkg/node"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"github.com/vulpemventures/neutrino-elements/pkg/repository/inmemory"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
)

var (
	baseURL = "http://localhost:3001"
	lbtc    = append(
		[]byte{0x01},
		elementsutil.ReverseBytes(H2b(network.Regtest.AssetID))...,
	)
	PeerAddrLocal   = "localhost:18886"
	EsploraUrlLocal = "http://localhost:3001"
)

func CreateTx() (string, string, error) {
	privkey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return "", "", err
	}
	pubkey := privkey.PubKey()
	p2wpkh := payment.FromPublicKey(pubkey, &network.Regtest, nil)
	addr, _ := p2wpkh.WitnessPubKeyHash()

	// Fund sender address.
	if _, err := Faucet(addr); err != nil {
		return "", "", err
	}

	// Retrieve sender utxos.
	utxos, err := Unspents(addr)
	if err != nil {
		return "", "", err
	}

	// The transaction will have 1 input and 3 outputs.
	txInputHash := elementsutil.ReverseBytes(H2b(utxos[0]["txid"].(string)))
	txInputIndex := uint32(utxos[0]["vout"].(float64))
	txInput := transaction.NewTxInput(txInputHash, txInputIndex)

	receiverValue, _ := elementsutil.SatoshiToElementsValue(60000000)
	receiverScript := H2b("76a91439397080b51ef22c59bd7469afacffbeec0da12e88ac")
	receiverOutput := transaction.NewTxOutput(lbtc, receiverValue, receiverScript)

	changeScript := p2wpkh.WitnessScript
	changeValue, _ := elementsutil.SatoshiToElementsValue(39999500)
	changeOutput := transaction.NewTxOutput(lbtc, changeValue, changeScript)

	// Create a new pset with all the outputs that need to be blinded first
	inputs := []*transaction.TxInput{txInput}
	outputs := []*transaction.TxOutput{receiverOutput, changeOutput}
	p, err := pset.New(inputs, outputs, 2, 0)
	if err != nil {
		return "", "", err
	}

	// Add sighash type and witness utxo to the partial input.
	updater, err := pset.NewUpdater(p)
	if err != nil {
		return "", "", err
	}

	witValue, _ := elementsutil.SatoshiToElementsValue(uint64(utxos[0]["value"].(float64)))
	witnessUtxo := transaction.NewTxOutput(lbtc, witValue, p2wpkh.WitnessScript)
	if err := updater.AddInWitnessUtxo(witnessUtxo, 0); err != nil {
		return "", "", err
	}

	//blind outputs
	inBlindingPrvKeys := [][]byte{{}}
	outBlindingPrvKeys := make([][]byte, 2)
	for i := range outBlindingPrvKeys {
		pk, err := btcec.NewPrivateKey(btcec.S256())
		if err != nil {
			return "", "", err
		}
		outBlindingPrvKeys[i] = pk.Serialize()
	}

	if err := BlindTransaction(
		p,
		inBlindingPrvKeys,
		outBlindingPrvKeys,
		nil,
	); err != nil {
		return "", "", err
	}

	// Add the unblinded outputs now, that's only the fee output in this case
	addFeesToTransaction(p, 500)

	prvKeys := []*btcec.PrivateKey{privkey}
	scripts := [][]byte{p2wpkh.Script}
	if err := SignTransaction(p, prvKeys, scripts, true, nil); err != nil {
		return "", "", err
	}

	if err := pset.FinalizeAll(p); err != nil {
		return "", "", err
	}
	// Extract the final signed transaction from the Pset wrapper.
	finalTx, err := pset.Extract(p)
	if err != nil {
		return "", "", err
	}
	// Serialize the transaction and try to broadcast.
	txHex, err := finalTx.ToHex()
	if err != nil {
		return "", "", err
	}

	return txHex, finalTx.TxHash().String(), nil
}

func Unspents(address string) ([]map[string]interface{}, error) {
	getUtxos := func(address string) ([]interface{}, error) {
		url := fmt.Sprintf("%s/address/%s/utxo", baseURL, address)
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var respBody interface{}
		if err := json.Unmarshal(data, &respBody); err != nil {
			return nil, err
		}
		return respBody.([]interface{}), nil
	}

	utxos := []map[string]interface{}{}
	for len(utxos) <= 0 {
		time.Sleep(1 * time.Second)
		u, err := getUtxos(address)
		if err != nil {
			return nil, err
		}
		for _, unspent := range u {
			utxo := unspent.(map[string]interface{})
			utxos = append(utxos, utxo)
		}
	}

	return utxos, nil
}

func H2b(str string) []byte {
	buf, _ := hex.DecodeString(str)
	return buf
}

func Faucet(address string) (string, error) {
	url := fmt.Sprintf("%s/faucet", baseURL)
	payload := map[string]string{"address": address}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if res := string(data); len(res) <= 0 || strings.Contains(res, "sendtoaddress") {
		return "", fmt.Errorf("cannot fund address with faucet: %s", res)
	}

	respBody := map[string]string{}
	if err := json.Unmarshal(data, &respBody); err != nil {
		return "", err
	}
	return respBody["txId"], nil
}

func GetTransactionHex(txHash string) (string, error) {
	url := fmt.Sprintf(
		"%s/tx/%s/hex",
		baseURL,
		txHash,
	)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %s", resp.Status, string(body))
	}

	return string(body), nil
}

func BlindTransaction(
	p *pset.Pset,
	inBlindKeys [][]byte,
	outBlindKeys [][]byte,
	issuanceBlindKeys []pset.IssuanceBlindingPrivateKeys,
) error {
	outputsPrivKeyByIndex := make(map[int][]byte, 0)
	for index, output := range p.UnsignedTx.Outputs {
		if len(output.Script) > 0 {
			outputsPrivKeyByIndex[index] = outBlindKeys[index]
		}
	}

	return BlindTransactionByIndex(p, inBlindKeys, outputsPrivKeyByIndex, issuanceBlindKeys)
}

func BlindTransactionByIndex(
	p *pset.Pset,
	inBlindKeys [][]byte,
	outBlindKeysMap map[int][]byte,
	issuanceBlindKeys []pset.IssuanceBlindingPrivateKeys,
) error {
	outBlindPubKeysMap := make(map[int][]byte)
	for index, k := range outBlindKeysMap {
		_, pubkey := btcec.PrivKeyFromBytes(btcec.S256(), k)
		outBlindPubKeysMap[index] = pubkey.SerializeCompressed()
	}

	psetBase64, err := p.ToBase64()
	if err != nil {
		return err
	}

	for {
		blindDataLike := make([]pset.BlindingDataLike, len(inBlindKeys), len(inBlindKeys))
		for i, inBlinKey := range inBlindKeys {
			blindDataLike[i] = pset.PrivateBlindingKey(inBlinKey)
		}

		ptx, _ := pset.NewPsetFromBase64(psetBase64)
		blinder, err := pset.NewBlinder(
			ptx,
			blindDataLike,
			outBlindPubKeysMap,
			issuanceBlindKeys,
			nil,
		)
		if err != nil {
			return err
		}

		for {
			if err := blinder.Blind(); err != nil {
				if err != pset.ErrGenerateSurjectionProof {
					return err
				}
				continue
			}
			break
		}

		verify, err := pset.VerifyBlinding(ptx, blindDataLike, outBlindKeysMap, issuanceBlindKeys)
		if err != nil {
			return err
		}

		if verify {
			*p = *ptx
			break
		}
	}

	return nil
}

type signOpts struct {
	pubkeyScript []byte
	script       []byte
}

func SignTransaction(
	p *pset.Pset,
	privKeys []*btcec.PrivateKey,
	scripts [][]byte,
	forWitness bool,
	opts *signOpts,
) error {
	updater, err := pset.NewUpdater(p)
	if err != nil {
		return err
	}

	for i, in := range p.Inputs {
		if err := updater.AddInSighashType(txscript.SigHashAll, i); err != nil {
			return err
		}

		var prevout *transaction.TxOutput
		if in.WitnessUtxo != nil {
			prevout = in.WitnessUtxo
		} else {
			prevout = in.NonWitnessUtxo.Outputs[p.UnsignedTx.Inputs[i].Index]
		}
		prvkey := privKeys[i]
		pubkey := prvkey.PubKey()
		script := scripts[i]

		var sigHash [32]byte
		if forWitness {
			sigHash = p.UnsignedTx.HashForWitnessV0(
				i,
				script,
				prevout.Value,
				txscript.SigHashAll,
			)
		} else {
			sigHash, err = p.UnsignedTx.HashForSignature(i, script, txscript.SigHashAll)
			if err != nil {
				return err
			}
		}

		sig, err := prvkey.Sign(sigHash[:])
		if err != nil {
			return err
		}
		sigWithHashType := append(sig.Serialize(), byte(txscript.SigHashAll))

		var witPubkeyScript []byte
		var witScript []byte
		if opts != nil {
			witPubkeyScript = opts.pubkeyScript
			witScript = opts.script
		}

		if _, err := updater.Sign(
			i,
			sigWithHashType,
			pubkey.SerializeCompressed(),
			witPubkeyScript,
			witScript,
		); err != nil {
			return err
		}
	}

	valid, err := p.ValidateAllSignatures()
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("invalid signatures")
	}

	return nil
}

func addFeesToTransaction(p *pset.Pset, feeAmount uint64) {
	updater, _ := pset.NewUpdater(p)
	feeScript := []byte{}
	feeValue, _ := elementsutil.SatoshiToElementsValue(feeAmount)
	feeOutput := transaction.NewTxOutput(lbtc, feeValue, feeScript)
	updater.AddOutput(feeOutput)
}

var repoFilter = inmemory.NewFilterInmemory()
var repoHeader = inmemory.NewHeaderInmemory()

func MakeNigiriTestServices(peerUrl, esploraUrl string) (node.NodeService, scanner.ScannerService, <-chan scanner.Report) {
	n, err := node.New(node.NodeConfig{
		Network:        "nigiri",
		UserAgent:      "neutrino-elements:test",
		FiltersDB:      repoFilter,
		BlockHeadersDB: repoHeader,
	})

	if err != nil {
		panic(err)
	}

	err = n.Start(peerUrl)
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second * 3) // wait for the node sync the first headers if the repo is empty

	blockSvc := blockservice.NewEsploraBlockService(esploraUrl)
	genesisBlockHash := protocol.GetCheckpoints(protocol.MagicNigiri)[0]
	h, err := chainhash.NewHashFromStr(genesisBlockHash)
	if err != nil {
		panic(err)
	}
	s := scanner.New(repoFilter, repoHeader, blockSvc, h)

	reportCh, err := s.Start()
	if err != nil {
		panic(err)
	}

	return n, s, reportCh
}

func GenerateMasterPrivateKey() (*hdkeychain.ExtendedKey, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return nil, err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, err
	}

	seed := bip39.NewSeed(mnemonic, "")

	privateKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}
