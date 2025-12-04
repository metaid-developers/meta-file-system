package indexer

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/bitcoinsv/bsvd/wire"
	btcwire "github.com/btcsuite/btcd/wire"
)

// MultiChainCoordinator coordinates multiple blockchain scanners and ensures time-ordered processing
type MultiChainCoordinator struct {
	scanners      map[string]*BlockScanner // Map of chain name to scanner
	eventQueue    *BlockEventQueue         // Priority queue ordered by timestamp
	chainProgress map[string]int64         // Latest timestamp scanned by each chain
	progressMu    sync.RWMutex             // Mutex for chainProgress
	handler       func(*BlockEvent) error  // Handler for processing block events
	ctx           context.Context          // Context for cancellation
	cancel        context.CancelFunc       // Cancel function
	wg            sync.WaitGroup           // Wait group for goroutines

	// Configuration
	timeOrderingEnabled bool          // Enable strict time ordering
	scanInterval        time.Duration // Interval for checking queue

	// Channels
	eventChan chan *BlockEvent // Channel for receiving block events
	stopChan  chan struct{}    // Channel for stopping

	// Statistics
	processedBlocks map[string]int64 // Number of blocks processed per chain
	statsMu         sync.RWMutex     // Mutex for statistics

	// Backpressure control
	maxQueueSize     int            // Maximum queue size before applying backpressure
	queueSemaphore   chan struct{}  // Semaphore to limit concurrent block loading
	perChainQuota    int            // Maximum slots each chain can occupy
	chainSlotUsage   map[string]int // Track how many slots each chain is using
	chainSlotUsageMu sync.Mutex     // Mutex for chain slot usage

	// State tracking for logging (only log on state change)
	chainBlocked         map[string]bool  // Track if chain is currently blocked
	chainCurrentHeight   map[string]int64 // Track current height of each chain
	chainLatestHeight    map[string]int64 // Track latest available height of each chain (from RPC)
	chainCaughtUp        map[string]bool  // Track if chain has caught up to latest height
	lastTimeOrderingWarn time.Time        // Last time we warned about time ordering
	stateTrackingMu      sync.Mutex       // Mutex for state tracking
}

// NewMultiChainCoordinator creates a new multi-chain coordinator
func NewMultiChainCoordinator(timeOrderingEnabled bool) *MultiChainCoordinator {
	ctx, cancel := context.WithCancel(context.Background())
	maxQueueSize := 50 // Maximum blocks in queue before backpressure
	// Each chain can use at most 70% of total slots to prevent monopoly
	perChainQuota := int(float64(maxQueueSize) * 0.7)

	return &MultiChainCoordinator{
		scanners:            make(map[string]*BlockScanner),
		eventQueue:          NewBlockEventQueue(),
		chainProgress:       make(map[string]int64),
		handler:             nil,
		ctx:                 ctx,
		cancel:              cancel,
		timeOrderingEnabled: timeOrderingEnabled,
		scanInterval:        100 * time.Millisecond,
		eventChan:           make(chan *BlockEvent, 10), // Small buffer, rely on backpressure
		stopChan:            make(chan struct{}),
		processedBlocks:     make(map[string]int64),
		maxQueueSize:        maxQueueSize,
		queueSemaphore:      make(chan struct{}, maxQueueSize), // Limit concurrent blocks in memory
		perChainQuota:       perChainQuota,                     // Prevent one chain from monopolizing
		chainSlotUsage:      make(map[string]int),              // Track per-chain usage
		chainBlocked:        make(map[string]bool),             // Track blocked state
		chainCurrentHeight:  make(map[string]int64),            // Track current height
		chainLatestHeight:   make(map[string]int64),            // Track latest available height
		chainCaughtUp:       make(map[string]bool),             // Track caught-up status
	}
}

