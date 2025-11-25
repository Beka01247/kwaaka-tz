package queue

import (
	"context"
)

type Broker interface {
	Publish(ctx context.Context, queueName string, message []byte) error
	Subscribe(ctx context.Context, queueName string, handler MessageHandler) error
	Close() error
}

type MessageHandler func(ctx context.Context, message []byte) error

const (
	QueueMenuParsing      = "menu-parsing"
	QueueProductStatus    = "product-status"
	QueueMenuParsingDLQ   = "menu-parsing-dlq"
	QueueProductStatusDLQ = "product-status-dlq"
)
