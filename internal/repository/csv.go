package repository

import (
	"context"
	"errors"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
)

type CSVRepository struct {
	filePath string
}

func NewCSVRepository(filePath string) (*CSVRepository, error) {
	return &CSVRepository{filePath: filePath}, nil
}

func (r *CSVRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}
	// TODO: Implement FindByEmail
	return nil, errors.New("not implemented")
}

func (r *CSVRepository) FindByID(ctx context.Context, id int) (*domain.User, error) {
	if id <= 0 {
		return nil, errors.New("id must be greater than zero")
	}
	// TODO: Implement FindByID
	return nil, errors.New("not implemented")
}

func (r *CSVRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	return nil
}

func (r *CSVRepository) Close() error {
	return nil
}
