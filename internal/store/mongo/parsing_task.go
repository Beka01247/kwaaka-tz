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

type ParsingTaskRepository struct {
	collection *mongo.Collection
}

func NewParsingTaskRepository(db *mongo.Database) *ParsingTaskRepository {
	return &ParsingTaskRepository{
		collection: db.Collection("parsing_tasks"),
	}
}

func (r *ParsingTaskRepository) Create(ctx context.Context, task *domain.ParsingTask) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if task.ID.IsZero() {
		task.ID = primitive.NewObjectID()
	}
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to create parsing task: %w", err)
	}

	return nil
}

func (r *ParsingTaskRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.ParsingTask, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var task domain.ParsingTask
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("parsing task not found")
		}
		return nil, fmt.Errorf("failed to get parsing task: %w", err)
	}

	return &task, nil
}

func (r *ParsingTaskRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status domain.ParsingTaskStatus, errorMsg string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	if errorMsg != "" {
		update["$set"].(bson.M)["error_message"] = errorMsg
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update parsing task status: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("parsing task not found")
	}

	return nil
}

func (r *ParsingTaskRepository) UpdateWithMenuID(ctx context.Context, id primitive.ObjectID, menuID primitive.ObjectID, status domain.ParsingTaskStatus) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"menu_id":    menuID,
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to update parsing task: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("parsing task not found")
	}

	return nil
}

func (r *ParsingTaskRepository) IncrementRetryCount(ctx context.Context, id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	update := bson.M{
		"$inc": bson.M{"retry_count": 1},
		"$set": bson.M{"updated_at": time.Now()},
	}

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("parsing task not found")
	}

	return nil
}
