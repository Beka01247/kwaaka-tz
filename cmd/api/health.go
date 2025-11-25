package main

import (
	"net/http"
	"time"
)

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// healthcheckHandler godoc
//
//	@Summary		Healthcheck
//	@Description	Healthcheck endpoint
//	@Tags			ops
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func (app *application) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// check db
	dbStatus := "ok"
	if err := app.storage.Ping(r.Context()); err != nil {
		dbStatus = "error"
	}

	// check broker: assume ok if app is running
	queueStatus := "ok"

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services: map[string]string{
			"database": dbStatus,
			"queue":    queueStatus,
		},
	}

	// if any service is down, mark as unhealthy
	if dbStatus != "ok" || queueStatus != "ok" {
		response.Status = "unhealthy"
		if err := writeJson(w, http.StatusServiceUnavailable, response); err != nil {
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := writeJson(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
	}
}
