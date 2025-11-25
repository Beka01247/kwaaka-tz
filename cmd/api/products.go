package main

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi"
)

var (
	ErrInvalidID = errors.New("invalid ID format")
)

type UpdateProductStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=available not_available deleted"`
	Reason string `json:"reason"`
	UserID string `json:"user_id,omitempty"`
}

// updateProductStatusHandler godoc
//
//	@Summary		Update product status
//	@Description	Update the status of a product
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Param			product_id	path		string						true	"Product ID"
//	@Param			request		body		UpdateProductStatusRequest	true	"Status update request"
//	@Success		202			{object}	map[string]interface{}
//	@Failure		400			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Router			/products/{product_id}/status [patch]
func (app *application) updateProductStatusHandler(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "product_id")
	if productID == "" {
		app.badRequestResponse(w, r, errors.New("product_id is required"))
		return
	}

	var req UpdateProductStatusRequest
	if err := readJson(w, r, &req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(req); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// use default user_id if not provided
	userID := req.UserID
	if userID == "" {
		userID = "admin_123"
	}

	if err := app.productService.UpdateProductStatus(r.Context(), productID, req.Status, req.Reason, userID); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Status update queued",
	}

	if err := app.jsonRespone(w, http.StatusAccepted, response); err != nil {
		app.internalServerError(w, r, err)
	}
}
