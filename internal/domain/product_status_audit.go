package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductStatusAudit struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID string             `bson:"product_id" json:"product_id"`
	EventType string             `bson:"event_type" json:"event_type"`
	OldStatus string             `bson:"old_status" json:"old_status"`
	NewStatus string             `bson:"new_status" json:"new_status"`
	Reason    string             `bson:"reason" json:"reason"`
	UserID    string             `bson:"user_id" json:"user_id"`
	Timestamp time.Time          `bson:"timestamp" json:"timestamp"`
}
