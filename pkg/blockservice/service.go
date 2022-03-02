package blockservice

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/vulpemventures/go-elements/block"
)

var ErrorBlockNotFound = fmt.Errorf("block not found")

type BlockService interface {
	GetBlock(hash *chainhash.Hash) (*block.Block, error)
}

type esploraBlockService struct {
	esploraURL string
}

var _ BlockService = (*esploraBlockService)(nil)

func NewEsploraBlockService(esploraURL string) BlockService {
	return &esploraBlockService{
		esploraURL: esploraURL,
	}
}

func (b *esploraBlockService) GetBlock(hash *chainhash.Hash) (*block.Block, error) {
	url := fmt.Sprintf(
		"%v/block/%v/raw",
		b.esploraURL,
		hash.String(),
	)

	httpClient := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return nil, ErrorBlockNotFound
		}

		return nil, fmt.Errorf("getBlock http get error, status code: %v", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	block, err := block.NewFromBuffer(bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	return block, nil
}