// AddChain adds a blockchain scanner to the coordinator
func (c *MultiChainCoordinator) AddChain(chainName string, scanner *BlockScanner) error {
	if _, exists := c.scanners[chainName]; exists {
		return fmt.Errorf("chain %s already exists", chainName)
	}

	c.scanners[chainName] = scanner
	c.progressMu.Lock()
	c.chainProgress[chainName] = 0
	c.progressMu.Unlock()

	c.statsMu.Lock()
	c.processedBlocks[chainName] = 0
	c.statsMu.Unlock()

	log.Printf("Added chain to coordinator: %s", chainName)
	return nil
}

// SetHandler sets the block event handler
func (c *MultiChainCoordinator) SetHandler(handler func(*BlockEvent) error) {
	c.handler = handler
}

// Start starts all chain scanners and the coordinator
func (c *MultiChainCoordinator) Start() error {
	if c.handler == nil {
		return fmt.Errorf("handler not set")
	}

	if len(c.scanners) == 0 {
		return fmt.Errorf("no chains configured")
	}

	log.Printf("Starting multi-chain coordinator with %d chains (time ordering: %v)",
		len(c.scanners), c.timeOrderingEnabled)

	// Start event processor
	c.wg.Add(1)
	go c.processEvents()

	// Start each chain scanner
	for chainName, scanner := range c.scanners {
		chainName := chainName // Capture for goroutine
		scanner := scanner

		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.runChainScanner(chainName, scanner)
		}()
	}

	// Start statistics reporter
	c.wg.Add(1)
	go c.reportStatistics()

	log.Println("Multi-chain coordinator started successfully")
	return nil
}

// Stop stops all scanners and the coordinator
func (c *MultiChainCoordinator) Stop() {
	log.Println("Stopping multi-chain coordinator...")

	// Stop all scanners
	for chainName, scanner := range c.scanners {
		log.Printf("Stopping scanner for chain: %s", chainName)
		scanner.Stop()
	}

	// Cancel context
	c.cancel()

	// Close channels
	close(c.stopChan)

	// Wait for all goroutines to finish
	c.wg.Wait()

	log.Println("Multi-chain coordinator stopped")
}

// runChainScanner runs a single chain scanner
func (c *MultiChainCoordinator) runChainScanner(chainName string, scanner *BlockScanner) {
	log.Printf("Starting scanner for chain: %s", chainName)

	// We manually control the scanning loop to emit block events
	c.scanBlocksForChain(chainName, scanner)

	// If we reach here, the scanner has stopped
	log.Printf("Scanner stopped for chain: %s", chainName)
}

