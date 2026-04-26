package categories

import "github.com/google/uuid"

type Category struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
}

type CreateCategoryRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type GetCategoryResponse struct {
	Category *Category `json:"category"`
}

type ListCategoriesResponse struct {
	Categories []*Category `json:"categories"`
}

type GetCategoriesByIDsRequest struct {
	CategoryIDs []string `json:"category_ids"`
}

type GetCategoriesByIDsResponse struct {
	Categories []*Category `json:"categories"`
}
