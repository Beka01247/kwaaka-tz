package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQBroker struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
	mu      sync.RWMutex
}

type Config struct {
	URL           string
	MaxRetries    int
	RetryDelay    time.Duration
	PrefetchCount int
}

func NewRabbitMQBroker(cfg Config) (*RabbitMQBroker, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// set QoS
	if err := channel.Qos(cfg.PrefetchCount, 0, false); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	broker := &RabbitMQBroker{
		conn:    conn,
		channel: channel,
		url:     cfg.URL,
	}

	// declare queues
	queues := []string{
		QueueMenuParsing,
		QueueProductStatus,
		QueueMenuParsingDLQ,
		QueueProductStatusDLQ,
	}

	for _, queueName := range queues {
		if err := broker.declareQueue(queueName); err != nil {
			broker.Close()
			return nil, err
		}
	}

	return broker, nil
}

func (b *RabbitMQBroker) declareQueue(queueName string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	_, err := b.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
	}

	return nil
}

func (b *RabbitMQBroker) Publish(ctx context.Context, queueName string, message []byte) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	err := b.channel.PublishWithContext(
		ctx,
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         message,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (b *RabbitMQBroker) Subscribe(ctx context.Context, queueName string, handler MessageHandler) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	msgs, err := b.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				b.handleMessage(ctx, msg, handler, queueName)
			}
		}
	}()

	return nil
}

func (b *RabbitMQBroker) handleMessage(ctx context.Context, msg amqp.Delivery, handler MessageHandler, queueName string) {
	err := handler(ctx, msg.Body)
	if err != nil {
		// retry count
		retryCount := 0
		if msg.Headers != nil {
			if count, ok := msg.Headers["x-retry-count"].(int32); ok {
				retryCount = int(count)
			}
		}

		maxRetries := 3
		if retryCount < maxRetries {
			// exponential backoff: 2^retryCount seconds
			// retry 1: 2 seconds, retry 2: 4 seconds, retry 3: 8 seconds
			delaySeconds := 1 << retryCount // 2^retryCount
			time.Sleep(time.Duration(delaySeconds) * time.Second)

			// requeue with incremented retry count
			headers := amqp.Table{
				"x-retry-count": int32(retryCount + 1),
			}

			b.mu.RLock()
			_ = b.channel.PublishWithContext(
				ctx,
				"",
				queueName,
				false,
				false,
				amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					ContentType:  msg.ContentType,
					Body:         msg.Body,
					Headers:      headers,
					Timestamp:    time.Now(),
				},
			)
			b.mu.RUnlock()

			msg.Ack(false)
		} else {
			// dlq
			dlqName := queueName + "-dlq"
			b.mu.RLock()
			_ = b.channel.PublishWithContext(
				ctx,
				"",
				dlqName,
				false,
				false,
				amqp.Publishing{
					DeliveryMode: amqp.Persistent,
					ContentType:  msg.ContentType,
					Body:         msg.Body,
					Headers: amqp.Table{
						"x-original-queue": queueName,
						"x-retry-count":    int32(retryCount),
						"x-error":          err.Error(),
					},
					Timestamp: time.Now(),
				},
			)
			b.mu.RUnlock()

			msg.Ack(false)
		}
	} else {
		msg.Ack(false)
	}
}

func (b *RabbitMQBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.channel != nil {
		b.channel.Close()
	}
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}
