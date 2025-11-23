package indexer

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"meta-file-system/common"
	"strings"
	"sync"
	"time"

	"github.com/bitcoinsv/bsvd/wire"
	btcwire "github.com/btcsuite/btcd/wire"
	"github.com/go-zeromq/zmq4"
)

// ZMQClient represents a ZeroMQ client for listening to mempool events sent by Bitcoin nodes
type ZMQClient struct {
	// ZMQ connection address, e.g. "tcp://127.0.0.1:28332"
	address string

	// List of topics to listen to, e.g. "rawtx", "hashtx", etc.
	topics []string

	// Context control
	ctx    context.Context
	cancel context.CancelFunc

	// Wait for all goroutines to finish
	wg sync.WaitGroup

	// Connection and reconnection interval
	reconnectInterval time.Duration

	// Handler mapping, each topic corresponds to a handler function
	handlers map[string]MessageHandler

	// Chain type
	chainType ChainType

	// Transaction handler
	txHandler func(tx interface{}, metaDataTx *MetaIDDataTx) error
}

// MessageHandler is the function type for handling ZMQ messages
type MessageHandler func(topic string, data []byte) error

// NewZMQClient creates a new ZMQ client
func NewZMQClient(address string, chainType ChainType) *ZMQClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &ZMQClient{
		address:           address,
		topics:            []string{},
		ctx:               ctx,
		cancel:            cancel,
		reconnectInterval: 5 * time.Second,
		handlers:          make(map[string]MessageHandler),
		chainType:         chainType,
	}
}

// SetTransactionHandler set transaction handler for processing MetaID transactions
func (c *ZMQClient) SetTransactionHandler(handler func(tx interface{}, metaDataTx *MetaIDDataTx) error) {
	c.txHandler = handler
}

// AddTopic adds a topic to listen to and its handler
func (c *ZMQClient) AddTopic(topic string, handler MessageHandler) {
	// Ensure topic is not duplicated
	for _, t := range c.topics {
		if t == topic {
			return
		}
	}

	c.topics = append(c.topics, topic)
	c.handlers[topic] = handler
}

// Start starts listening to ZMQ messages
func (c *ZMQClient) Start() error {
	if len(c.topics) == 0 {
		return fmt.Errorf("no topics added, please use AddTopic to add topics to listen to")
	}

	log.Printf("Starting ZMQ client for %s chain: %s", c.chainType, c.address)
	log.Printf("Listening to topics: %s", strings.Join(c.topics, ", "))

	// Start listening goroutine
	c.wg.Add(1)
	go c.listen()

	return nil
}

// Stop stops listening
func (c *ZMQClient) Stop() {
	log.Println("Stopping ZMQ client...")
	c.cancel()
	c.wg.Wait()
	log.Println("ZMQ client stopped")
}

// listen is an internal method for listening to ZMQ messages
func (c *ZMQClient) listen() {
	defer c.wg.Done()

	log.Printf("Starting ZMQ client listener: %s", c.address)
	for {
		select {
		case <-c.ctx.Done():
			log.Println("Received stop signal, ZMQ client is shutting down...")
			return
		default:
			// Create a new socket
			socket := zmq4.NewSub(c.ctx)
			defer socket.Close()

			// Connect to ZMQ server
			if err := socket.Dial(c.address); err != nil {
				log.Printf("Failed to connect to ZMQ server: %v, will retry in %v",
					err, c.reconnectInterval)
				time.Sleep(c.reconnectInterval)
				continue
			}

			// Subscribe to all topics
			for _, topic := range c.topics {
				if err := socket.SetOption(zmq4.OptionSubscribe, topic); err != nil {
					log.Printf("Failed to subscribe to topic %s: %v", topic, err)
					continue
				}
				log.Printf("Successfully subscribed to topic: %s", topic)
			}

			log.Printf("Successfully connected to ZMQ server: %s", c.address)

			// Receive message loop
			c.receiveMessages(socket)

			// If receiveMessages returns, the connection is broken or an error occurred, reconnect
			log.Printf("ZMQ connection lost, will reconnect in %v", c.reconnectInterval)
			time.Sleep(c.reconnectInterval)
		}
	}
}

// receiveMessages receives and processes ZMQ messages
func (c *ZMQClient) receiveMessages(socket zmq4.Socket) {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// Receive message
			msg, err := socket.Recv()
			if err != nil {
				log.Printf("Failed to receive message: %v", err)
				return
			}

			// Ensure message has at least two parts: topic and data
			if len(msg.Frames) < 2 {
				log.Printf("Received message with incorrect format: %v", msg)
				continue
			}

			// First frame is topic
			topic := string(msg.Frames[0])

			// Find corresponding handler
			handler, ok := c.handlers[topic]
			if !ok {
				log.Printf("Received message for unknown topic: %s", topic)
				continue
			}

			// Call handler to process message
			if err := handler(topic, msg.Frames[1]); err != nil {
				log.Printf("Failed to process message [%s]: %v", topic, err)
			}
		}
	}
}

// handleRawTx handles raw transaction messages
func (c *ZMQClient) handleRawTx(topic string, data []byte) error {
	// Parse transaction based on chain type
	var tx interface{}
	var err error

	if c.chainType == ChainTypeBTC {
		// Parse as BTC transaction
		var btcTx btcwire.MsgTx
		if err = btcTx.Deserialize(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("failed to deserialize BTC transaction: %w", err)
		}
		tx = &btcTx
		log.Printf("Received BTC transaction from ZMQ: %s", btcTx.TxHash().String())
	} else {
		// Parse as MVC transaction
		var mvcTx wire.MsgTx
		if err = mvcTx.Deserialize(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("failed to deserialize MVC transaction: %w", err)
		}
		tx = &mvcTx
		log.Printf("Received MVC transaction from ZMQ: %s", common.GetMvcTxhashFromRaw(hex.EncodeToString(data)))
	}

	// Parse MetaID data
	parser := NewMetaIDParser("")
	metaDataTx, err := parser.ParseAllPINs(tx, c.chainType)
	if err != nil || metaDataTx == nil {
		// Not a MetaID transaction, skip
		return nil
	}

	log.Printf("Found MetaID transaction from ZMQ: %s (chain: %s), PIN count: %d",
		metaDataTx.TxID, metaDataTx.ChainName, len(metaDataTx.MetaIDData))

	// Call transaction handler if set
	if c.txHandler != nil {
		if err := c.txHandler(tx, metaDataTx); err != nil {
			return fmt.Errorf("failed to handle transaction: %w", err)
		}
	}

	return nil
}

// handleHashTx handles transaction hash messages
func (c *ZMQClient) handleHashTx(topic string, data []byte) error {
	txHash := hex.EncodeToString(data)
	log.Printf("Received transaction hash from ZMQ: %s", txHash)
	// You can implement additional logic here if needed
	return nil
}

// StartWithRawTx starts ZMQ client and listens to raw transaction topic
func (c *ZMQClient) StartWithRawTx() error {
	// Add rawtx topic with handler
	c.AddTopic("rawtx", c.handleRawTx)
	return c.Start()
}

// StartWithHashTx starts ZMQ client and listens to transaction hash topic
func (c *ZMQClient) StartWithHashTx() error {
	// Add hashtx topic with handler
	c.AddTopic("hashtx", c.handleHashTx)
	return c.Start()
}

// StartWithBothTopics starts ZMQ client and listens to both rawtx and hashtx topics
func (c *ZMQClient) StartWithBothTopics() error {
	// Add both topics
	c.AddTopic("rawtx", c.handleRawTx)
	c.AddTopic("hashtx", c.handleHashTx)
	return c.Start()
}
