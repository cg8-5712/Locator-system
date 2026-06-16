package mqtt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	mqttlib "github.com/eclipse/paho.mqtt.golang"

	"locator/internal/config"
)

var ErrNotConnected = errors.New("mqtt client is not connected")

type MessageSnapshot struct {
	Topic      string    `json:"topic"`
	Payload    string    `json:"payload"`
	QoS        byte      `json:"qos"`
	Retained   bool      `json:"retained"`
	ReceivedAt time.Time `json:"received_at"`
}

type ReceivedMessage struct {
	Topic      string
	Payload    []byte
	QoS        byte
	Retained   bool
	ReceivedAt time.Time
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, message ReceivedMessage) error
}

type Client struct {
	cfg     config.MQTTConfig
	logger  *slog.Logger
	ctx     context.Context
	handler MessageHandler

	mu        sync.RWMutex
	client    mqttlib.Client
	messages  []MessageSnapshot
	closeOnce sync.Once
}

func New(cfg config.MQTTConfig, logger *slog.Logger, handler MessageHandler) *Client {
	return &Client{
		cfg:      cfg,
		logger:   logger,
		handler:  handler,
		messages: make([]MessageSnapshot, 0, 50),
	}
}

func (c *Client) Start(ctx context.Context) error {
	if !c.cfg.Enabled {
		c.logger.Info("mqtt client disabled")
		return nil
	}

	c.ctx = ctx

	options := mqttlib.NewClientOptions()
	options.AddBroker(c.cfg.Broker)
	options.SetClientID(c.cfg.ClientID)
	options.SetUsername(c.cfg.Username)
	options.SetPassword(c.cfg.Password)
	options.SetAutoReconnect(true)
	options.SetConnectRetry(true)
	options.SetConnectRetryInterval(3 * time.Second)
	options.SetOrderMatters(false)
	options.SetConnectionLostHandler(func(_ mqttlib.Client, err error) {
		c.logger.Warn("mqtt connection lost", "error", err)
	})
	options.SetOnConnectHandler(func(client mqttlib.Client) {
		c.logger.Info("mqtt connected", "broker", c.cfg.Broker, "client_id", c.cfg.ClientID)

		if err := c.subscribeAll(client); err != nil {
			c.logger.Error("mqtt subscribe failed after connect", "error", err)
		}
	})

	client := mqttlib.NewClient(options)
	token := client.Connect()
	if !token.WaitTimeout(c.cfg.ConnectTimeout) {
		return fmt.Errorf("mqtt connect timeout after %s", c.cfg.ConnectTimeout)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("connect mqtt broker: %w", err)
	}

	c.mu.Lock()
	c.client = client
	c.mu.Unlock()

	go func() {
		<-ctx.Done()
		c.Close()
	}()

	return nil
}

func (c *Client) Enabled() bool {
	return c.cfg.Enabled
}

func (c *Client) Connected() bool {
	client, ok := c.currentClient()
	if !ok {
		return false
	}

	return client.IsConnected()
}

func (c *Client) Topics() []string {
	return append([]string(nil), c.cfg.Topics...)
}

func (c *Client) Publish(ctx context.Context, topic string, payload []byte, qos byte, retained bool) error {
	if !c.cfg.Enabled {
		return ErrNotConnected
	}

	client, ok := c.currentClient()
	if !ok || !client.IsConnected() {
		return ErrNotConnected
	}

	timeout := c.cfg.OperationTimeout
	if deadline, ok := ctx.Deadline(); ok {
		untilDeadline := time.Until(deadline)
		if untilDeadline > 0 && untilDeadline < timeout {
			timeout = untilDeadline
		}
	}

	token := client.Publish(topic, qos, retained, payload)
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("mqtt publish timeout after %s", timeout)
	}

	if err := token.Error(); err != nil {
		return fmt.Errorf("publish mqtt message: %w", err)
	}

	c.logger.Info("mqtt message published",
		"topic", topic,
		"qos", qos,
		"retained", retained,
	)

	return nil
}

func (c *Client) RecentMessages(limit int) []MessageSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.messages) {
		limit = len(c.messages)
	}

	result := make([]MessageSnapshot, 0, limit)
	for i := len(c.messages) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, c.messages[i])
	}

	return result
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		client, ok := c.currentClient()
		if ok && client.IsConnectionOpen() {
			client.Disconnect(250)
			c.logger.Info("mqtt client disconnected")
		}
	})
}

func (c *Client) subscribeAll(client mqttlib.Client) error {
	for _, topic := range c.cfg.Topics {
		token := client.Subscribe(topic, c.cfg.QoS, c.handleMessage)
		if !token.WaitTimeout(c.cfg.OperationTimeout) {
			return fmt.Errorf("subscribe timeout for topic %s", topic)
		}

		if err := token.Error(); err != nil {
			return fmt.Errorf("subscribe topic %s: %w", topic, err)
		}

		c.logger.Info("mqtt subscribed", "topic", topic, "qos", c.cfg.QoS)
	}

	return nil
}

func (c *Client) handleMessage(_ mqttlib.Client, message mqttlib.Message) {
	receivedAt := time.Now().UTC()
	received := ReceivedMessage{
		Topic:      message.Topic(),
		Payload:    append([]byte(nil), message.Payload()...),
		QoS:        message.Qos(),
		Retained:   message.Retained(),
		ReceivedAt: receivedAt,
	}

	snapshot := MessageSnapshot{
		Topic:      message.Topic(),
		Payload:    string(message.Payload()),
		QoS:        message.Qos(),
		Retained:   message.Retained(),
		ReceivedAt: receivedAt,
	}

	c.appendMessage(snapshot)

	c.logger.Info("mqtt message received",
		"topic", snapshot.Topic,
		"qos", snapshot.QoS,
		"retained", snapshot.Retained,
		"payload", snapshot.Payload,
	)

	if c.handler != nil {
		go c.dispatchMessage(received)
	}
}

func (c *Client) appendMessage(message MessageSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	const maxMessages = 50

	c.messages = append(c.messages, message)
	if len(c.messages) > maxMessages {
		c.messages = append([]MessageSnapshot(nil), c.messages[len(c.messages)-maxMessages:]...)
	}
}

func (c *Client) currentClient() (mqttlib.Client, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.client == nil {
		return nil, false
	}

	return c.client, true
}

func (c *Client) dispatchMessage(message ReceivedMessage) {
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	if c.cfg.OperationTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.cfg.OperationTimeout)
		defer cancel()
	}

	if err := c.handler.HandleMessage(ctx, message); err != nil {
		c.logger.Error("mqtt message processing failed",
			"topic", message.Topic,
			"error", err,
		)
	}
}
