package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Menu struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name            string             `bson:"name" json:"name"`
	RestaurantID    string             `bson:"restaurant_id" json:"restaurant_id"`
	Products        []Product          `bson:"products" json:"products"`
	AttributeGroups []AttributeGroup   `bson:"attributes_groups" json:"attributes_groups"`
	Attributes      []Attribute        `bson:"attributes" json:"attributes"`
	CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}

type Product struct {
	ID          string   `bson:"id" json:"id"`
	Name        string   `bson:"name" json:"name"`
	IsCombo     bool     `bson:"is_combo" json:"is_combo"`
	Price       float64  `bson:"price" json:"price"`
	Category    string   `bson:"category" json:"category"`
	Description string   `bson:"description" json:"description"`
	Status      string   `bson:"status" json:"status"`
	Attributes  []string `bson:"attributes" json:"attributes"`
}

type AttributeGroup struct {
	ID         string   `bson:"id" json:"id"`
	Name       string   `bson:"name" json:"name"`
	Min        int      `bson:"min" json:"min"`
	Max        int      `bson:"max" json:"max"`
	Attributes []string `bson:"attributes" json:"attributes"`
}

type Attribute struct {
	ID    string  `bson:"id" json:"id"`
	Name  string  `bson:"name" json:"name"`
	Min   int     `bson:"min" json:"min"`
	Max   int     `bson:"max" json:"max"`
	Price float64 `bson:"price" json:"price"`
}
