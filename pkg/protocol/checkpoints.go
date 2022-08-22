package protocol

type Checkpoints map[uint32]string

// TODO populate checkpoints and add check during node sync
var (
	// Checkpoints for mainnet
	mainnetCheckpoints = Checkpoints{
		0: "1466275836220db2944ca059a3a10ef6fd2ea684b0688d2c379296888a206003",
	}

	// Checkpoints for testnet
	testnetCheckpoints = Checkpoints{
		0: "a771da8e52ee6ad581ed1e9a99825e5b3b7992225534eaa2ae23244fe26ab1c1",
	}

	// Checkpoints for regtest
	regtestCheckpoints = Checkpoints{
		0: "00902a6b70c2ca83b5d9c815d96a0e2f4202179316970d14ea1847dae5b1ca21",
	}
)

func GetCheckpoints(net Magic) Checkpoints {
	switch net {
	case MagicLiquid:
		return mainnetCheckpoints
	case MagicLiquidTestnet:
		return testnetCheckpoints
	case MagicRegtest:
		return regtestCheckpoints
	default:
		return regtestCheckpoints
	}
}
