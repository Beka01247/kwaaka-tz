package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Beka01247/kwaaka-tz/internal/domain"
	"github.com/Beka01247/kwaaka-tz/internal/queue"
	"github.com/Beka01247/kwaaka-tz/internal/service"
	"go.uber.org/zap"
)

type ProductStatusWorker struct {
	productService *service.ProductService
	broker         queue.Broker
	logger         *zap.SugaredLogger
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewProductStatusWorker(
	productService *service.ProductService,
	broker queue.Broker,
	logger *zap.SugaredLogger,
) *ProductStatusWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &ProductStatusWorker{
		productService: productService,
		broker:         broker,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (w *ProductStatusWorker) Start() error {
	w.logger.Info("starting product status worker")

	return w.broker.Subscribe(w.ctx, queue.QueueProductStatus, w.handleMessage)
}

func (w *ProductStatusWorker) Stop() {
	w.logger.Info("stopping product status worker")
	w.cancel()
}

func (w *ProductStatusWorker) handleMessage(ctx context.Context, message []byte) error {
	var event domain.ProductStatusEvent
	if err := json.Unmarshal(message, &event); err != nil {
		w.logger.Errorw("failed to unmarshal event", "error", err)
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	w.logger.Infow("processing product status event", "product_id", event.ProductID, "event_type", event.EventType)

	if err := w.productService.ProcessProductStatusEvent(ctx, event); err != nil {
		w.logger.Errorw("failed to process product status event", "product_id", event.ProductID, "error", err)
		return err
	}

	return nil
}
