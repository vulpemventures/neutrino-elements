package application

import (
	"errors"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
	"runtime/debug"
)

type NotificationService interface {
	Start() error
	Stop()
	Subscribe(subscriber Subscriber) error
	UnSubscribe(subscriber Subscriber) error
	EventReport() chan SubscriberEventReport
	ErrorReport() chan SubscriberErrorReport
}

type notificationService struct {
	// subscribers is a map of subscriber IDs and Subscribers with wallet descriptors
	subscribers map[SubscriberID]Subscriber
	// scannerSvc is the scanner service used to watch on-chain events
	scannerSvc scanner.Service

	// registerSubs is a channel used to register subscribers
	registerSubs chan Subscriber
	// unregisterSubs is a channel used to unregister subscribers
	unregisterSubs chan Subscriber

	// subsEventReport is a channel used to report events to subscribers
	subsEventReport chan SubscriberEventReport
	// subsErrorReport is a channel used to report errors to subscribers
	subsErrorReport chan SubscriberErrorReport

	// quitHandleOnChainEvents is a channel used to stop handleOnChainEvents
	quitHandleOnChainEvents chan struct{}
	// quitHandleOnChainEvents is a channel used to stop handleOnChainEvents
	quitHandleSubscribers chan struct{}
}

func NewNotificationService(
	scannerSvc scanner.Service,
) NotificationService {
	return &notificationService{
		subscribers:             make(map[SubscriberID]Subscriber),
		scannerSvc:              scannerSvc,
		registerSubs:            make(chan Subscriber),
		unregisterSubs:          make(chan Subscriber),
		subsEventReport:         make(chan SubscriberEventReport),
		subsErrorReport:         make(chan SubscriberErrorReport),
		quitHandleOnChainEvents: make(chan struct{}),
		quitHandleSubscribers:   make(chan struct{}),
	}
}

func (n *notificationService) Start() error {
	go n.handleSubscribers()

	scannerReport, err := n.scannerSvc.Start()
	if err != nil {
		return err
	}

	go n.handleOnChainEvents(scannerReport)

	log.Debug("notification-service started")
	return nil
}

func (n *notificationService) EventReport() chan SubscriberEventReport {
	return n.subsEventReport
}

func (n *notificationService) ErrorReport() chan SubscriberErrorReport {
	return n.subsErrorReport
}

func (n *notificationService) Stop() {
	log.Debug("sss")
	n.quitHandleOnChainEvents <- struct{}{}
	n.quitHandleSubscribers <- struct{}{}
	log.Debug("sssdsds")
}

func (n *notificationService) handleOnChainEvents(scannerReport <-chan scanner.Report) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("handleOnChainEvents recovered from panic: %v", err)
			log.Tracef("handleOnChainEvents recovered from panic: %v", string(debug.Stack()))
		}
	}()
	for {
		select {
		case report := <-scannerReport:
			n.subsEventReport <- SubscriberEventReport{
				SubscriberID: SubscriberID(report.Request.ClientID),
				EventType:    report.Request.Item.EventType(),
				BlockHeight:  int(report.BlockHeight),
				BlockHash:    report.BlockHash,
				Transaction:  report.Transaction,
			}
		case <-n.quitHandleOnChainEvents:
			log.Debug("notificationService -> handleOnChainEvents stopped")
			return
		}
	}
}

func (n *notificationService) handleSubscribers() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("handleSubscribers recovered from panic: %v", err)
			log.Tracef("handleSubscribers recovered from panic: %v", string(debug.Stack()))
		}
	}()

	for {
		select {
		case sub := <-n.registerSubs:
			n.subscribers[sub.ID] = sub

			if err := n.scannerSvc.WatchDescriptorWallet(
				uuid.UUID(sub.ID),
				sub.WalletDescriptor,
				sub.Events,
				sub.BlockHeight,
			); err != nil {
				n.subsErrorReport <- SubscriberErrorReport{
					SubscriberID: sub.ID,
					ErrorMsg:     err,
				}

				return
			}
		case sub := <-n.unregisterSubs:
			delete(n.subscribers, sub.ID)
		case <-n.quitHandleSubscribers:
			log.Debug("notificationService -> handleSubscribers stopped")
			return
		}
	}
}

func (n *notificationService) Subscribe(
	subscriber Subscriber,
) error {
	if err := subscriber.validate(); err != nil {
		return err
	}

	go func() {
		n.registerSubs <- subscriber
	}()

	return nil
}

func (n *notificationService) UnSubscribe(subscriber Subscriber) error {
	_, ok := n.subscribers[subscriber.ID]
	if !ok {
		return errors.New("subscriber not found")
	}

	go func() {
		n.unregisterSubs <- subscriber
	}()

	return nil
}
