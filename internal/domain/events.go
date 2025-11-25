package domain

import "time"

type MenuParsingMessage struct {
	TaskID         string `json:"task_id"`
	SpreadsheetID  string `json:"spreadsheet_id"`
	RestaurantName string `json:"restaurant_name"`
}

type ProductStatusEvent struct {
	EventType string    `json:"event_type"`
	ProductID string    `json:"product_id"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"user_id"`
}

const (
	EventProductCreated       = "product.created"
	EventProductUpdated       = "product.updated"
	EventProductStatusChanged = "product.status_changed"
	EventProductDeleted       = "product.deleted"
)
