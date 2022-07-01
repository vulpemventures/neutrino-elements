package neutrinodtypes

import "github.com/vulpemventures/neutrino-elements/pkg/scanner"

const (
	Register   ActionType = "register"
	Unregister ActionType = "unregister"

	Unspents EventType = "UNSPENT"
)

type EventType string

type ActionType string

type WsMessageRequest struct {
	ActionType       ActionType
	EventTypes       []scanner.EventType `json:"eventTypes"`
	DescriptorWallet string              `json:"descriptorWallet"`
	StartBlockHeight int                 `json:"startBlockHeight"`
}

type WsOnChainEventResponse struct {
	EventType string `json:"eventType"`
	TxID      string `json:"txId"`
}

type WsGeneralMessageResponse struct {
	Message string `json:"message"`
}

type WsMessageErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
}
