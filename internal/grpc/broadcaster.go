package grpc

import (
	"sync"
	"sync/atomic"

	"github.com/mr1hm/go-disaster-alerts/internal/models"
)

type Broadcaster struct {
	subscribers map[uint64]chan *models.Disaster
	nextID      atomic.Uint64
	mu          sync.RWMutex
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[uint64]chan *models.Disaster),
	}
}

func (b *Broadcaster) Subscribe() (uint64, chan *models.Disaster) {
	id := b.nextID.Add(1)
	ch := make(chan *models.Disaster, 100) // Buffer for max disasters per poll

	b.mu.Lock()
	b.subscribers[id] = ch
	b.mu.Unlock()

	return id, ch
}

func (b *Broadcaster) Unsubscribe(id uint64) {
	b.mu.Lock()
	if ch, ok := b.subscribers[id]; ok {
		close(ch)
		delete(b.subscribers, id)
	}
	b.mu.Unlock()
}

func (b *Broadcaster) Broadcast(d *models.Disaster) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- d:
		default:
			// Skip slow subscribers
		}
	}
}

func (b *Broadcaster) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// Close closes all subscriber channels, causing streams to exit gracefully
func (b *Broadcaster) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, ch := range b.subscribers {
		close(ch)
		delete(b.subscribers, id)
	}
}
