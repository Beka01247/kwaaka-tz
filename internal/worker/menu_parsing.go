package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Beka01247/kwaaka-tz/internal/domain"
	"github.com/Beka01247/kwaaka-tz/internal/queue"
	"github.com/Beka01247/kwaaka-tz/internal/service"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type MenuParsingWorker struct {
	parsingService *service.ParsingService
	broker         queue.Broker
	logger         *zap.SugaredLogger
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewMenuParsingWorker(
	parsingService *service.ParsingService,
	broker queue.Broker,
	logger *zap.SugaredLogger,
) *MenuParsingWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &MenuParsingWorker{
		parsingService: parsingService,
		broker:         broker,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (w *MenuParsingWorker) Start() error {
	w.logger.Info("starting menu parsing worker")

	return w.broker.Subscribe(w.ctx, queue.QueueMenuParsing, w.handleMessage)
}

func (w *MenuParsingWorker) Stop() {
	w.logger.Info("stopping menu parsing worker")
	w.cancel()
}

func (w *MenuParsingWorker) handleMessage(ctx context.Context, message []byte) error {
	var msg domain.MenuParsingMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		w.logger.Errorw("failed to unmarshal message", "error", err)
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	w.logger.Infow("processing menu parsing message", "task_id", msg.TaskID)

	taskID, err := primitive.ObjectIDFromHex(msg.TaskID)
	if err != nil {
		w.logger.Errorw("invalid task ID", "task_id", msg.TaskID, "error", err)
		return fmt.Errorf("invalid task ID: %w", err)
	}

	if err := w.parsingService.ProcessParsingTask(ctx, taskID); err != nil {
		w.logger.Errorw("failed to process parsing task", "task_id", msg.TaskID, "error", err)
		return err
	}

	return nil
}
