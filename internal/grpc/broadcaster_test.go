package grpc

import (
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"

	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
	"github.com/mr1hm/go-disaster-alerts/internal/models"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestBroadcaster_SubscribeUnsubscribe(t *testing.T) {
	b := NewBroadcaster()

	id, ch := b.Subscribe()
	if b.SubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", b.SubscriberCount())
	}

	b.Unsubscribe(id)
	if b.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", b.SubscriberCount())
	}

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed")
		}
	default:
		t.Error("channel should be closed and readable")
	}
}

func TestBroadcaster_Broadcast(t *testing.T) {
	b := NewBroadcaster()

	id, ch := b.Subscribe()
	defer b.Unsubscribe(id)

	disaster := &models.Disaster{
		ID:        "test_1",
		Type:      disastersv1.DisasterType_EARTHQUAKE,
		Magnitude: 6.5,
	}

	b.Broadcast(disaster)

	select {
	case received := <-ch:
		if received.ID != disaster.ID {
			t.Errorf("expected ID %s, got %s", disaster.ID, received.ID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for broadcast")
	}
}

func TestBroadcaster_ConcurrentSubscribeUnsubscribe(t *testing.T) {
	b := NewBroadcaster()
	var wg sync.WaitGroup

	// Concurrently subscribe and unsubscribe
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, _ := b.Subscribe()
			time.Sleep(time.Millisecond)
			b.Unsubscribe(id)
		}()
	}

	wg.Wait()

	if b.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after cleanup, got %d", b.SubscriberCount())
	}
}

func TestBroadcaster_ConcurrentBroadcast(t *testing.T) {
	b := NewBroadcaster()
	var wg sync.WaitGroup

	// Create subscribers
	numSubscribers := 10
	channels := make([]chan *models.Disaster, numSubscribers)
	ids := make([]uint64, numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		ids[i], channels[i] = b.Subscribe()
	}

	// Concurrently broadcast
	numBroadcasts := 50
	for i := 0; i < numBroadcasts; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			b.Broadcast(&models.Disaster{
				ID:        "test_" + string(rune(n)),
				Magnitude: float64(n),
			})
		}(i)
	}

	wg.Wait()

	// Cleanup
	for i := 0; i < numSubscribers; i++ {
		b.Unsubscribe(ids[i])
	}
}

func TestBroadcaster_ConcurrentSubscribeBroadcast(t *testing.T) {
	b := NewBroadcaster()
	var wg sync.WaitGroup

	// Concurrent subscribers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, ch := b.Subscribe()
			// Drain channel to prevent blocking
			go func() {
				for range ch {
				}
			}()
			time.Sleep(5 * time.Millisecond)
			b.Unsubscribe(id)
		}()
	}

	// Concurrent broadcasts
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			b.Broadcast(&models.Disaster{
				ID:        "broadcast_test",
				Magnitude: float64(n),
			})
		}(i)
	}

	wg.Wait()

	if b.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers, got %d", b.SubscriberCount())
	}
}

func TestBroadcaster_Close(t *testing.T) {
	b := NewBroadcaster()

	// Create multiple subscribers
	var channels []chan *models.Disaster
	for i := 0; i < 5; i++ {
		_, ch := b.Subscribe()
		channels = append(channels, ch)
	}

	if b.SubscriberCount() != 5 {
		t.Errorf("expected 5 subscribers, got %d", b.SubscriberCount())
	}

	b.Close()

	if b.SubscriberCount() != 0 {
		t.Errorf("expected 0 subscribers after close, got %d", b.SubscriberCount())
	}

	// All channels should be closed
	for i, ch := range channels {
		select {
		case _, ok := <-ch:
			if ok {
				t.Errorf("channel %d should be closed", i)
			}
		default:
			t.Errorf("channel %d should be closed and readable", i)
		}
	}
}

func TestBroadcaster_SlowSubscriber(t *testing.T) {
	b := NewBroadcaster()

	id, ch := b.Subscribe()
	defer b.Unsubscribe(id)

	// Fill the buffer (100) + 1 more
	for i := 0; i < 101; i++ {
		b.Broadcast(&models.Disaster{
			ID:        "flood_test",
			Magnitude: float64(i),
		})
	}

	// Should not block - the 101st message is dropped
	// Drain what we can
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:

	if count != 100 {
		t.Errorf("expected 100 buffered messages, got %d", count)
	}
}