// scanBlocksForChain manually scans blocks for a chain and emits events
func (c *MultiChainCoordinator) scanBlocksForChain(chainName string, scanner *BlockScanner) {
	currentHeight := scanner.startHeight
	log.Printf("Chain %s scanner started from height %d", chainName, currentHeight)

	zmqStarted := false // Track if ZMQ has been started for this chain

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// Get latest block height
			latestHeight, err := scanner.GetBlockCount()
			if err != nil {
				log.Printf("[%s] Failed to get block count: %v", chainName, err)
				time.Sleep(scanner.interval)
				continue
			}

			// Update latest height for this chain
			c.stateTrackingMu.Lock()
			c.chainLatestHeight[chainName] = latestHeight
			// Check if this chain has caught up (within 1 block of latest)
			wasCaughtUp := c.chainCaughtUp[chainName]
			isCaughtUp := currentHeight >= latestHeight
			c.chainCaughtUp[chainName] = isCaughtUp
			if !wasCaughtUp && isCaughtUp {
				log.Printf("✅ [%s] Caught up to latest block %d", chainName, latestHeight)
			} else if wasCaughtUp && !isCaughtUp {
				// Chain was synced but now has new blocks
				newBlocksCount := latestHeight - currentHeight + 1
				log.Printf("🔄 [%s] New blocks detected! Resuming sync from %d to %d (%d blocks)",
					chainName, currentHeight, latestHeight, newBlocksCount)
			}
			c.stateTrackingMu.Unlock()

			// Scan new blocks
			if currentHeight <= latestHeight {
				for currentHeight <= latestHeight {
					select {
					case <-c.ctx.Done():
						return
					default:
						// Update current height for this chain
						c.stateTrackingMu.Lock()
						c.chainCurrentHeight[chainName] = currentHeight
						c.stateTrackingMu.Unlock()

						// **PER-CHAIN QUOTA CHECK**: Prevent one chain from monopolizing the queue
						c.chainSlotUsageMu.Lock()
						chainUsage := c.chainSlotUsage[chainName]
						quotaExceeded := chainUsage >= c.perChainQuota
						c.chainSlotUsageMu.Unlock()

						if quotaExceeded {
							c.stateTrackingMu.Lock()
							wasBlocked := c.chainBlocked[chainName]
							if !wasBlocked {
								log.Printf("⏸️  [%s] Blocked at block %d - Chain quota exceeded (%d/%d slots)",
									chainName, currentHeight, chainUsage, c.perChainQuota)
								c.chainBlocked[chainName] = true
							}
							c.stateTrackingMu.Unlock()

							// Wait a bit before retrying
							time.Sleep(100 * time.Millisecond)
							continue
						}

						// **BACKPRESSURE**: Acquire semaphore before loading block
						// This prevents loading too many blocks into memory at once
						queueFull := len(c.queueSemaphore) >= c.maxQueueSize

						// Only log when state changes (blocked -> not blocked or vice versa)
						if queueFull {
							c.stateTrackingMu.Lock()
							wasBlocked := c.chainBlocked[chainName]
							if !wasBlocked {
								// Get other chains' current heights
								otherChainsInfo := c.getOtherChainsInfo(chainName)
								log.Printf("⏸️  [%s] Blocked at block %d - Queue full (%d/%d), %s",
									chainName, currentHeight, len(c.queueSemaphore), c.maxQueueSize, otherChainsInfo)
								c.chainBlocked[chainName] = true
							}
							c.stateTrackingMu.Unlock()
						}

						select {
						case c.queueSemaphore <- struct{}{}:
							// Successfully acquired semaphore
							// Increment this chain's slot usage
							c.chainSlotUsageMu.Lock()
							c.chainSlotUsage[chainName]++
							c.chainSlotUsageMu.Unlock()

							c.stateTrackingMu.Lock()
							wasBlocked := c.chainBlocked[chainName]
							if wasBlocked {
								log.Printf("▶️  [%s] Resumed at block %d - Queue has space",
									chainName, currentHeight)
								c.chainBlocked[chainName] = false
							}
							c.stateTrackingMu.Unlock()
						case <-c.ctx.Done():
							return
						}

						// Get block message (this loads large data into memory)
						msgBlock, txCount, err := scanner.GetBlockMsg(currentHeight)
						if err != nil {
							log.Printf("❌ [%s] Failed to get block %d: %v", chainName, currentHeight, err)
							<-c.queueSemaphore // Release semaphore on error
							time.Sleep(scanner.interval)
							continue
						}

						// Extract timestamp based on chain type
						var timestamp int64
						if scanner.chainType == ChainTypeBTC {
							if btcBlock, ok := msgBlock.(*btcwire.MsgBlock); ok {
								timestamp = btcBlock.Header.Timestamp.UnixMilli()
							}
						} else {
							if mvcBlock, ok := msgBlock.(*wire.MsgBlock); ok {
								timestamp = mvcBlock.Header.Timestamp.UnixMilli()
							}
						}

						// Create block event
						event := &BlockEvent{
							ChainName: chainName,
							Height:    currentHeight,
							Timestamp: timestamp,
							Block:     msgBlock,
							TxCount:   txCount,
						}

						// Send event to channel (will block if full - good backpressure)
						select {
						case c.eventChan <- event:
							// Event sent successfully
							// Note: semaphore will be released after processing in processBlockEvent
						case <-c.ctx.Done():
							<-c.queueSemaphore // Release semaphore on cancellation
							return
						}

						// Update progress
						c.progressMu.Lock()
						c.chainProgress[chainName] = timestamp
						c.progressMu.Unlock()

						currentHeight++
					}
				}

				// Start ZMQ client after catching up to latest block (only once)
				if !zmqStarted && scanner.zmqEnabled && scanner.zmqClient != nil {
					log.Printf("✅ [%s] Caught up to latest block, scanning mempool before starting ZMQ...", chainName)

					// Scan mempool first to process any existing transactions
					// The scanner already has the ZMQ transaction handler set by IndexerService
					// which will call handleTransaction directly
					if _, err := scanner.ScanMempool(func(tx interface{}, metaDataTx *MetaIDDataTx, height, timestamp int64) error {
						// Use the ZMQ transaction handler if set (it's already configured by IndexerService)
						if scanner.zmqClient != nil && scanner.zmqClient.TxHandler != nil {
							return scanner.zmqClient.TxHandler(tx, metaDataTx)
						}
						return nil
					}); err != nil {
						log.Printf("[%s] Failed to scan mempool: %v (continuing anyway)", chainName, err)
					}

					log.Printf("✅ [%s] Mempool scan complete, starting ZMQ real-time monitoring...", chainName)

					// Start ZMQ client
					if err := scanner.zmqClient.StartWithRawTx(); err != nil {
						log.Printf("[%s] Failed to start ZMQ client: %v", chainName, err)
					} else {
						zmqStarted = true
						log.Printf("✅ [%s] ZMQ real-time monitoring started successfully", chainName)
					}
				}
			} else {
				// Already at latest block
				if !zmqStarted && scanner.zmqEnabled && scanner.zmqClient != nil {
					log.Printf("✅ [%s] At latest block, scanning mempool before starting ZMQ...", chainName)

					// Scan mempool first to process any existing transactions
					if _, err := scanner.ScanMempool(func(tx interface{}, metaDataTx *MetaIDDataTx, height, timestamp int64) error {
						// Use the ZMQ transaction handler if set (it's already configured by IndexerService)
						if scanner.zmqClient != nil && scanner.zmqClient.TxHandler != nil {
							return scanner.zmqClient.TxHandler(tx, metaDataTx)
						}
						return nil
					}); err != nil {
						log.Printf("[%s] Failed to scan mempool: %v (continuing anyway)", chainName, err)
					}

					log.Printf("✅ [%s] Mempool scan complete, starting ZMQ real-time monitoring...", chainName)

					// Start ZMQ client
					if err := scanner.zmqClient.StartWithRawTx(); err != nil {
						log.Printf("[%s] Failed to start ZMQ client: %v", chainName, err)
					} else {
						zmqStarted = true
						log.Printf("✅ [%s] ZMQ real-time monitoring started successfully", chainName)
					}
				}
			}

			// Wait before next scan
			time.Sleep(scanner.interval)
		}
	}
}

