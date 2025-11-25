package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CreateParseTaskRequest struct {
	SpreadsheetID  string `json:"spreadsheet_id" validate:"required"`
	RestaurantName string `json:"restaurant_name" validate:"required"`
}

// createParseTaskHandler godoc
//
//	@Summary		Create menu parsing task
//	@Description	Creates a new menu parsing task from Google Sheets
//	@Tags			parsing
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateParseTaskRequest	true	"Parse task request"
//	@Success		201		{object}	map[string]string
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/parse [post]
func (app *application) createParseTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateParseTaskRequest
	if err := readJson(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	taskID, err := app.parsingService.CreateParsingTask(r.Context(), req.SpreadsheetID, req.RestaurantName)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	response := map[string]string{
		"task_id": taskID.Hex(),
		"status":  "queued",
	}

	if err := app.jsonRespone(w, http.StatusCreated, response); err != nil {
		app.internalServerError(w, r, err)
	}
}

// getParseTaskHandler godoc
//
//	@Summary		Get parsing task status
//	@Description	Get the status of a menu parsing task
//	@Tags			parsing
//	@Produce		json
//	@Param			task_id	path		string	true	"Task ID"
//	@Success		200		{object}	domain.ParsingTask
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/parse/{task_id} [get]
func (app *application) getParseTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskIDStr := chi.URLParam(r, "task_id")
	if taskIDStr == "" {
		app.badRequestResponse(w, r, ErrInvalidID)
		return
	}

	taskID, err := primitive.ObjectIDFromHex(taskIDStr)
	if err != nil {
		app.badRequestResponse(w, r, ErrInvalidID)
		return
	}

	task, err := app.parsingService.GetTaskStatus(r.Context(), taskID)
	if err != nil {
		app.notFoundError(w, r, err)
		return
	}

	if err := app.jsonRespone(w, http.StatusOK, task); err != nil {
		app.internalServerError(w, r, err)
	}
}
