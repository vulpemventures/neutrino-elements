package protocol

func NewSendHeadersMessage(network string) (*Message, error) {
	return NewMessage("sendheaders", network, []byte{})
}