// processEvents processes block events from the queue
func (c *MultiChainCoordinator) processEvents() {
	defer c.wg.Done()

	log.Println("Event processor started")

	ticker := time.NewTicker(c.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			log.Println("Event processor stopped")
			return

		case event := <-c.eventChan:
			// Add event to queue
			c.eventQueue.PushEvent(event)

			// **IMMEDIATELY** try to process queued events
			// Don't wait for ticker - process as soon as event arrives
			c.processQueuedEvents()

		case <-ticker.C:
			// Also periodically try to process in case of time-ordering waits
			c.processQueuedEvents()
		}
	}
}

// processQueuedEvents processes events from the queue that are ready
func (c *MultiChainCoordinator) processQueuedEvents() {
	queueSize := c.eventQueue.Size()

	// **DEADLOCK PREVENTION**: If queue is nearly full and time ordering is blocking,
	// process slow chain events anyway to prevent deadlock
	if c.timeOrderingEnabled && queueSize >= c.maxQueueSize-5 {
		log.Printf("⚠️  Queue nearly full (%d/%d), processing slow chain events to prevent deadlock",
			queueSize, c.maxQueueSize)
		c.processSlowChainEvents()
	}

	for {
		// Peek at the next event
		event := c.eventQueue.Peek()
		if event == nil {
			// Queue is empty
			return
		}

		// Check if we can process this event
		if c.timeOrderingEnabled && !c.canProcessEvent(event) {
			// Need to wait for other chains to catch up
			c.logTimeOrderingBlock(event)
			return
		}

		// Pop the event and process it
		event = c.eventQueue.PopEvent()
		if event == nil {
			return
		}

		// Process the event
		if err := c.processBlockEvent(event); err != nil {
			log.Printf("❌ [%s] Failed to process block %d: %v",
				event.ChainName, event.Height, err)
		} else {
			// Update statistics
			c.statsMu.Lock()
			c.processedBlocks[event.ChainName]++
			c.statsMu.Unlock()
		}
	}
}

