package indexer

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestProcessQueuedEventsReleasesSlotWhenSlowestChainHasNoQueuedEvents(t *testing.T) {
	coordinator := NewMultiChainCoordinator(true)
	coordinator.SetHandler(func(*BlockEvent) error {
		return nil
	})

	coordinator.chainProgress = map[string]int64{
		"doge": 0,
		"btc":  10_000,
		"mvc":  10_000,
	}
	coordinator.chainSlotUsage = map[string]int{
		"doge": 0,
		"btc":  16,
		"mvc":  34,
	}

	for i := 0; i < coordinator.maxQueueSize; i++ {
		coordinator.queueSemaphore <- struct{}{}
		chainName := "mvc"
		if i >= 34 {
			chainName = "btc"
		}
		coordinator.eventQueue.PushEvent(&BlockEvent{
			ChainName: chainName,
			Height:    int64(i + 1),
			Timestamp: int64(1_000 + i),
		})
	}

	var logOutput bytes.Buffer
	originalLogOutput := log.Writer()
	log.SetOutput(&logOutput)
	defer log.SetOutput(originalLogOutput)

	coordinator.processQueuedEvents()

	if got := coordinator.eventQueue.Size(); got >= coordinator.maxQueueSize {
		t.Fatalf("expected processQueuedEvents to release queued events, queue size = %d", got)
	}
	if got := len(coordinator.queueSemaphore); got >= coordinator.maxQueueSize {
		t.Fatalf("expected processQueuedEvents to release semaphore slots, in use = %d", got)
	}
	if !strings.Contains(logOutput.String(), "Slowest chain [doge] has no queued events") {
		t.Fatalf("expected explicit slowest-chain fallback log, got logs:\n%s", logOutput.String())
	}
}
