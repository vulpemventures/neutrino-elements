package handler

import (
	"bytes"
	"encoding/json"
	"github.com/google/uuid"
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

	wsType   SubscriberType = "ws"
	httpType SubscriberType = "http"
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
	HandleSubscriptionRequestWs(w http.ResponseWriter, req *http.Request)
	HandleSubscriptionRequestHttp(w http.ResponseWriter, req *http.Request)
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

func (d *descriptorWalletNotifierHandler) handleSubscribers() {
	for {
		select {
		case sub := <-d.registerSubs:
			log.Debugf("subscriber: %v, registrated", uuid.UUID(sub.SubscriberID()).String())
			d.addSubscriberSafe(sub)

		case sub := <-d.unregisterSubs:
			log.Infof("subscriber: %v, un-registrated", uuid.UUID(sub.SubscriberID()).String())
			d.deleteSubscriberSafe(sub.SubscriberID())
			if err := d.notificationSvc.UnSubscribe(application.Subscriber{
				ID: application.SubscriberID(sub.SubscriberID()),
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

			response := neutrinodtypes.OnChainEventResponse{
				EventType: string(neutrinodtypes.Unspents),
				TxID:      eventReport.Transaction.TxHash().String(),
			}

			switch subscriber.Type() {
			case wsType:
				wsSubscriber := subscriber.(*WsSubscriber)
				if err := sendResponseToSubscriberWs(*wsSubscriber, response); err != nil {
					log.Errorf("failed sending response to subscriber: %v", err.Error())
					continue
				}
			case httpType:
				httpSubscriber := subscriber.(*HttpSubscriber)
				go func() {
					respJson, err := json.Marshal(response)
					if err != nil {
						log.Errorf("handleOnChainNotifications -> %v", err)
					}

					resp, err := http.Post(
						httpSubscriber.EndpointUrl,
						"application/json",
						bytes.NewBuffer(respJson),
					)
					if err != nil {
						log.Errorf("failed sending response to subscriber: %v", err.Error())
						return
					}
					defer resp.Body.Close()
				}()
			}

		case errReport := <-d.notificationSvc.ErrorReport():
			log.Errorf(
				"error: %v occured while observing chain for subscriber: %v\n",
				errReport.ErrorMsg,
				errReport.SubscriberID,
			)

			subscriber := d.getSubscriberSafe(SubscriberID(errReport.SubscriberID))

			response := neutrinodtypes.MessageErrorResponse{
				ErrorMessage: errReport.ErrorMsg.Error(),
			}

			switch subscriber.Type() {
			case wsType:
				wsSubscriber := subscriber.(*WsSubscriber)
				if err := sendResponseToSubscriberWs(*wsSubscriber, response); err != nil {
					log.Errorf("failed sending response to subscriber: %v", err.Error())
					continue
				}
			case httpType:
				httpSubscriber := subscriber.(*HttpSubscriber)
				go func() {
					respJson, err := json.Marshal(response)
					if err != nil {
						log.Errorf("handleOnChainNotifications -> %v", err)
					}

					resp, err := http.Post(
						httpSubscriber.EndpointUrl,
						"application/json",
						bytes.NewBuffer(respJson),
					)
					if err != nil {
						log.Errorf("failed sending response to subscriber: %v", err.Error())
						return
					}
					defer resp.Body.Close()
				}()
			}

		case <-d.quitHandleOnChainNotifications:
			log.Debugf("descriptorWalletNotifierHandler -> handleOnChainNotifications stopped")
			return
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

	d.subscribers[subscriber.SubscriberID()] = subscriber
}

func (d *descriptorWalletNotifierHandler) deleteSubscriberSafe(subscriberID SubscriberID) {
	d.subscribersLock.Lock()
	defer d.subscribersLock.Unlock()

	delete(d.subscribers, subscriberID)
}