// processSlowChainEvents processes events from slow chains even if time ordering isn't met
// This prevents deadlock when fast chain fills the queue waiting for slow chain
func (c *MultiChainCoordinator) processSlowChainEvents() {
	c.progressMu.RLock()

	// Find the slowest chain (lowest timestamp progress)
	var slowestChain string
	var slowestProgress int64 = 9999999999999
	for chainName, progress := range c.chainProgress {
		if progress < slowestProgress {
			slowestProgress = progress
			slowestChain = chainName
		}
	}
	c.progressMu.RUnlock()

	if slowestChain == "" {
		return
	}

	// Process up to 5 events from the slow chain
	processed := 0
	for processed < 5 {
		event := c.eventQueue.PopEventForChain(slowestChain)
		if event == nil {
			break
		}

		log.Printf("🔓 [%s] Processing block %d (bypassing time ordering to free queue)",
			event.ChainName, event.Height)

		if err := c.processBlockEvent(event); err != nil {
			log.Printf("❌ [%s] Failed to process block %d: %v",
				event.ChainName, event.Height, err)
		} else {
			c.statsMu.Lock()
			c.processedBlocks[event.ChainName]++
			c.statsMu.Unlock()
			processed++
		}
	}

	if processed > 0 {
		log.Printf("✅ Processed %d events from slow chain [%s] to prevent deadlock", processed, slowestChain)
	}
}

// getOtherChainsInfo returns info about other chains (must be called with stateTrackingMu locked)
func (c *MultiChainCoordinator) getOtherChainsInfo(excludeChain string) string {
	var info []string
	for chain, height := range c.chainCurrentHeight {
		if chain != excludeChain && height > 0 {
			info = append(info, fmt.Sprintf("%s at block %d", chain, height))
		}
	}
	if len(info) == 0 {
		return "waiting..."
	}
	return fmt.Sprintf("others: %v", info)
}

// logTimeOrderingBlock logs which chains are blocking due to time ordering
func (c *MultiChainCoordinator) logTimeOrderingBlock(event *BlockEvent) {
	c.progressMu.RLock()
	defer c.progressMu.RUnlock()

	// Only log once every 10 seconds to avoid spam
	c.stateTrackingMu.Lock()
	defer c.stateTrackingMu.Unlock()

	now := time.Now()
	if now.Sub(c.lastTimeOrderingWarn) < 10*time.Second {
		return // Too soon, skip logging
	}

	queueSize := c.eventQueue.Size()
	if queueSize > 5 {
		// Show which chains haven't caught up yet with their current heights
		var slowChains []string
		for chainName, progress := range c.chainProgress {
			if chainName != event.ChainName && progress < event.Timestamp {
				timeDiff := event.Timestamp - progress
				currentHeight := c.chainCurrentHeight[chainName]
				slowChains = append(slowChains, fmt.Sprintf("%s at block %d (-%dms)", chainName, currentHeight, timeDiff))
			}
		}
		if len(slowChains) > 0 {
			log.Printf("⏳ [%s] Block %d waiting for: %v (Queue: %d)",
				event.ChainName, event.Height, slowChains, queueSize)
			c.lastTimeOrderingWarn = now
		}
	}
}

// canProcessEvent checks if an event can be processed based on time ordering
func (c *MultiChainCoordinator) canProcessEvent(event *BlockEvent) bool {
	c.progressMu.RLock()
	defer c.progressMu.RUnlock()

	// Check if all chains have scanned past this timestamp
	for chainName, progress := range c.chainProgress {
		if chainName == event.ChainName {
			continue // Skip the event's own chain
		}

		// Check if this chain has caught up to latest
		c.stateTrackingMu.Lock()
		chainCaughtUp := c.chainCaughtUp[chainName]
		c.stateTrackingMu.Unlock()

		// If this chain has caught up to latest, don't wait for it
		// It won't produce new blocks with earlier timestamps
		if chainCaughtUp {
			continue
		}

		// If any chain hasn't reached this timestamp yet, wait
		if progress < event.Timestamp {
			return false
		}
	}

	return true
}

