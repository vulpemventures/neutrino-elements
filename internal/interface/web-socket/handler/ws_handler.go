package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/internal/core/application"
	neutrinodtypes "github.com/vulpemventures/neutrino-elements/pkg/neutrinod-types"
	"net/http"
	"runtime/debug"
	"time"
)

func (d *descriptorWalletNotifierHandler) HandleSubscriptionRequestWs(
	w http.ResponseWriter,
	r *http.Request,
) {
	wsUpgrader := websocket.Upgrader{}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("upgrading error: %#v\n", err)
		return
	}

	go d.handleRequest(conn)
}

func (d *descriptorWalletNotifierHandler) handleRequest(conn *websocket.Conn) {
	defer func() {
		conn.Close()
		if err := recover(); err != nil {
			log.Errorf("handleRequest recovered from panic: %v", err)
			log.Tracef("handleRequest recovered from panic: %v", string(debug.Stack()))
		}
	}()

	conn.SetReadLimit(maxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		if err := conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
			log.Warnf("Error writing close message: %#v\n", err)
		}

		log.Warnf(err.Error())
		return
	}

	conn.SetPongHandler(
		func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		},
	)

	subsID := uuid.New()
	d.registerSubs <- &WsSubscriber{
		ID:           SubscriberID(subsID),
		WsConnection: conn,
	}

	log.Debugf("new ws subscriber connected: %v", subsID)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if e, ok := err.(*websocket.CloseError); ok {
				if e.Code != websocket.CloseNormalClosure {
					log.Warnf(
						"Error reading message: %v, error code: %v\n",
						e.Text,
						e.Code,
					)
				}
			} else {
				log.Warnf("Error reading message: %v\n", err)
			}

			d.unregisterSubs <- &WsSubscriber{
				ID: SubscriberID(subsID),
			}

			return
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		wsMsg := &neutrinodtypes.SubscriptionRequestWs{}
		if err = json.Unmarshal(message, wsMsg); err != nil {
			log.Warn(err)
			return
		}

		log.Debugf("new message from subscriber: %v", subsID)

		subscriber := d.getSubscriberSafe(SubscriberID(subsID)).(*WsSubscriber)

		switch wsMsg.ActionType {
		case neutrinodtypes.Register:
			if err := d.notificationSvc.Subscribe(application.Subscriber{
				ID:               application.SubscriberID(subsID),
				BlockHeight:      wsMsg.StartBlockHeight,
				Events:           wsMsg.EventTypes,
				WalletDescriptor: wsMsg.DescriptorWallet,
			}); err != nil {
				log.Errorf("unsucesfull registration: %v, subscriber: %v", err, subsID)

				if err := sendResponseToSubscriberWs(*subscriber, neutrinodtypes.MessageErrorResponse{
					ErrorMessage: err.Error(),
				}); err != nil {
					log.Errorf("failed sending response to subscriber: %v", err.Error())
					continue
				}
			}
			log.Infof("sucesfull registration, subscriber: %v", subsID)

			if err := sendResponseToSubscriberWs(*subscriber, neutrinodtypes.GeneralMessageResponse{
				Message: "successfully registered",
			}); err != nil {
				log.Errorf("failed sending response to subscriber: %v", err.Error())
				continue
			}
		case neutrinodtypes.Unregister:
			if err := d.notificationSvc.UnSubscribe(application.Subscriber{
				ID: application.SubscriberID(subsID),
			}); err != nil {
				log.Errorf("unsucesfull un-registration: %v, subscriber: %v", err, subsID)

				if err := sendResponseToSubscriberWs(*subscriber, neutrinodtypes.MessageErrorResponse{
					ErrorMessage: err.Error(),
				}); err != nil {
					log.Errorf("failed sending response to subscriber: %v", err.Error())
					continue
				}
			}

			log.Infof("sucesfull un-registration: %v, subscriber: %v", err, subsID)

			if err := sendResponseToSubscriberWs(*subscriber, neutrinodtypes.GeneralMessageResponse{
				Message: "successfully un-registered",
			}); err != nil {
				log.Errorf("failed sending response to subscriber: %v", err.Error())
				continue
			}
		default:
			log.Errorf("unknown action type: %v\n", wsMsg.ActionType)
		}
	}
}

func sendResponseToSubscriberWs[
V neutrinodtypes.MessageErrorResponse |
neutrinodtypes.OnChainEventResponse |
neutrinodtypes.GeneralMessageResponse](
	subscriber WsSubscriber,
	resp V,
) error {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("sendResponseToSubscriberWs recovered from panic: %v", err)
			log.Tracef("sendResponseToSubscriberWs recovered from panic: %v", string(debug.Stack()))
		}
	}()

	r, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("sendResponseToSubscriberWs -> %v", err)
	}

	if err = subscriber.WsConnection.WriteMessage(websocket.TextMessage, r); err != nil {
		return fmt.Errorf("sendResponseToSubscriberWs -> %v", err)
	}

	return nil
}
