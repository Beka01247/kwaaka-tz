package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/Beka01247/kwaaka-tz/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MenuRepository struct {
	collection *mongo.Collection
}

func NewMenuRepository(db *mongo.Database) *MenuRepository {
	return &MenuRepository{
		collection: db.Collection("menus"),
	}
}

func (r *MenuRepository) Create(ctx context.Context, menu *domain.Menu) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if menu.ID.IsZero() {
		menu.ID = primitive.NewObjectID()
	}
	menu.CreatedAt = time.Now()
	menu.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, menu)
	if err != nil {
		return fmt.Errorf("failed to create menu: %w", err)
	}

	return nil
}

func (r *MenuRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Menu, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var menu domain.Menu
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&menu)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("menu not found")
		}
		return nil, fmt.Errorf("failed to get menu: %w", err)
	}

	return &menu, nil
}

func (r *MenuRepository) GetByRestaurantID(ctx context.Context, restaurantID string) (*domain.Menu, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var menu domain.Menu
	err := r.collection.FindOne(ctx, bson.M{"restaurant_id": restaurantID}).Decode(&menu)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("menu not found")
		}
		return nil, fmt.Errorf("failed to get menu: %w", err)
	}

	return &menu, nil
}

func (r *MenuRepository) Update(ctx context.Context, menu *domain.Menu) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	menu.UpdatedAt = time.Now()

	filter := bson.M{"_id": menu.ID}
	update := bson.M{
		"$set": menu,
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update menu: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("menu not found")
	}

	return nil
}

func (r *MenuRepository) UpdateProductStatus(ctx context.Context, menuID primitive.ObjectID, productID string, status string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"_id":         menuID,
		"products.id": productID,
	}
	update := bson.M{
		"$set": bson.M{
			"products.$.status": status,
			"updated_at":        time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update product status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("menu or product not found")
	}

	return nil
}

func (r *MenuRepository) FindMenuByProductID(ctx context.Context, productID string) (*domain.Menu, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var menu domain.Menu
	filter := bson.M{"products.id": productID}
	err := r.collection.FindOne(ctx, filter).Decode(&menu)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("menu with product not found")
		}
		return nil, fmt.Errorf("failed to find menu by product: %w", err)
	}

	return &menu, nil
}

func (r *MenuRepository) UpdateProductStatusByProductID(ctx context.Context, productID string, status string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"products.id": productID}
	update := bson.M{
		"$set": bson.M{
			"products.$.status": status,
			"updated_at":        time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update product status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("product not found")
	}

	if result.ModifiedCount == 0 {
		// Product found but status didn't change (already at that status)
		return nil
	}

	return nil
}

func (r *MenuRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete menu: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("menu not found")
	}

	return nil
}
