package protocol

func NewSendHeadersMessage(network Magic) (*Message, error) {
	return NewMessage("sendheaders", network, []byte{})
}
