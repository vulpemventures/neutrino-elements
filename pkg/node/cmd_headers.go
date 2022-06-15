package node

import (
	"context"
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

	//synchronization needs to be done in sequence since node is fetching
	//headers from the beginning of the chain in portions of 2K blocks
	//prevent storing un-sequenced headers in db
	if headers.Headers[0].Height != tip.Height+1 {
		return nil
	}

	for _, v := range headers.Headers {
		n.blockHeadersCh <- *v
	}

	n.sync(p)

	return nil
}
