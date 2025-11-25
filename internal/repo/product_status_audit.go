package repo

import (
	"context"

	"github.com/Beka01247/kwaaka-tz/internal/domain"
)

type ProductStatusAuditRepository interface {
	Create(ctx context.Context, audit *domain.ProductStatusAudit) error
	GetByProductID(ctx context.Context, productID string, limit int) ([]domain.ProductStatusAudit, error)
}
