package repo

import (
	"context"

	"github.com/Beka01247/kwaaka-tz/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ParsingTaskRepository interface {
	Create(ctx context.Context, task *domain.ParsingTask) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.ParsingTask, error)
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status domain.ParsingTaskStatus, errorMsg string) error
	UpdateWithMenuID(ctx context.Context, id primitive.ObjectID, menuID primitive.ObjectID, status domain.ParsingTaskStatus) error
	IncrementRetryCount(ctx context.Context, id primitive.ObjectID) error
}
