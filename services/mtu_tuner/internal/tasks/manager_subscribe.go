package tasks

import (
	"errors"
	"sync"

	"mtu-tuner/internal/core"
)

const DefaultSubscriptionBuffer = 64

var ErrSubscriptionOverflow = errors.New("task event subscriber fell behind")

type Subscription struct {
	manager *Manager
	events  chan core.TaskEvent

	closeOnce sync.Once
	errMu     sync.Mutex
	err       error
}

func (manager *Manager) Subscribe(buffer int) *Subscription {
	if buffer < 1 {
		buffer = DefaultSubscriptionBuffer
	}
	subscription := &Subscription{
		manager: manager,
		events:  make(chan core.TaskEvent, buffer),
	}

	manager.mu.Lock()
	manager.subscribers[subscription] = struct{}{}
	manager.mu.Unlock()
	return subscription
}

func (subscription *Subscription) Events() <-chan core.TaskEvent {
	if subscription == nil {
		return nil
	}
	return subscription.events
}

func (subscription *Subscription) Err() error {
	if subscription == nil {
		return nil
	}
	subscription.errMu.Lock()
	defer subscription.errMu.Unlock()
	return subscription.err
}

func (subscription *Subscription) Close() {
	if subscription == nil {
		return
	}
	if subscription.manager != nil {
		subscription.manager.removeSubscription(subscription, nil)
		return
	}
	subscription.finish(nil)
}

func (manager *Manager) broadcast(event core.TaskEvent) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	for subscription := range manager.subscribers {
		select {
		case subscription.events <- event:
		default:
			delete(manager.subscribers, subscription)
			subscription.finish(ErrSubscriptionOverflow)
		}
	}
}

func (manager *Manager) removeSubscription(subscription *Subscription, err error) {
	manager.mu.Lock()
	delete(manager.subscribers, subscription)
	manager.mu.Unlock()
	subscription.finish(err)
}

func (subscription *Subscription) finish(err error) {
	subscription.closeOnce.Do(func() {
		subscription.errMu.Lock()
		subscription.err = err
		subscription.errMu.Unlock()
		close(subscription.events)
	})
}
