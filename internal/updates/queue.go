package updates

import (
	"context"
	"sync"
	"time"

	"github.com/skrashevich/telegram-mock-ai/internal/models"
)

// Queue is a per-bot FIFO queue of updates with long-polling support.
type Queue struct {
	mu     sync.Mutex
	items  []models.Update
	nextID int64
	signal chan struct{}
}

// NewQueue creates a new update queue.
func NewQueue() *Queue {
	return &Queue{
		signal: make(chan struct{}, 1),
		nextID: 1,
	}
}

const maxQueueSize = 10000

// Enqueue adds an update to the queue, assigning an update_id.
// If the queue exceeds maxQueueSize, the oldest update is dropped.
func (q *Queue) Enqueue(update models.Update) {
	q.mu.Lock()
	update.UpdateID = q.nextID
	q.nextID++
	if len(q.items) >= maxQueueSize {
		q.items = q.items[1:]
	}
	q.items = append(q.items, update)
	q.mu.Unlock()

	// Signal waiting long-poll consumers
	select {
	case q.signal <- struct{}{}:
	default:
	}
}

// PendingCount returns the number of pending updates.
func (q *Queue) PendingCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// Dequeue returns updates with update_id >= offset.
// If no updates are available and timeout > 0, blocks until updates arrive or timeout expires.
func (q *Queue) Dequeue(ctx context.Context, offset int64, limit int, timeout time.Duration) []models.Update {
	if limit <= 0 {
		limit = 100
	}

	// Try immediate fetch
	result := q.fetch(offset, limit)
	if len(result) > 0 || timeout <= 0 {
		return result
	}

	// Long poll: wait for signal or timeout
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-q.signal:
			result = q.fetch(offset, limit)
			if len(result) > 0 {
				return result
			}
			// Spurious wake, keep waiting
		case <-timer.C:
			return q.fetch(offset, limit)
		case <-ctx.Done():
			return nil
		}
	}
}

// Confirm removes all updates with update_id < offset (confirmed by the bot).
func (q *Queue) Confirm(offset int64) {
	q.mu.Lock()
	defer q.mu.Unlock()
	i := 0
	for i < len(q.items) && q.items[i].UpdateID < offset {
		i++
	}
	if i > 0 {
		q.items = q.items[i:]
	}
}

func (q *Queue) fetch(offset int64, limit int) []models.Update {
	q.mu.Lock()
	defer q.mu.Unlock()

	var result []models.Update
	for _, u := range q.items {
		if u.UpdateID >= offset {
			result = append(result, u)
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}
