package application

import (
	"errors"
	"github.com/google/uuid"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"
)

type NotificationService interface {
	Start() error
	Stop() error
	Subscribe(subscriber Subscriber) error
	UnSubscribe(subscriber Subscriber) error
	EventReport() chan SubscriberEventReport
	ErrorReport() chan SubscriberErrorReport
}

type notificationService struct {
	subscribers map[SubscriberID]Subscriber
	scannerSvc  scanner.Service

	registerSubs   chan Subscriber
	unregisterSubs chan Subscriber

	subsEventReport chan SubscriberEventReport
	subsErrorReport chan SubscriberErrorReport
}

func NewNotificationService(
	scannerSvc scanner.Service,
) NotificationService {
	return &notificationService{
		subscribers:     make(map[SubscriberID]Subscriber),
		scannerSvc:      scannerSvc,
		registerSubs:    make(chan Subscriber),
		unregisterSubs:  make(chan Subscriber),
		subsEventReport: make(chan SubscriberEventReport),
		subsErrorReport: make(chan SubscriberErrorReport),
	}
}

func (n *notificationService) Start() error {
	go n.handleSubscribers()

	scannerReport, err := n.scannerSvc.Start()
	if err != nil {
		return err
	}

	go n.handleOnChainEvents(scannerReport)

	return nil
}

func (n *notificationService) EventReport() chan SubscriberEventReport {
	return n.subsEventReport
}

func (n *notificationService) ErrorReport() chan SubscriberErrorReport {
	return n.subsErrorReport
}

func (n *notificationService) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (n *notificationService) handleOnChainEvents(scannerReport <-chan scanner.Report) {
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
		}

		//TODO quit on signal
	}
}

func (n *notificationService) handleSubscribers() {
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
		}

		//TODO quit on signal
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
