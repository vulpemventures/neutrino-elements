package neutrinodtypes

import (
	"errors"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
)

var (
	ErrInvalidEventType = errors.New("invalid event type")
)

const (
	Register   ActionType = "register"
	Unregister ActionType = "unregister"

	UnspentUtxo EventType = "unspentUtxo"
	SpentUtxo   EventType = "spentUtxo"
)

type EventType string

type ActionType string

type SubscriptionRequestWs struct {
	ActionType       ActionType  `json:"actionType"`
	EventTypes       []EventType `json:"eventTypes"`
	DescriptorWallet string      `json:"descriptorWallet"`
	StartBlockHeight int         `json:"startBlockHeight"`
}

type OnChainEventResponse struct {
	EventType EventType `json:"eventType"`
	TxID      string    `json:"txId"`
}

type GeneralMessageResponse struct {
	Message string `json:"message"`
}

type MessageErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
}

func FromScannerEventTypeToNeutrinodType(eventType scanner.EventType) (EventType, error) {
	switch eventType {
	case scanner.UnspentUtxo:
		return UnspentUtxo, nil
	case scanner.SpentUtxo:
		return SpentUtxo, nil
	default:
		return "", ErrInvalidEventType
	}
}

func FromNeutrinodTypeToScannerEventType(eventType EventType) (scanner.EventType, error) {
	switch eventType {
	case UnspentUtxo:
		return scanner.UnspentUtxo, nil
	case SpentUtxo:
		return scanner.SpentUtxo, nil
	default:
		return 0, ErrInvalidEventType
	}
}
