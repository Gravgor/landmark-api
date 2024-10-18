package services

import (
	"context"
	"landmark-api/internal/repository"
)

type CategoryService interface {
	GetAllCategories(ctx context.Context) ([]string, error)
}

type categoryService struct {
	categoryRepo repository.CategoryRepository
}

func NewCategoryService(categoryRepo repository.CategoryRepository) CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
	}
}

func (s *categoryService) GetAllCategories(ctx context.Context) ([]string, error) {
	return s.categoryRepo.ListAllCategories(ctx)
}
