package protocol

var (
	MagicLiquid        Magic = [magicLength]byte{0xfa, 0xbf, 0xb5, 0xda}
	MagicNigiri        Magic = [magicLength]byte{0x12, 0x34, 0x56, 0x78}
	MagicLiquidTestnet Magic = [magicLength]byte{0x41, 0x0e, 0xdd, 0x62}
	Networks                 = map[string][magicLength]byte{
		"liquid":         MagicLiquid,
		"liquid-testnet": MagicLiquidTestnet,
		"nigiri":         MagicNigiri,
	}
)

type Magic [magicLength]byte
