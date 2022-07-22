package pgtest

import (
	"bytes"
	"encoding/hex"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/vulpemventures/go-elements/block"
)

func (s *PgDbTestSuite) TestChainTip() {
	tip, err := headerRepo.ChainTip(ctx)
	if err != nil {
		s.FailNow(err.Error())
	}

	hash, err := tip.Hash()
	if err != nil {
		s.FailNow(err.Error())
	}

	prevHash, err := chainhash.NewHash(tip.PrevBlockHash)
	if err != nil {
		s.FailNow(err.Error())
	}

	s.Equal(uint32(10), tip.Height)
	s.Equal("278a266440efc99fae9f62a0812ead54c79b74ca961b9f6eb42e11e9b0c74875", hash.String())
	s.Equal("e753485de599299193ef013bc8a04605fabe86c9b8f6ee4410c2ea9186c30c8c", prevHash.String())
}

func (s *PgDbTestSuite) TestGetBlockHeader() {
	hash, err := chainhash.NewHashFromStr("278a266440efc99fae9f62a0812ead54c79b74ca961b9f6eb42e11e9b0c74875")
	if err != nil {
		s.FailNow(err.Error())
	}

	bh, err := headerRepo.GetBlockHeader(ctx, *hash)
	if err != nil {
		s.FailNow(err.Error())
	}

	s.Equal(uint32(10), bh.Height)
	s.Equal("278a266440efc99fae9f62a0812ead54c79b74ca961b9f6eb42e11e9b0c74875", hash.String())
}

func (s *PgDbTestSuite) TestGetBlockHashByHeight() {
	hash, err := headerRepo.GetBlockHashByHeight(ctx, 10)
	if err != nil {
		s.FailNow(err.Error())
	}

	s.Equal("278a266440efc99fae9f62a0812ead54c79b74ca961b9f6eb42e11e9b0c74875", hash.String())
}

func (s *PgDbTestSuite) TestHasAllAncestors() {
	hash, err := chainhash.NewHashFromStr("278a266440efc99fae9f62a0812ead54c79b74ca961b9f6eb42e11e9b0c74875")
	if err != nil {
		s.FailNow(err.Error())
	}

	hasAllAncestors, err := headerRepo.HasAllAncestors(ctx, *hash)
	if err != nil {
		s.FailNow(err.Error())
	}

	s.Equal(true, hasAllAncestors)
}

func (s *PgDbTestSuite) TestWriteHeaders() {
	block11 := "000000a07548c7b0e9112eb46e9f1b96ca749bc754ad2e81a0629fae9fc9ef4064268a271912a11c2035cc26032387ca459768fea7e01e11f01f591d3a6acdb76d81a752c752da620b000000012200204ae81572f06e1b88fd5ced7a1a000945432e83e1551e6f721ee9c00b8cc332604a000000fbee9cea00d8efdc49cfbec328537e0d7032194de6ebf3cf42e5c05bb89a08b100010151010200000001010000000000000000000000000000000000000000000000000000000000000000ffffffff035b0101ffffffff020125b251070e29ca19043cf33ccd7324e2ddab03ecc4ae0b5e77c4fc0e5cf6c95a01000000000000000000016a0125b251070e29ca19043cf33ccd7324e2ddab03ecc4ae0b5e77c4fc0e5cf6c95a01000000000000000000266a24aa21a9ed94f15ed3a62165e4a0b99699cc28b48e19cb5bc1b1f47155db62d63f1e047d45000000000000012000000000000000000000000000000000000000000000000000000000000000000000000000"
	block12 := "000000a0078938ba8644baacf4d861b994b8075fbc594511dcecf40ba236c4b7b932d16eafa247a9d4acd19400aa8006ad5790cf89f7678942a3ca58c7d3b4ec1c5542da835eda620c000000012200204ae81572f06e1b88fd5ced7a1a000945432e83e1551e6f721ee9c00b8cc332604a000000fbee9cea00d8efdc49cfbec328537e0d7032194de6ebf3cf42e5c05bb89a08b100010151020200000001010000000000000000000000000000000000000000000000000000000000000000ffffffff035c0101ffffffff020125b251070e29ca19043cf33ccd7324e2ddab03ecc4ae0b5e77c4fc0e5cf6c95a010000000000000016001600148da94ba4d8c16c20399ff551f7a2f15e2ae27b2d0125b251070e29ca19043cf33ccd7324e2ddab03ecc4ae0b5e77c4fc0e5cf6c95a01000000000000000000266a24aa21a9ed9b1d0d73201f13ebd5461521c6ecd7bee90953e5041e9ebc697682543b04997d0000000000000120000000000000000000000000000000000000000000000000000000000000000000000000000200000000019bb9df0a0c1bd4764e4aa201a357f43fec0d268359b3a5b3bdab41643e1da5120000000000feffffff030125b251070e29ca19043cf33ccd7324e2ddab03ecc4ae0b5e77c4fc0e5cf6c95a01000775f054115eea0016001490a46d8dd99d8762aa6de5783902d874abc3bac50125b251070e29ca19043cf33ccd7324e2ddab03ecc4ae0b5e77c4fc0e5cf6c95a010000000005f5e1000016001485a1c3ea366a4955b135bbf6a564b8aa2ef7e0f20125b251070e29ca19043cf33ccd7324e2ddab03ecc4ae0b5e77c4fc0e5cf6c95a01000000000000001600000b000000"
	blocks := []string{block11, block12}
	headers := make([]block.Header, 0, len(blocks))
	for _, v := range blocks {
		blockBytes, err := hex.DecodeString(v)
		if err != nil {
			s.FailNow(err.Error())
		}

		b, err := block.NewFromBuffer(bytes.NewBuffer(blockBytes))
		if err != nil {
			s.FailNow(err.Error())
		}

		headers = append(headers, *b.Header)
	}

	if err := headerRepo.WriteHeaders(ctx, headers...); err != nil {
		s.FailNow(err.Error())
	}

	hash, err := headerRepo.GetBlockHashByHeight(ctx, 12)
	if err != nil {
		s.FailNow(err.Error())
	}

	s.Equal("3a3712abba3d5161d58a324a4ac022bde782471e1d81bc5d4b45f2cf782c26db", hash.String())
}

func (s *PgDbTestSuite) TestLatestBlockLocator() {
	locator, err := headerRepo.LatestBlockLocator(ctx)
	if err != nil {
		s.FailNow(err.Error())
	}

	s.Equal(11, len(locator))
}
