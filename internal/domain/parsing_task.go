package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ParsingTaskStatus string

const (
	StatusQueued     ParsingTaskStatus = "queued"
	StatusProcessing ParsingTaskStatus = "processing"
	StatusCompleted  ParsingTaskStatus = "completed"
	StatusFailed     ParsingTaskStatus = "failed"
)

type ParsingTask struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Status         ParsingTaskStatus   `bson:"status" json:"status"`
	SpreadsheetID  string              `bson:"spreadsheet_id" json:"spreadsheet_id"`
	RestaurantName string              `bson:"restaurant_name" json:"restaurant_name"`
	MenuID         *primitive.ObjectID `bson:"menu_id,omitempty" json:"menu_id,omitempty"`
	ErrorMessage   string              `bson:"error_message,omitempty" json:"error_message,omitempty"`
	RetryCount     int                 `bson:"retry_count" json:"retry_count"`
	CreatedAt      time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time           `bson:"updated_at" json:"updated_at"`
}
