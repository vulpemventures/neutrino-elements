package handler

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
)

const (
	register   = "register"
	unregister = "unregister"
)

type EventType string

type WsMessageRequest struct {
	ActionType       string
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

type SubscriberID uuid.UUID

type Subscriber struct {
	ID           SubscriberID
	WsConnection *websocket.Conn
}
