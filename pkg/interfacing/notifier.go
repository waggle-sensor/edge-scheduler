package interfacing

import (
	"sync"

	"github.com/waggle-sensor/edge-scheduler/pkg/datatype"
)

type Notifier struct {
	subscribers map[chan datatype.Event]struct{}
	rm          sync.RWMutex
}

func NewNotifier() *Notifier {
	return &Notifier{
		subscribers: map[chan datatype.Event]struct{}{},
	}
}

func (n *Notifier) Subscribe(sub chan datatype.Event) {
	n.subscribers[sub] = struct{}{}
}

func (n *Notifier) Notify(e datatype.Event) {
	n.rm.RLock()
	for s := range n.subscribers {
		s <- e
	}
	n.rm.RUnlock()
}
