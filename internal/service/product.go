package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Beka01247/kwaaka-tz/internal/domain"
	"github.com/Beka01247/kwaaka-tz/internal/queue"
	"github.com/Beka01247/kwaaka-tz/internal/repo"
	"github.com/Beka01247/kwaaka-tz/internal/store/mongo"
	"go.uber.org/zap"
)

type ProductService struct {
	menuRepo  repo.MenuRepository
	auditRepo repo.ProductStatusAuditRepository
	broker    queue.Broker
	storage   *mongo.Storage
	logger    *zap.SugaredLogger
}

func NewProductService(
	menuRepo repo.MenuRepository,
	auditRepo repo.ProductStatusAuditRepository,
	broker queue.Broker,
	storage *mongo.Storage,
	logger *zap.SugaredLogger,
) *ProductService {
	return &ProductService{
		menuRepo:  menuRepo,
		auditRepo: auditRepo,
		broker:    broker,
		storage:   storage,
		logger:    logger,
	}
}

func (s *ProductService) UpdateProductStatus(ctx context.Context, productID, newStatus, reason, userID string) error {
	// find menu containing this product to get current status
	menu, err := s.menuRepo.FindMenuByProductID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to find product: %w", err)
	}

	// find product to get current status
	var oldStatus string
	found := false
	for _, product := range menu.Products {
		if product.ID == productID {
			oldStatus = product.Status
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("product not found")
	}

	// publish status change event (worker will update DB)
	event := domain.ProductStatusEvent{
		EventType: domain.EventProductStatusChanged,
		ProductID: productID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Reason:    reason,
		UserID:    userID,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := s.broker.Publish(ctx, queue.QueueProductStatus, eventBytes); err != nil {
		s.logger.Errorw("failed to publish status change event", "product_id", productID, "error", err)
		return fmt.Errorf("failed to publish event: %w", err)
	}

	s.logger.Infow("product status change queued", "product_id", productID, "old_status", oldStatus, "new_status", newStatus)

	return nil
}

func (s *ProductService) ProcessProductStatusEvent(ctx context.Context, event domain.ProductStatusEvent) error {
	session, err := s.storage.StartSession()
	if err != nil {
		s.logger.Errorw("failed to start session", "product_id", event.ProductID, "error", err)
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	err = session.StartTransaction()
	if err != nil {
		s.logger.Errorw("failed to start transaction", "product_id", event.ProductID, "error", err)
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// update product status in database
	if err := s.menuRepo.UpdateProductStatusByProductID(ctx, event.ProductID, event.NewStatus); err != nil {
		s.logger.Errorw("failed to update product status", "product_id", event.ProductID, "error", err)
		session.AbortTransaction(ctx)
		return fmt.Errorf("failed to update product status: %w", err)
	}

	s.logger.Infow("product status updated in DB", "product_id", event.ProductID, "new_status", event.NewStatus)

	// create audit record
	audit := &domain.ProductStatusAudit{
		ProductID: event.ProductID,
		EventType: event.EventType,
		OldStatus: event.OldStatus,
		NewStatus: event.NewStatus,
		Reason:    event.Reason,
		UserID:    event.UserID,
		Timestamp: event.Timestamp,
	}

	if err := s.auditRepo.Create(ctx, audit); err != nil {
		s.logger.Errorw("failed to create audit record", "product_id", event.ProductID, "error", err)
		session.AbortTransaction(ctx)
		return fmt.Errorf("failed to create audit record: %w", err)
	}

	// commit transaction
	if err := session.CommitTransaction(ctx); err != nil {
		s.logger.Errorw("failed to commit transaction", "product_id", event.ProductID, "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Infow("product status audit created", "product_id", event.ProductID, "event_type", event.EventType)

	return nil
}

func (s *ProductService) GetProductAudit(ctx context.Context, productID string, limit int) ([]domain.ProductStatusAudit, error) {
	audits, err := s.auditRepo.GetByProductID(ctx, productID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get product audit: %w", err)
	}

	return audits, nil
}
