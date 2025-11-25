package repo

import (
	"context"

	"github.com/Beka01247/kwaaka-tz/internal/domain"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MenuRepository interface {
	Create(ctx context.Context, menu *domain.Menu) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Menu, error)
	GetByRestaurantID(ctx context.Context, restaurantID string) (*domain.Menu, error)
	Update(ctx context.Context, menu *domain.Menu) error
	UpdateProductStatus(ctx context.Context, menuID primitive.ObjectID, productID string, status string) error
	FindMenuByProductID(ctx context.Context, productID string) (*domain.Menu, error)
	UpdateProductStatusByProductID(ctx context.Context, productID string, status string) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}
