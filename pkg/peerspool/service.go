package peerspool

import (
	"github.com/vulpemventures/neutrino-elements/internal/domain"
	"github.com/vulpemventures/neutrino-elements/pkg/peer"
)

// PeersPool maintains a list of peers in order to keep a sync state of the blockchain - block headers + filters
type PeersPool interface {
	Start() error
	Stop() error
	Connect(peer peer.Peer) error
}

type peersPool struct {
	peers            []peer.Peer
	filterRepository domain.FilterRepository
	headerRepository domain.BlockHeaderRepository
}

// var _ PeersPool = (*peersPool)(nil)
