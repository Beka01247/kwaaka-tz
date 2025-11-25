package main

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/Beka01247/kwaaka-tz/internal/env"
	"github.com/Beka01247/kwaaka-tz/internal/parser"
	"github.com/Beka01247/kwaaka-tz/internal/queue"
	"github.com/Beka01247/kwaaka-tz/internal/ratelimiter"
	"github.com/Beka01247/kwaaka-tz/internal/service"
	"github.com/Beka01247/kwaaka-tz/internal/store/mongo"
	"github.com/Beka01247/kwaaka-tz/internal/worker"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

const version = "0.0.0"

//	@title			Kwaaka Tech
//	@description	API for Kwaaka Tech
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath					/api/v1
//
// @securityDefinitions.apiKey	ApiKeyAuth
// @in							header
// @name						Authorization
// @description
func main() {
	_ = godotenv.Load()

	cfg := config{
		addr:   env.GetString("ADDR", ":8080"),
		apiURL: env.GetString("EXTERNAL_URL", "localhost:8080"),
		env:    env.GetString("ENV", "development"),
		rateLimiter: ratelimiter.Config{
			RequestsPerTimeFrame: env.GetInt("RATELIMITER_REQUESTS_COUNT", 20),
			TimeFrame:            time.Second * 5,
			Enabled:              env.GetBool("RATE_LIMITER_ENABLED", true),
		},
		mongo: mongoConfig{
			URI:      env.GetString("MONGO_URI", "mongodb://localhost:27017"),
			Database: env.GetString("MONGO_DATABASE", "kwaaka"),
			Timeout:  time.Second * 10,
		},
		rabbitMQ: rabbitMQConfig{
			URL:           env.GetString("RABBITMQ_URL", "amqp://admin:password@localhost:5672/"),
			MaxRetries:    env.GetInt("RABBITMQ_MAX_RETRIES", 3),
			RetryDelay:    time.Second * 2,
			PrefetchCount: env.GetInt("RABBITMQ_PREFETCH_COUNT", 10),
		},
		googleCreds: env.GetString("GOOGLE_CREDENTIALS_PATH", ""),
	}

	// logger
	logger := zap.Must(zap.NewProduction()).Sugar()
	defer logger.Sync()

	// rate limiter
	rateLimiter := ratelimiter.NewFixedWindowLimiter(
		cfg.rateLimiter.RequestsPerTimeFrame,
		cfg.rateLimiter.TimeFrame,
	)

	// storage
	storage, err := mongo.New(mongo.Config{
		URI:      cfg.mongo.URI,
		Database: cfg.mongo.Database,
		Timeout:  cfg.mongo.Timeout,
	})
	if err != nil {
		logger.Fatalw("failed to connect to MongoDB", "error", err)
	}

	logger.Info("connected to MongoDB")

	// create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := storage.CreateIndexes(ctx); err != nil {
		logger.Warnw("failed to create indexes", "error", err)
	} else {
		logger.Info("MongoDB indexes created successfully")
	}

	// repos
	menuRepo := mongo.NewMenuRepository(storage.Database())
	parsingTaskRepo := mongo.NewParsingTaskRepository(storage.Database())
	productStatusAuditRepo := mongo.NewProductStatusAuditRepository(storage.Database())

	// rabbitmq broker
	broker, err := queue.NewRabbitMQBroker(queue.Config{
		URL:           cfg.rabbitMQ.URL,
		MaxRetries:    cfg.rabbitMQ.MaxRetries,
		RetryDelay:    cfg.rabbitMQ.RetryDelay,
		PrefetchCount: cfg.rabbitMQ.PrefetchCount,
	})
	if err != nil {
		logger.Fatalw("failed to connect to RabbitMQ", "error", err)
	}

	logger.Info("connected to RabbitMQ")

	var googleParser *parser.GoogleSheetsParser
	if cfg.googleCreds != "" {
		credsJSON, err := ioutil.ReadFile(cfg.googleCreds)
		if err != nil {
			logger.Fatalw("failed to read Google credentials", "error", err)
		}

		googleParser, err = parser.New(parser.Config{
			CredentialsJSON: credsJSON,
		})
		if err != nil {
			logger.Fatalw("failed to create Google Sheets parser", "error", err)
		}
		logger.Info("Google Sheets parser initialized")
	} else {
		logger.Warn("Google credentials not provided, parsing functionality will be limited")
	}

	parsingService := service.NewParsingService(
		parsingTaskRepo,
		menuRepo,
		googleParser,
		broker,
		storage,
		logger,
	)

	productService := service.NewProductService(
		menuRepo,
		productStatusAuditRepo,
		broker,
		storage,
		logger,
	)

	menuWorker := worker.NewMenuParsingWorker(parsingService, broker, logger)
	productWorker := worker.NewProductStatusWorker(productService, broker, logger)

	app := &application{
		config:         cfg,
		logger:         logger,
		rateLimiter:    rateLimiter,
		storage:        storage,
		broker:         broker,
		menuRepo:       menuRepo,
		parsingService: parsingService,
		productService: productService,
		menuWorker:     menuWorker,
		productWorker:  productWorker,
	}

	mux := app.mount()

	logger.Fatal(app.run(mux))
}
