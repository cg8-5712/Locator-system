package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"locator/internal/model"
	"locator/internal/repository"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUsernameConflict   = errors.New("username already exists")
	ErrInvalidUserRole    = errors.New("invalid user role")
	ErrInvalidPassword    = errors.New("password must be at least 6 characters")
	ErrInvalidUsername    = errors.New("username is required")
	ErrNoUserFieldChange  = errors.New("no user fields to update")
)

type UserListResult struct {
	Users      []UserSummary `json:"users"`
	Pagination Pagination    `json:"pagination"`
}

type UserListQuery struct {
	Page     int
	PageSize int
}

type UserCreateInput struct {
	Username string
	Password string
	Role     string
}

type UserUpdateInput struct {
	Password *string
	Role     *string
}

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) ListUsers(ctx context.Context, query UserListQuery) (*UserListResult, error) {
	page := normalizePage(query.Page)
	pageSize := normalizePageSize(query.PageSize, 50, 200)

	users, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	result := make([]UserSummary, 0, len(users))
	for _, user := range users {
		result = append(result, UserSummary{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		})
	}

	return &UserListResult{
		Users:      result,
		Pagination: buildPagination(page, pageSize, total),
	}, nil
}

func (s *UserService) CreateUser(ctx context.Context, input UserCreateInput) (*UserSummary, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return nil, ErrInvalidUsername
	}

	role := normalizeRole(input.Role)
	if role == "" {
		return nil, ErrInvalidUserRole
	}

	if len(input.Password) < 6 {
		return nil, ErrInvalidPassword
	}

	existing, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUsernameConflict
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &UserSummary{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID uint64, input UserUpdateInput) (*UserSummary, error) {
	updates := make(map[string]any)

	if input.Role != nil {
		role := normalizeRole(*input.Role)
		if role == "" {
			return nil, ErrInvalidUserRole
		}
		updates["role"] = role
	}

	if input.Password != nil {
		if len(strings.TrimSpace(*input.Password)) < 6 {
			return nil, ErrInvalidPassword
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(*input.Password)), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		updates["password_hash"] = string(hash)
	}

	if len(updates) == 0 {
		return nil, ErrNoUserFieldChange
	}

	user, err := s.repo.Update(ctx, userID, updates)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &UserSummary{
		ID:       user.ID,
		Username: user.Username,
		Role:     user.Role,
	}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID uint64) error {
	if err := s.repo.Delete(ctx, userID); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}

	return nil
}

func normalizeRole(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "admin":
		return "admin"
	case "user":
		return "user"
	default:
		return ""
	}
}
