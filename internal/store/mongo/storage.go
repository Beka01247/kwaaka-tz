package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Storage struct {
	client   *mongo.Client
	database *mongo.Database
	config   Config
}

type Config struct {
	URI      string
	Database string
	Timeout  time.Duration
}

func New(cfg Config) (*Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(100).
		SetMinPoolSize(10)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	database := client.Database(cfg.Database)

	return &Storage{
		client:   client,
		database: database,
		config:   cfg,
	}, nil
}

func (s *Storage) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *Storage) Ping(ctx context.Context) error {
	return s.client.Ping(ctx, readpref.Primary())
}

func (s *Storage) Database() *mongo.Database {
	return s.database
}

func (s *Storage) Client() *mongo.Client {
	return s.client
}

func (s *Storage) StartSession() (mongo.Session, error) {
	return s.client.StartSession()
}

func (s *Storage) CreateIndexes(ctx context.Context) error {
	// create indexes for menus collection
	menusIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "restaurant_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "products.id", Value: 1}},
		},
	}
	if _, err := s.database.Collection("menus").Indexes().CreateMany(ctx, menusIndexes); err != nil {
		return fmt.Errorf("failed to create menus indexes: %w", err)
	}

	// create indexes for parsing_tasks collection
	tasksIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "status", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: 1}},
		},
	}
	if _, err := s.database.Collection("parsing_tasks").Indexes().CreateMany(ctx, tasksIndexes); err != nil {
		return fmt.Errorf("failed to create parsing_tasks indexes: %w", err)
	}

	// create indexes for product_status_audit collection
	auditIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "product_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "timestamp", Value: 1}},
		},
	}
	if _, err := s.database.Collection("product_status_audit").Indexes().CreateMany(ctx, auditIndexes); err != nil {
		return fmt.Errorf("failed to create product_status_audit indexes: %w", err)
	}

	return nil
}
