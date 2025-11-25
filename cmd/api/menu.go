package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// getMenuHandler godoc
//
//	@Summary		Get menu by ID
//	@Description	Get menu details by menu ID
//	@Tags			menus
//	@Produce		json
//	@Param			menu_id	path		string	true	"Menu ID"
//	@Success		200		{object}	domain.Menu
//	@Failure		400		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/menu/{menu_id} [get]
func (app *application) getMenuHandler(w http.ResponseWriter, r *http.Request) {
	menuIDStr := chi.URLParam(r, "menu_id")
	if menuIDStr == "" {
		app.badRequestResponse(w, r, ErrInvalidID)
		return
	}

	menuID, err := primitive.ObjectIDFromHex(menuIDStr)
	if err != nil {
		app.badRequestResponse(w, r, ErrInvalidID)
		return
	}

	menu, err := app.menuRepo.GetByID(r.Context(), menuID)
	if err != nil {
		app.notFoundError(w, r, err)
		return
	}

	if err := app.jsonRespone(w, http.StatusOK, menu); err != nil {
		app.internalServerError(w, r, err)
	}
}
