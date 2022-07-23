package handler

import (
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type SubscriberID uuid.UUID

type SubscriberType string

type Subscriber interface {
	Type() SubscriberType
	SubscriberID() SubscriberID
}

type WsSubscriber struct {
	ID           SubscriberID
	WsConnection *websocket.Conn
}

func (w *WsSubscriber) Type() SubscriberType {
	return wsType
}

func (w *WsSubscriber) SubscriberID() SubscriberID {
	return w.ID
}

type HttpSubscriber struct {
	ID          SubscriberID
	EndpointUrl string
}

func (h *HttpSubscriber) Type() SubscriberType {
	return httpType
}

func (h *HttpSubscriber) SubscriberID() SubscriberID {
	return h.ID
}