// processBlockEvent processes a single block event
func (c *MultiChainCoordinator) processBlockEvent(event *BlockEvent) error {
	// Ensure semaphore and chain quota are released after processing
	defer func() {
		// Decrement this chain's slot usage
		c.chainSlotUsageMu.Lock()
		if c.chainSlotUsage[event.ChainName] > 0 {
			c.chainSlotUsage[event.ChainName]--
		}
		c.chainSlotUsageMu.Unlock()

		// Release semaphore to allow loading next block
		<-c.queueSemaphore
	}()

	// Call the handler
	if c.handler != nil {
		err := c.handler(event)

		// Clear large objects after processing to help GC
		// Block data can be very large (especially MVC blocks up to 4GB)
		event.Block = nil

		return err
	}
	return nil
}

// reportStatistics periodically reports processing statistics
func (c *MultiChainCoordinator) reportStatistics() {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.printStatistics()
		}
	}
}

// printStatistics prints current processing statistics
func (c *MultiChainCoordinator) printStatistics() {
	c.progressMu.RLock()
	c.statsMu.RLock()
	defer c.progressMu.RUnlock()
	defer c.statsMu.RUnlock()

	queueSize := c.eventQueue.Size()
	channelSize := len(c.eventChan)
	semaphoreInUse := len(c.queueSemaphore)

	// Get memory statistics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	log.Println("=== Multi-Chain Coordinator Statistics ===")
	log.Printf("Event Queue: %d, Channel: %d/%d, Blocks in memory: %d/%d",
		queueSize, channelSize, cap(c.eventChan), semaphoreInUse, c.maxQueueSize)
	log.Printf("Memory: Alloc=%vMB, TotalAlloc=%vMB, Sys=%vMB, NumGC=%v",
		m.Alloc/1024/1024, m.TotalAlloc/1024/1024, m.Sys/1024/1024, m.NumGC)

	c.chainSlotUsageMu.Lock()
	c.stateTrackingMu.Lock()
	for chainName := range c.scanners {
		progress := c.chainProgress[chainName]
		processed := c.processedBlocks[chainName]
		slotUsage := c.chainSlotUsage[chainName]
		currentHeight := c.chainCurrentHeight[chainName]
		latestHeight := c.chainLatestHeight[chainName]
		caughtUp := c.chainCaughtUp[chainName]

		// Convert timestamp to readable format
		timeStr := "N/A"
		if progress > 0 {
			timeStr = time.UnixMilli(progress).Format("2006-01-02 15:04:05")
		}

		// Status indicator
		status := "syncing"
		if caughtUp {
			status = "✅ synced"
		}

		log.Printf("  [%s] %s | Height: %d/%d | Slots: %d/%d | Latest: %s | Processed: %d",
			chainName, status, currentHeight, latestHeight, slotUsage, c.perChainQuota, timeStr, processed)
	}
	c.stateTrackingMu.Unlock()
	c.chainSlotUsageMu.Unlock()

	log.Println("==========================================")
}

// GetChainProgress returns the current progress for all chains
func (c *MultiChainCoordinator) GetChainProgress() map[string]int64 {
	c.progressMu.RLock()
	defer c.progressMu.RUnlock()

	progress := make(map[string]int64)
	for k, v := range c.chainProgress {
		progress[k] = v
	}
	return progress
}

// GetQueueSize returns the current size of the event queue
func (c *MultiChainCoordinator) GetQueueSize() int {
	return c.eventQueue.Size()
}

// GetScanner returns the block scanner for a specific chain
func (c *MultiChainCoordinator) GetScanner(chainName string) *BlockScanner {
	return c.scanners[chainName]
}
