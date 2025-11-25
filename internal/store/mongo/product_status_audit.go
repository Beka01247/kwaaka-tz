package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/Beka01247/kwaaka-tz/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProductStatusAuditRepository struct {
	collection *mongo.Collection
}

func NewProductStatusAuditRepository(db *mongo.Database) *ProductStatusAuditRepository {
	return &ProductStatusAuditRepository{
		collection: db.Collection("product_status_audit"),
	}
}

func (r *ProductStatusAuditRepository) Create(ctx context.Context, audit *domain.ProductStatusAudit) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if audit.ID.IsZero() {
		audit.ID = primitive.NewObjectID()
	}
	if audit.Timestamp.IsZero() {
		audit.Timestamp = time.Now()
	}

	_, err := r.collection.InsertOne(ctx, audit)
	if err != nil {
		return fmt.Errorf("failed to create product status audit: %w", err)
	}

	return nil
}

func (r *ProductStatusAuditRepository) GetByProductID(ctx context.Context, productID string, limit int) ([]domain.ProductStatusAudit, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{"product_id": productID}
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get product status audits: %w", err)
	}
	defer cursor.Close(ctx)

	var audits []domain.ProductStatusAudit
	if err := cursor.All(ctx, &audits); err != nil {
		return nil, fmt.Errorf("failed to decode product status audits: %w", err)
	}

	return audits, nil
}
