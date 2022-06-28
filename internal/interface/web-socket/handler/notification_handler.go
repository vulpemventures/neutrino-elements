package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/internal/core/application"
	"net/http"
	"sync"
	"time"
)

const (
	pongWait       = 60 * time.Second
	maxMessageSize = 512

	unspents EventType = "UNSPENT"
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type descriptorWalletNotifierHandler struct {
	notificationSvc application.NotificationService

	subscribers     map[SubscriberID]Subscriber
	subscribersLock *sync.RWMutex
	registerSubs    chan Subscriber
	unregisterSubs  chan Subscriber

	internalQuit chan struct{}
	externalQuit chan struct{}
}

type DescriptorWalletNotifierService interface {
	HandleSubscriptionRequest(w http.ResponseWriter, req *http.Request)
	Stop()
}

func NewDescriptorWalletNotifierService(
	notificationSvc application.NotificationService,
) DescriptorWalletNotifierService {
	return &descriptorWalletNotifierHandler{
		notificationSvc: notificationSvc,
		subscribers:     make(map[SubscriberID]Subscriber),
		subscribersLock: new(sync.RWMutex),
		registerSubs:    make(chan Subscriber),
		unregisterSubs:  make(chan Subscriber),
		internalQuit:    make(chan struct{}),
		externalQuit:    make(chan struct{}),
	}
}

func (d *descriptorWalletNotifierHandler) HandleSubscriptionRequest(
	w http.ResponseWriter,
	r *http.Request,
) {
	wsUpgrader := websocket.Upgrader{}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("upgrading error: %#v\n", err)
		return
	}

	go d.handleOnChainNotifications()
	go d.handleSubscribers()
	go d.handleRequest(conn)
}

func (d *descriptorWalletNotifierHandler) Stop() {
	d.externalQuit <- struct{}{}
}

func (d *descriptorWalletNotifierHandler) handleSubscribers() {
	for {
		select {
		case sub := <-d.registerSubs:
			log.Debugf("subscriber: %v, registrated", uuid.UUID(sub.ID).String())
			d.addSubscriberSafe(sub)

		case sub := <-d.unregisterSubs:
			log.Debugf("subscriber: %v, un-registrated", uuid.UUID(sub.ID).String())
			d.deleteSubscriberSafe(sub.ID)

		case <-d.internalQuit:
			log.Debugf("handleSubscribers stopped internally")
			return

		case <-d.externalQuit:
			log.Debugf("handleSubscribers stopped externally")
			return
		}
	}
}

func (d *descriptorWalletNotifierHandler) handleOnChainNotifications() {
	for {
		select {
		case eventReport := <-d.notificationSvc.EventReport():
			subscriber := d.getSubscriberSafe(SubscriberID(eventReport.SubscriberID))

			if err := sendReplyToSubscriber(subscriber, WsOnChainEventResponse{
				EventType: string(unspents),
				TxID:      eventReport.Transaction.TxHash().String(),
			}); err != nil {
				log.Error(err)
				continue
			}

		case errReport := <-d.notificationSvc.ErrorReport():
			log.Errorf(
				"error: %v occured while observing chain for subscriber: %v\n",
				errReport.ErrorMsg,
				errReport.SubscriberID,
			)

			subscriber := d.getSubscriberSafe(SubscriberID(errReport.SubscriberID))

			if err := sendReplyToSubscriber(subscriber, WsMessageErrorResponse{
				ErrorMessage: errReport.ErrorMsg.Error(),
			}); err != nil {
				log.Error(err)
				continue
			}

		case <-d.internalQuit:
			log.Debugf("handleOnChainNotifications stopped internally")
			return

		case <-d.externalQuit:
			log.Debugf("handleOnChainNotifications stopped externally")
			return
		}
	}
}

