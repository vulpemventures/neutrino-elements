package node

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/neutrino-elements/pkg/repository"
	"io"

	"github.com/vulpemventures/neutrino-elements/pkg/binary"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
)

func (n node) handleHeaders(msgHeader *protocol.MessageHeader, p peer.Peer) error {
	var headers protocol.MsgHeaders

	conn := p.Connection()
	lr := io.LimitReader(conn, int64(msgHeader.Length))

	if err := binary.NewDecoder(lr).Decode(&headers); err != nil {
		return err
	}

	tip, err := n.blockHeadersDb.ChainTip(context.Background())
	if err != nil {
		if err == repository.ErrNoBlocksHeaders {
			tip = &block.Header{
				Height: 0,
			}
		} else {
			return err
		}
	}

	if len(headers.Headers) == 0 {
		return nil
	}

	if !checkHeadersInSequence(headers) {
		return nil
	}

	//synchronization needs to be done in sequence since node is fetching
	//headers from the beginning of the chain in portions of 2K blocks
	//prevent storing un-sequenced headers in db, retry the request again
	firstHeaderBlockHeight := headers.Headers[0].Height
	lastHeaderBlockHeight := headers.Headers[len(headers.Headers)-1].Height
	if firstHeaderBlockHeight > tip.Height+1 {
		n.sync(p)
		return nil
	}

	if lastHeaderBlockHeight <= tip.Height {
		return nil
	}

	//in case received headers portion has first header block height less than tip
	//check if there are other received headers that are continuation of the tip in sequence
	if firstHeaderBlockHeight < tip.Height && lastHeaderBlockHeight > tip.Height {
		newHeaders := make([]*block.Header, 0)
		for _, header := range headers.Headers {
			if header.Height > tip.Height {
				newHeaders = append(newHeaders, header)
			}
		}

		if len(newHeaders) > 0 {
			headers.Headers = newHeaders
		}
	}

	if len(headers.Headers) > 0 {
		for _, v := range headers.Headers {
			n.blockHeadersCh <- *v
		}

		log.Debugf("node: local tip: %v", headers.Headers[len(headers.Headers)-1].Height)
		log.Debugf("node: peers tip: %v", p.PeersTip())

		n.sync(p)
	}

	return nil
}

func checkHeadersInSequence(headers protocol.MsgHeaders) bool {
	for i := 0; i < len(headers.Headers); i++ {
		if i < len(headers.Headers)-1 {
			if headers.Headers[i].Height != headers.Headers[i+1].Height-1 {
				return false
			}
		}
	}

	return true
}
