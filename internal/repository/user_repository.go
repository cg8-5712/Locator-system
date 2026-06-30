package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"locator/internal/model"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return nil, nil
	}

	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", trimmed).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, fmt.Errorf("get user %s: %w", trimmed, err)
	}

	return &user, nil
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *UserRepository) List(ctx context.Context, page int, pageSize int) ([]model.User, int64, error) {
	var (
		users []model.User
		total int64
	)

	query := r.db.WithContext(ctx).Model(&model.User{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	query = query.Order("created_at DESC").Order("id DESC")
	if pageSize > 0 {
		offset := 0
		if page > 1 {
			offset = (page - 1) * pageSize
		}
		query = query.Offset(offset).Limit(pageSize)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}

	return users, total, nil
}

func (r *UserRepository) Update(ctx context.Context, userID uint64, updates map[string]any) (*model.User, error) {
	result := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("update user %d: %w", userID, result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrUserNotFound
	}

	var user model.User
	if err := r.db.WithContext(ctx).Where("id = ?", userID).Take(&user).Error; err != nil {
		return nil, fmt.Errorf("load updated user %d: %w", userID, err)
	}

	return &user, nil
}

func (r *UserRepository) Delete(ctx context.Context, userID uint64) error {
	result := r.db.WithContext(ctx).Delete(&model.User{}, "id = ?", userID)
	if result.Error != nil {
		return fmt.Errorf("delete user %d: %w", userID, result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}