func (d *descriptorWalletNotifierHandler) handleRequest(conn *websocket.Conn) {
	defer func() {
		conn.Close()
	}()

	conn.SetReadLimit(maxMessageSize)
	if err := conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		if err := conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
			log.Errorf("Error writing close message: %#v\n", err)
		}

		log.Error(err)
		return
	}

	conn.SetPongHandler(
		func(string) error {
			return conn.SetReadDeadline(time.Now().Add(pongWait))
		},
	)

	subsID := uuid.New()
	d.registerSubs <- Subscriber{
		ID:           SubscriberID(subsID),
		WsConnection: conn,
	}

	log.Debugf("new subscriber connected: %v", subsID)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if e, ok := err.(*websocket.CloseError); ok {
				if e.Code != websocket.CloseNormalClosure {
					log.Errorf(
						"Error reading message: %v, error code: %v\n",
						e.Text,
						e.Code,
					)
				}
			} else {
				log.Errorf("Error reading message: %v\n", err)
			}

			d.unregisterSubs <- Subscriber{
				ID: SubscriberID(subsID),
			}

			return
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		wsMsg := &WsMessageRequest{}
		if err = json.Unmarshal(message, wsMsg); err != nil {
			log.Error(err)
			return
		}

		log.Debugf("new message from subscriber: %v", subsID)

		subscriber := d.getSubscriberSafe(SubscriberID(subsID))

		switch wsMsg.ActionType {
		case register:
			if err := d.notificationSvc.Subscribe(application.Subscriber{
				ID:               application.SubscriberID(subsID),
				BlockHeight:      wsMsg.StartBlockHeight,
				Events:           wsMsg.EventTypes,
				WalletDescriptor: wsMsg.DescriptorWallet,
			}); err != nil {
				log.Errorf("unsucesfull registration: %v, subscriber: %v", err, subsID)

				if err := sendReplyToSubscriber(subscriber, WsMessageErrorResponse{
					ErrorMessage: err.Error(),
				}); err != nil {
					log.Error(err)
					continue
				}
			}
			log.Infof("sucesfull registration, subscriber: %v", subsID)

			if err := sendReplyToSubscriber(subscriber, WsGeneralMessageResponse{
				Message: "successfully registered",
			}); err != nil {
				log.Error(err)
				continue
			}
		case unregister:
			if err := d.notificationSvc.UnSubscribe(application.Subscriber{
				ID: application.SubscriberID(subsID),
			}); err != nil {
				log.Errorf("unsucesfull un-registration: %v, subscriber: %v", err, subsID)

				if err := sendReplyToSubscriber(subscriber, WsMessageErrorResponse{
					ErrorMessage: err.Error(),
				}); err != nil {
					log.Error(err)
					continue
				}
			}

			log.Infof("sucesfull un-registration: %v, subscriber: %v", err, subsID)

			if err := sendReplyToSubscriber(subscriber, WsGeneralMessageResponse{
				Message: "successfully un-registered",
			}); err != nil {
				log.Error(err)
				continue
			}
		default:
			log.Errorf("unknown action type: %v\n", wsMsg.ActionType)
		}

	}
}

func (d *descriptorWalletNotifierHandler) getSubscriberSafe(subscriberID SubscriberID) Subscriber {
	d.subscribersLock.RLock()
	defer d.subscribersLock.RUnlock()

	return d.subscribers[subscriberID]
}

func (d *descriptorWalletNotifierHandler) addSubscriberSafe(subscriber Subscriber) {
	d.subscribersLock.Lock()
	defer d.subscribersLock.Unlock()

	d.subscribers[subscriber.ID] = subscriber
}

func (d *descriptorWalletNotifierHandler) deleteSubscriberSafe(subscriberID SubscriberID) {
	d.subscribersLock.Lock()
	defer d.subscribersLock.Unlock()

	delete(d.subscribers, subscriberID)
}

func sendReplyToSubscriber[V WsMessageErrorResponse | WsOnChainEventResponse | WsGeneralMessageResponse](
	subscriber Subscriber,
	resp V,
) error {
	r, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("sendReplyToSubscriber -> %v", err)
	}

	if err = subscriber.WsConnection.WriteMessage(websocket.TextMessage, r); err != nil {
		return fmt.Errorf("sendReplyToSubscriber -> %v", err)
	}

	return nil
}
