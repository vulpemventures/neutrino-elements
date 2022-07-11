package neutrinodtypes

import "github.com/vulpemventures/neutrino-elements/pkg/scanner"

const (
	Register   ActionType = "register"
	Unregister ActionType = "unregister"

	Unspents EventType = "UNSPENT"
)

type EventType string

type ActionType string

type SubscriptionRequestWs struct {
	ActionType       ActionType          `json:"actionType"`
	EventTypes       []scanner.EventType `json:"eventTypes"`
	DescriptorWallet string              `json:"descriptorWallet"`
	StartBlockHeight int                 `json:"startBlockHeight"`
}

type OnChainEventResponse struct {
	EventType string `json:"eventType"`
	TxID      string `json:"txId"`
}

type GeneralMessageResponse struct {
	Message string `json:"message"`
}

type MessageErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
}
