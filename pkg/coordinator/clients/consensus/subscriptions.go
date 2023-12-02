package consensus

import "sync"

type Subscription[T interface{}] struct {
	Handler    func(evt T) error
	dispatcher *Dispatcher[T]
}

type Dispatcher[T interface{}] struct {
	mutex         sync.Mutex
	subscriptions []*Subscription[T]
}

func (d *Dispatcher[T]) Subscribe(subscription *Subscription[T]) *Subscription[T] {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if subscription.dispatcher != nil {
		return nil
	}
	d.subscriptions = append(d.subscriptions, subscription)
	subscription.dispatcher = d
	return subscription
}

func (s *Subscription[T]) Unsubscribe() {
	s.dispatcher.Unsubscribe(s)
}

func (d *Dispatcher[T]) Unsubscribe(subscription *Subscription[T]) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	count := len(d.subscriptions)
	for i, s := range d.subscriptions {
		if s == subscription {
			if i < count-1 {
				d.subscriptions[i] = d.subscriptions[count-1]
			}
			d.subscriptions = d.subscriptions[:count-1]
			subscription.dispatcher = nil
			return
		}
	}
}

func (d *Dispatcher[T]) Fire(data T) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	for _, s := range d.subscriptions {
		err := s.Handler(data)
		if err != nil {
			return err
		}
	}
	return nil
}
