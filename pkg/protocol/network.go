package protocol

var (
	MagicMainnet         Magic = [magicLength]byte{0xf9, 0xbe, 0xb4, 0xd9}
	MagicSimnet          Magic = [magicLength]byte{0x16, 0x1c, 0x14, 0x12}
	MagicTestNetV3       Magic = [magicLength]byte{0x0b, 0x11, 0x09, 0x07}
	MagicElements        Magic = [magicLength]byte{0xfa, 0xbf, 0xb5, 0xda}
	MagicNigiri          Magic = [magicLength]byte{0x12, 0x34, 0x56, 0x78}
	MagicElementsTestnet Magic = [magicLength]byte{0x41, 0x0e, 0xdd, 0x62}
	Networks                   = map[string][magicLength]byte{
		"mainnet":          MagicMainnet,
		"simnet":           MagicSimnet,
		"testnet":          MagicTestNetV3,
		"elements":         MagicElements,
		"elements-testnet": MagicElementsTestnet,
		"nigiri":           MagicNigiri,
	}
)

type Magic [magicLength]byte
