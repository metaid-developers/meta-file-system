package indexer

import (
	"testing"
	"time"
)

func TestBlockEventQueue_PushAndPop(t *testing.T) {
	queue := NewBlockEventQueue()

	// Create test events with different timestamps
	event1 := &BlockEvent{
		ChainName: "btc",
		Height:    100,
		Timestamp: 1000,
	}
	event2 := &BlockEvent{
		ChainName: "mvc",
		Height:    200,
		Timestamp: 500, // Earlier timestamp
	}
	event3 := &BlockEvent{
		ChainName: "btc",
		Height:    101,
		Timestamp: 1500,
	}

	// Push events
	queue.PushEvent(event1)
	queue.PushEvent(event2)
	queue.PushEvent(event3)

	// Verify queue size
	if queue.Size() != 3 {
		t.Errorf("Expected queue size 3, got %d", queue.Size())
	}

	// Pop events - should come out in timestamp order
	popped1 := queue.PopEvent()
	if popped1.Timestamp != 500 {
		t.Errorf("Expected first event timestamp 500, got %d", popped1.Timestamp)
	}
	if popped1.ChainName != "mvc" {
		t.Errorf("Expected first event chain 'mvc', got %s", popped1.ChainName)
	}

	popped2 := queue.PopEvent()
	if popped2.Timestamp != 1000 {
		t.Errorf("Expected second event timestamp 1000, got %d", popped2.Timestamp)
	}

	popped3 := queue.PopEvent()
	if popped3.Timestamp != 1500 {
		t.Errorf("Expected third event timestamp 1500, got %d", popped3.Timestamp)
	}

	// Queue should be empty
	if queue.Size() != 0 {
		t.Errorf("Expected empty queue, got size %d", queue.Size())
	}

	// Pop from empty queue should return nil
	popped4 := queue.PopEvent()
	if popped4 != nil {
		t.Error("Expected nil from empty queue")
	}
}

func TestBlockEventQueue_Peek(t *testing.T) {
	queue := NewBlockEventQueue()

	// Peek empty queue
	if queue.Peek() != nil {
		t.Error("Expected nil from empty queue peek")
	}

	// Add events
	event1 := &BlockEvent{
		ChainName: "btc",
		Height:    100,
		Timestamp: 2000,
	}
	event2 := &BlockEvent{
		ChainName: "mvc",
		Height:    200,
		Timestamp: 1000,
	}

	queue.PushEvent(event1)
	queue.PushEvent(event2)

	// Peek should return earliest timestamp without removing
	peeked := queue.Peek()
	if peeked.Timestamp != 1000 {
		t.Errorf("Expected peek timestamp 1000, got %d", peeked.Timestamp)
	}

	// Size should remain unchanged
	if queue.Size() != 2 {
		t.Errorf("Expected queue size 2 after peek, got %d", queue.Size())
	}

	// Pop should return the same event
	popped := queue.PopEvent()
	if popped.Timestamp != 1000 {
		t.Errorf("Expected pop timestamp 1000, got %d", popped.Timestamp)
	}
}

func TestBlockEventQueue_Concurrent(t *testing.T) {
	queue := NewBlockEventQueue()
	done := make(chan bool)

	// Producer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			event := &BlockEvent{
				ChainName: "test",
				Height:    int64(i),
				Timestamp: int64(i * 1000),
			}
			queue.PushEvent(event)
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Consumer goroutine
	go func() {
		count := 0
		for count < 100 {
			if event := queue.PopEvent(); event != nil {
				count++
			}
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Queue should be empty
	if queue.Size() != 0 {
		t.Errorf("Expected empty queue after concurrent operations, got size %d", queue.Size())
	}
}
