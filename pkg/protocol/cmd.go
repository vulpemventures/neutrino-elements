package protocol

import "github.com/sirupsen/logrus"

const (
	cmdBlock       = "block"
	cmdGetData     = "getdata"
	cmdInv         = "inv"
	cmdPing        = "ping"
	cmdPong        = "pong"
	cmdTx          = "tx"
	cmdVerack      = "verack"
	cmdVersion     = "version"
	cmdAddr        = "addr"
	cmdNotFound    = "notfound"
	cmdGetBlocks   = "getblocks"
	cmdGetHeaders  = "getheaders"
	cmdHeaders     = "headers"
	cmdGetAddr     = "getaddr"
	cmdMempool     = "mempool"
	cmdCheckOrder  = "checkorder"
	cmdSubmitOrder = "submitorder"
	cmdReply       = "reply"
	cmdReject      = "reject"
	cmdFilterLoad  = "filterload"
	cmdFilterAdd   = "filteradd"
	cmdFilterClear = "filterclear"
	cmdMerkleBlock = "merkleblock"
	cmdAlert       = "alert"
	cmdSendHeaders = "sendheaders"
	cmdFeeFilter   = "feefilter"
	cmdSendCmpct   = "sendcmpct"
	cmdCmpctBlock  = "cmpctblock"
	cmdGetBlockTxn = "getblocktxn"
	cmdBlockTxn    = "blocktxn"
	commandLength  = 12
)

var commands = map[string][commandLength]byte{
	cmdBlock:       newCommand(cmdBlock),
	cmdGetData:     newCommand(cmdGetData),
	cmdInv:         newCommand(cmdInv),
	cmdPing:        newCommand(cmdPing),
	cmdPong:        newCommand(cmdPong),
	cmdTx:          newCommand(cmdTx),
	cmdVerack:      newCommand(cmdVerack),
	cmdVersion:     newCommand(cmdVersion),
	cmdAddr:        newCommand(cmdAddr),
	cmdNotFound:    newCommand(cmdNotFound),
	cmdGetBlocks:   newCommand(cmdGetBlocks),
	cmdGetHeaders:  newCommand(cmdGetHeaders),
	cmdHeaders:     newCommand(cmdHeaders),
	cmdGetAddr:     newCommand(cmdGetAddr),
	cmdMempool:     newCommand(cmdMempool),
	cmdCheckOrder:  newCommand(cmdCheckOrder),
	cmdSubmitOrder: newCommand(cmdSubmitOrder),
	cmdReply:       newCommand(cmdReply),
	cmdReject:      newCommand(cmdReject),
	cmdFilterLoad:  newCommand(cmdFilterLoad),
	cmdFilterAdd:   newCommand(cmdFilterAdd),
	cmdFilterClear: newCommand(cmdFilterClear),
	cmdMerkleBlock: newCommand(cmdMerkleBlock),
	cmdAlert:       newCommand(cmdAlert),
	cmdSendHeaders: newCommand(cmdSendHeaders),
	cmdFeeFilter:   newCommand(cmdFeeFilter),
	cmdSendCmpct:   newCommand(cmdSendCmpct),
	cmdCmpctBlock:  newCommand(cmdCmpctBlock),
	cmdGetBlockTxn: newCommand(cmdGetBlockTxn),
	cmdBlockTxn:    newCommand(cmdBlockTxn),
}

func newCommand(command string) [commandLength]byte {
	l := len(command)
	if l > commandLength {
		logrus.Fatalf("command %s is too long", command)
	}

	var packed [commandLength]byte
	buf := make([]byte, commandLength-l)
	copy(packed[:], append([]byte(command), buf...)[:])

	return packed
}
