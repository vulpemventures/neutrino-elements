package application

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/google/uuid"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/neutrino-elements/pkg/scanner"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type SubscriberID uuid.UUID

type Subscriber struct {
	ID               SubscriberID
	BlockHeight      int
	Events           []scanner.EventType
	WalletDescriptor string
}

func (s *Subscriber) validate() error {
	return validation.ValidateStruct(
		s,
		validation.Field(&s.ID, validation.Required),
		validation.Field(&s.BlockHeight, validation.Min(1)),
		validation.Field(&s.Events, validation.Required),
		validation.Field(&s.WalletDescriptor, validation.Required),
	)
}

type SubscriberEventReport struct {
	SubscriberID SubscriberID
	EventType    scanner.EventType
	BlockHeight  int
	Transaction  *transaction.Transaction
	BlockHash    *chainhash.Hash
}

type SubscriberErrorReport struct {
	SubscriberID SubscriberID
	ErrorMsg     error
}
