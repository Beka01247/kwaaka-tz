package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Beka01247/kwaaka-tz/internal/domain"
	"github.com/Beka01247/kwaaka-tz/internal/parser"
	"github.com/Beka01247/kwaaka-tz/internal/queue"
	"github.com/Beka01247/kwaaka-tz/internal/repo"
	"github.com/Beka01247/kwaaka-tz/internal/store/mongo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type ParsingService struct {
	parsingTaskRepo repo.ParsingTaskRepository
	menuRepo        repo.MenuRepository
	parser          *parser.GoogleSheetsParser
	broker          queue.Broker
	storage         *mongo.Storage
	logger          *zap.SugaredLogger
}

func NewParsingService(
	parsingTaskRepo repo.ParsingTaskRepository,
	menuRepo repo.MenuRepository,
	parser *parser.GoogleSheetsParser,
	broker queue.Broker,
	storage *mongo.Storage,
	logger *zap.SugaredLogger,
) *ParsingService {
	return &ParsingService{
		parsingTaskRepo: parsingTaskRepo,
		menuRepo:        menuRepo,
		parser:          parser,
		broker:          broker,
		storage:         storage,
		logger:          logger,
	}
}

func (s *ParsingService) CreateParsingTask(ctx context.Context, spreadsheetID, restaurantName string) (primitive.ObjectID, error) {
	// create parsing task
	task := &domain.ParsingTask{
		Status:         domain.StatusQueued,
		SpreadsheetID:  spreadsheetID,
		RestaurantName: restaurantName,
		RetryCount:     0,
	}

	if err := s.parsingTaskRepo.Create(ctx, task); err != nil {
		return primitive.NilObjectID, fmt.Errorf("failed to create parsing task: %w", err)
	}

	// publish message to queue
	message := domain.MenuParsingMessage{
		TaskID:         task.ID.Hex(),
		SpreadsheetID:  spreadsheetID,
		RestaurantName: restaurantName,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := s.broker.Publish(ctx, queue.QueueMenuParsing, messageBytes); err != nil {
		// update task status to failed
		_ = s.parsingTaskRepo.UpdateStatus(ctx, task.ID, domain.StatusFailed, err.Error())
		return primitive.NilObjectID, fmt.Errorf("failed to publish message: %w", err)
	}

	s.logger.Infow("parsing task created", "task_id", task.ID.Hex(), "spreadsheet_id", spreadsheetID)

	return task.ID, nil
}

func (s *ParsingService) GetTaskStatus(ctx context.Context, taskID primitive.ObjectID) (*domain.ParsingTask, error) {
	task, err := s.parsingTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get parsing task: %w", err)
	}

	return task, nil
}

func (s *ParsingService) ProcessParsingTask(ctx context.Context, taskID primitive.ObjectID) error {
	// get task
	task, err := s.parsingTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// update status to processing
	if err := s.parsingTaskRepo.UpdateStatus(ctx, taskID, domain.StatusProcessing, ""); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	s.logger.Infow("processing parsing task", "task_id", taskID.Hex())

	// parse menu from Google Sheets
	menu, err := s.parser.ParseMenu(ctx, task.SpreadsheetID, task.RestaurantName)
	if err != nil {
		s.logger.Errorw("failed to parse menu", "task_id", taskID.Hex(), "error", err)
		_ = s.parsingTaskRepo.UpdateStatus(ctx, taskID, domain.StatusFailed, err.Error())
		return fmt.Errorf("failed to parse menu: %w", err)
	}

	// Use transaction to save menu and update task atomically
	session, err := s.storage.StartSession()
	if err != nil {
		s.logger.Errorw("failed to start session", "task_id", taskID.Hex(), "error", err)
		_ = s.parsingTaskRepo.UpdateStatus(ctx, taskID, domain.StatusFailed, "failed to start transaction")
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	err = session.StartTransaction()
	if err != nil {
		s.logger.Errorw("failed to start transaction", "task_id", taskID.Hex(), "error", err)
		_ = s.parsingTaskRepo.UpdateStatus(ctx, taskID, domain.StatusFailed, "failed to start transaction")
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// save menu to database
	if err := s.menuRepo.Create(ctx, menu); err != nil {
		s.logger.Errorw("failed to save menu", "task_id", taskID.Hex(), "error", err)
		session.AbortTransaction(ctx)
		_ = s.parsingTaskRepo.UpdateStatus(ctx, taskID, domain.StatusFailed, err.Error())
		return fmt.Errorf("failed to save menu: %w", err)
	}

	// update task with menu ID and status
	if err := s.parsingTaskRepo.UpdateWithMenuID(ctx, taskID, menu.ID, domain.StatusCompleted); err != nil {
		s.logger.Errorw("failed to update task", "task_id", taskID.Hex(), "error", err)
		session.AbortTransaction(ctx)
		return fmt.Errorf("failed to update task: %w", err)
	}

	// commit transaction
	if err := session.CommitTransaction(ctx); err != nil {
		s.logger.Errorw("failed to commit transaction", "task_id", taskID.Hex(), "error", err)
		_ = s.parsingTaskRepo.UpdateStatus(ctx, taskID, domain.StatusFailed, "failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Infow("parsing task completed", "task_id", taskID.Hex(), "menu_id", menu.ID.Hex())

	return nil
}
