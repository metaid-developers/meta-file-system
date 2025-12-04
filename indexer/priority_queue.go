package indexer

import (
	"container/heap"
	"sync"
)

// BlockEventQueue is a priority queue for BlockEvent, ordered by timestamp
type BlockEventQueue struct {
	items []*BlockEvent
	mu    sync.RWMutex
}

// NewBlockEventQueue creates a new priority queue
func NewBlockEventQueue() *BlockEventQueue {
	q := &BlockEventQueue{
		items: make([]*BlockEvent, 0),
	}
	heap.Init(q)
	return q
}

// Len returns the number of items in the queue
func (q *BlockEventQueue) Len() int {
	return len(q.items)
}

// Less compares two items by timestamp (earlier timestamp has higher priority)
func (q *BlockEventQueue) Less(i, j int) bool {
	return q.items[i].Timestamp < q.items[j].Timestamp
}

// Swap swaps two items in the queue
func (q *BlockEventQueue) Swap(i, j int) {
	q.items[i], q.items[j] = q.items[j], q.items[i]
}

// Push adds an item to the queue
func (q *BlockEventQueue) Push(x interface{}) {
	q.items = append(q.items, x.(*BlockEvent))
}

// Pop removes and returns the item with the smallest timestamp
func (q *BlockEventQueue) Pop() interface{} {
	old := q.items
	n := len(old)
	item := old[n-1]
	q.items = old[0 : n-1]
	return item
}

// PushEvent adds a block event to the queue (thread-safe)
func (q *BlockEventQueue) PushEvent(event *BlockEvent) {
	q.mu.Lock()
	defer q.mu.Unlock()
	heap.Push(q, event)
}

// PopEvent removes and returns the event with the smallest timestamp (thread-safe)
func (q *BlockEventQueue) PopEvent() *BlockEvent {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.Len() == 0 {
		return nil
	}
	return heap.Pop(q).(*BlockEvent)
}

// Peek returns the event with the smallest timestamp without removing it (thread-safe)
func (q *BlockEventQueue) Peek() *BlockEvent {
	q.mu.RLock()
	defer q.mu.RUnlock()
	if q.Len() == 0 {
		return nil
	}
	return q.items[0]
}

// Size returns the number of items in the queue (thread-safe)
func (q *BlockEventQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.Len()
}

// PopEventForChain removes and returns the first event for a specific chain (thread-safe)
// Used for deadlock prevention - allows processing slow chain events out of order
func (q *BlockEventQueue) PopEventForChain(chainName string) *BlockEvent {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Find the first event for this chain
	for i := 0; i < len(q.items); i++ {
		if q.items[i].ChainName == chainName {
			// Remove this item from the heap
			event := q.items[i]
			heap.Remove(q, i)
			return event
		}
	}

	return nil
}
