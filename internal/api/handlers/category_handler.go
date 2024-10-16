package handlers

import (
	"landmark-api/internal/services"
	"log"
	"net/http"
)

type CategoryHandler struct {
	categoryService services.CategoryService
}

func NewCategoryHandler(categoryService services.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
	}
}

func (h *CategoryHandler) ListAdminCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	categories, err := h.categoryService.GetAllCategories(ctx)
	if err != nil {
		log.Printf("Error fetching categories: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Error fetching categories")
		return
	}

	response := map[string]interface{}{
		"categories": categories,
		"total":      len(categories),
	}

	respondWithJSON(w, http.StatusOK, response)
}
