package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Beka01247/kwaaka-tz/docs"
	"github.com/Beka01247/kwaaka-tz/internal/queue"
	"github.com/Beka01247/kwaaka-tz/internal/ratelimiter"
	"github.com/Beka01247/kwaaka-tz/internal/repo"
	"github.com/Beka01247/kwaaka-tz/internal/service"
	"github.com/Beka01247/kwaaka-tz/internal/store/mongo"
	"github.com/Beka01247/kwaaka-tz/internal/worker"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
)

type application struct {
	config         config
	logger         *zap.SugaredLogger
	rateLimiter    ratelimiter.Limiter
	storage        *mongo.Storage
	broker         queue.Broker
	menuRepo       repo.MenuRepository
	parsingService *service.ParsingService
	productService *service.ProductService
	menuWorker     *worker.MenuParsingWorker
	productWorker  *worker.ProductStatusWorker
}

type config struct {
	addr        string
	env         string
	apiURL      string
	rateLimiter ratelimiter.Config
	mongo       mongoConfig
	rabbitMQ    rabbitMQConfig
	googleCreds string
}

type mongoConfig struct {
	URI      string
	Database string
	Timeout  time.Duration
}

type rabbitMQConfig struct {
	URL           string
	MaxRetries    int
	RetryDelay    time.Duration
	PrefetchCount int
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", app.healthCheckHandler)

		r.Post("/parse", app.createParseTaskHandler)
		r.Get("/parse/{task_id}", app.getParseTaskHandler)

		r.Get("/menu/{menu_id}", app.getMenuHandler)

		r.Patch("/products/{product_id}/status", app.updateProductStatusHandler)

		docsURL := fmt.Sprintf("%s/swagger/doc.json", app.config.addr)
		r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL(docsURL)))
	})

	return r
}

func (app *application) run(mux http.Handler) error {
	// docs
	docs.SwaggerInfo.Title = "Kwaaka Tech"
	docs.SwaggerInfo.Description = "API for Kwaaka Tech"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = app.config.apiURL
	docs.SwaggerInfo.BasePath = "/api/v1"

	// workers
	if app.menuWorker != nil {
		if err := app.menuWorker.Start(); err != nil {
			return fmt.Errorf("failed to start menu worker: %w", err)
		}
	}
	if app.productWorker != nil {
		if err := app.productWorker.Start(); err != nil {
			return fmt.Errorf("failed to start product worker: %w", err)
		}
	}

	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	shutdown := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		app.logger.Infow("signal caught", "signal", s.String())

		if app.menuWorker != nil {
			app.menuWorker.Stop()
		}
		if app.productWorker != nil {
			app.productWorker.Stop()
		}

		if app.storage != nil {
			if err := app.storage.Close(ctx); err != nil {
				app.logger.Errorw("error closing MongoDB", "error", err)
			} else {
				app.logger.Info("MongoDB connection closed gracefully")
			}
		}

		if app.broker != nil {
			if err := app.broker.Close(); err != nil {
				app.logger.Errorw("error closing RabbitMQ", "error", err)
			} else {
				app.logger.Info("RabbitMQ connection closed gracefully")
			}
		}

		shutdown <- srv.Shutdown(ctx)
	}()

	app.logger.Infow("server have started", "addr", app.config.addr, "env", app.config.env)

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdown
	if err != nil {
		return err
	}

	app.logger.Infow("server has stopped", "addr", app.config.addr, "env", app.config.env)

	return nil
}
