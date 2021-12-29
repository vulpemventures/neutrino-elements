package protocol

type MsgGetCFilters struct {
	FilterType  byte
	StartHeight uint32
	StopHash    [32]byte
}
