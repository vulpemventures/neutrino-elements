package node

import (
	"errors"
	"github.com/vulpemventures/go-elements/block"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/neutrino-elements/pkg/protocol"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	TxConfirmed TxEventType = iota
	TxUnConfirmed
)

type MemPool struct {
	//txChan receives new transactions from the network
	txChan chan protocol.MsgTx
	//quitChan receives a quit signal
	quitChan chan struct{}

	//txMap is a map of transactions that are in the memPool
	txs map[string]txData
	//txsMutex is a mutex to protect the txs map
	txsMutex *sync.RWMutex

	//txSubscribersMutex is a mutex to protect the txs map
	txSubscribersMutex *sync.RWMutex
	//txSubscribers is a list of subscribers listening for new transactions
	txSubscribers []txSubscriber
}

func NewMemPool() MemPool {
	return MemPool{
		txChan:             make(chan protocol.MsgTx),
		txs:                make(map[string]txData),
		quitChan:           make(chan struct{}),
		txsMutex:           new(sync.RWMutex),
		txSubscribersMutex: new(sync.RWMutex),
	}
}

type TxEventType int

type TxEvent interface {
	Type() TxEventType
	TxID() string
}

type TxConfirmedEvent struct {
	txID string
	tx   transaction.Transaction
}

func (t TxConfirmedEvent) Type() TxEventType {
	return TxConfirmed
}

func (t TxConfirmedEvent) TxID() string {
	return t.txID
}

type TxUnConfirmedEvent struct {
	txID string
	tx   transaction.Transaction
}

func (t TxUnConfirmedEvent) Type() TxEventType {
	return TxUnConfirmed
}

func (t TxUnConfirmedEvent) TxID() string {
	return t.txID
}

type txData struct {
	tx           transaction.Transaction
	timeReceived time.Time
}

type txSubscriber struct {
	id      string
	txEvent chan<- TxEvent
}

func (m *MemPool) Start() {
	log.Debugln("mem-pool: mem-pool started")
	go m.listenForNewTxs()
}

func (m *MemPool) Stop() {
	close(m.txChan)
	close(m.quitChan)
	subs := m.getSubscribersSafe()
	for _, v := range subs {
		close(v.txEvent)
	}
}

func (m *MemPool) GetMemPool() map[string]transaction.Transaction {
	m.txsMutex.RLock()
	defer m.txsMutex.RUnlock()

	txMap := make(map[string]transaction.Transaction)
	for k, v := range m.txs {
		txMap[k] = v.tx
	}

	return txMap
}

func (m *MemPool) AddTx(tx protocol.MsgTx) {
	m.txChan <- tx
}

func (m *MemPool) AddSubscriber(id string) <-chan TxEvent {
	m.txSubscribersMutex.Lock()
	defer m.txSubscribersMutex.Unlock()
	txEvent := make(chan TxEvent)
	m.txSubscribers = append(m.txSubscribers, txSubscriber{
		id:      id,
		txEvent: txEvent,
	})

	return txEvent
}

func (m *MemPool) getSubscribersSafe() []txSubscriber {
	m.txSubscribersMutex.RLock()
	defer m.txSubscribersMutex.RUnlock()

	return m.txSubscribers
}

func (m *MemPool) CheckTxConfirmed(block block.Block) {
	go m.checkTxConfirmed(block)
}

func (m *MemPool) checkTxConfirmed(block block.Block) {
	//TODO: check if this is the best way to do this
	for _, v := range m.txs {
		for _, tx := range block.TransactionsData.Transactions {
			if tx.TxHash().String() == v.tx.TxHash().String() {
				if err := m.removeTxFromMemPool(v.tx.TxHash().String()); err != nil {
					log.Errorln("failed to remove tx from memPool")
				}

				m.notifySubscribers(TxConfirmedEvent{
					txID: tx.TxHash().String(),
					tx:   *tx,
				})
			}
		}
	}
}

func (m *MemPool) removeTxFromMemPool(txID string) error {
	m.txsMutex.Lock()
	defer m.txsMutex.Unlock()
	if _, ok := m.txs[txID]; ok {
		delete(m.txs, txID)
		return nil
	}

	return errors.New("tx not found")
}

func (m *MemPool) addTx(txID string, txData txData) {
	m.txsMutex.Lock()
	defer m.txsMutex.Unlock()

	if _, ok := m.txs[txID]; !ok {
		m.txs[txID] = txData
	}
}

func (m *MemPool) listenForNewTxs() {
	log.Debugln("mem-pool: listening for new transactions")

	for tx := range m.txChan {
		m.addTx(
			tx.HashStr(),
			txData{
				tx:           tx.Transaction,
				timeReceived: time.Now(),
			},
		)

		m.notifySubscribers(TxUnConfirmedEvent{
			txID: tx.HashStr(),
			tx:   tx.Transaction,
		})

		log.Debugf("tx %s added to memPool", tx.HashStr())
	}

	log.Debugln("mem-pool: memPool listener stopped")
}

func (m *MemPool) notifySubscribers(txEvent TxEvent) {
	subs := m.getSubscribersSafe()
	for _, v := range subs {
		go func(subscriber txSubscriber) {
			log.Debugf("notifying subscriber %s of new tx started", subscriber.id)
			subscriber.txEvent <- txEvent
			log.Debugf("notifying subscriber %s of new tx done", subscriber.id)
		}(v)
	}
}
