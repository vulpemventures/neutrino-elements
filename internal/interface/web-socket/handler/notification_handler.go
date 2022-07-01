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
	"sync"
	"time"
)

const (
	pongWait       = 60 * time.Second
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

type descriptorWalletNotifierHandler struct {
	notificationSvc application.NotificationService

	// subscribers is a map of subscribers with their IDs as keys and ws conn
	subscribers map[SubscriberID]Subscriber
	// subscribersLock is a mutex for subscribers map
	subscribersLock *sync.RWMutex
	// registerSubs is a channel for registering subscribers
	registerSubs chan Subscriber
	// unregisterSubs is a channel for unregistering subscribers
	unregisterSubs chan Subscriber

	// quit is a channel for stopping handleSubscribers
	quitHandleSubscribers chan struct{}
	// quit is a channel for stopping handleSubscribers
	quitHandleOnChainNotifications chan struct{}
}

type DescriptorWalletNotifierHandler interface {
	Start()
	Stop()
	HandleSubscriptionRequest(w http.ResponseWriter, req *http.Request)
}

func NewDescriptorWalletNotifierHandler(
	notificationSvc application.NotificationService,
) DescriptorWalletNotifierHandler {
	return &descriptorWalletNotifierHandler{
		notificationSvc:                notificationSvc,
		subscribers:                    make(map[SubscriberID]Subscriber),
		subscribersLock:                new(sync.RWMutex),
		registerSubs:                   make(chan Subscriber),
		unregisterSubs:                 make(chan Subscriber),
		quitHandleSubscribers:          make(chan struct{}),
		quitHandleOnChainNotifications: make(chan struct{}),
	}
}

func (d *descriptorWalletNotifierHandler) Start() {
	go d.handleOnChainNotifications()
	go d.handleSubscribers()
}

func (d *descriptorWalletNotifierHandler) Stop() {
	d.quitHandleSubscribers <- struct{}{}
	d.quitHandleOnChainNotifications <- struct{}{}
	time.Sleep(time.Second) // wait for DescriptorWalletNotifierHandler to stop

	d.notificationSvc.Stop()
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

	go d.handleRequest(conn)
}

func (d *descriptorWalletNotifierHandler) handleSubscribers() {
	for {
		select {
		case sub := <-d.registerSubs:
			log.Debugf("subscriber: %v, registrated", uuid.UUID(sub.ID).String())
			d.addSubscriberSafe(sub)

		case sub := <-d.unregisterSubs:
			log.Infof("subscriber: %v, un-registrated", uuid.UUID(sub.ID).String())
			d.deleteSubscriberSafe(sub.ID)
			if err := d.notificationSvc.UnSubscribe(application.Subscriber{
				ID: application.SubscriberID(sub.ID),
			}); err != nil {
				log.Errorf("failed unsubscribing from chain: %v", err.Error())
			}

		case <-d.quitHandleSubscribers:
			log.Debugf("descriptorWalletNotifierHandler -> handleSubscribers stopped")
			return
		}
	}
}

func (d *descriptorWalletNotifierHandler) handleOnChainNotifications() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("handleOnChainNotifications recovered from panic: %v", err)
			log.Tracef("handleOnChainNotifications recovered from panic: %v", string(debug.Stack()))
		}
	}()
	for {
		select {
		case eventReport := <-d.notificationSvc.EventReport():
			subscriber := d.getSubscriberSafe(SubscriberID(eventReport.SubscriberID))

			if err := sendReplyToSubscriber(subscriber, neutrinodtypes.WsOnChainEventResponse{
				EventType: string(neutrinodtypes.Unspents),
				TxID:      eventReport.Transaction.TxHash().String(),
			}); err != nil {
				log.Errorf("failed sending response to subscriber: %v", err.Error())
				continue
			}

		case errReport := <-d.notificationSvc.ErrorReport():
			log.Errorf(
				"error: %v occured while observing chain for subscriber: %v\n",
				errReport.ErrorMsg,
				errReport.SubscriberID,
			)

			subscriber := d.getSubscriberSafe(SubscriberID(errReport.SubscriberID))

			if err := sendReplyToSubscriber(subscriber, neutrinodtypes.WsMessageErrorResponse{
				ErrorMessage: errReport.ErrorMsg.Error(),
			}); err != nil {
				log.Errorf("failed sending response to subscriber: %v", err.Error())
				continue
			}

		case <-d.quitHandleOnChainNotifications:
			log.Debugf("descriptorWalletNotifierHandler -> handleOnChainNotifications stopped")
			return
		}
	}
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
					log.Warnf(
						"Error reading message: %v, error code: %v\n",
						e.Text,
						e.Code,
					)
				}
			} else {
				log.Warnf("Error reading message: %v\n", err)
			}

			d.unregisterSubs <- Subscriber{
				ID: SubscriberID(subsID),
			}

			return
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		wsMsg := &neutrinodtypes.WsMessageRequest{}
		if err = json.Unmarshal(message, wsMsg); err != nil {
			log.Warn(err)
			return
		}

		log.Debugf("new message from subscriber: %v", subsID)

		subscriber := d.getSubscriberSafe(SubscriberID(subsID))

		switch wsMsg.ActionType {
		case neutrinodtypes.Register:
			if err := d.notificationSvc.Subscribe(application.Subscriber{
				ID:               application.SubscriberID(subsID),
				BlockHeight:      wsMsg.StartBlockHeight,
				Events:           wsMsg.EventTypes,
				WalletDescriptor: wsMsg.DescriptorWallet,
			}); err != nil {
				log.Errorf("unsucesfull registration: %v, subscriber: %v", err, subsID)

				if err := sendReplyToSubscriber(subscriber, neutrinodtypes.WsMessageErrorResponse{
					ErrorMessage: err.Error(),
				}); err != nil {
					log.Errorf("failed sending response to subscriber: %v", err.Error())
					continue
				}
			}
			log.Infof("sucesfull registration, subscriber: %v", subsID)

			if err := sendReplyToSubscriber(subscriber, neutrinodtypes.WsGeneralMessageResponse{
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

				if err := sendReplyToSubscriber(subscriber, neutrinodtypes.WsMessageErrorResponse{
					ErrorMessage: err.Error(),
				}); err != nil {
					log.Errorf("failed sending response to subscriber: %v", err.Error())
					continue
				}
			}

			log.Infof("sucesfull un-registration: %v, subscriber: %v", err, subsID)

			if err := sendReplyToSubscriber(subscriber, neutrinodtypes.WsGeneralMessageResponse{
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

func sendReplyToSubscriber[
	V neutrinodtypes.WsMessageErrorResponse |
		neutrinodtypes.WsOnChainEventResponse |
		neutrinodtypes.WsGeneralMessageResponse](
	subscriber Subscriber,
	resp V,
) error {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("sendReplyToSubscriber recovered from panic: %v", err)
			log.Tracef("sendReplyToSubscriber recovered from panic: %v", string(debug.Stack()))
		}
	}()

	r, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("sendReplyToSubscriber -> %v", err)
	}

	if err = subscriber.WsConnection.WriteMessage(websocket.TextMessage, r); err != nil {
		return fmt.Errorf("sendReplyToSubscriber -> %v", err)
	}

	return nil
}

type SubscriberID uuid.UUID

type Subscriber struct {
	ID           SubscriberID
	WsConnection *websocket.Conn
}
