package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"locator/internal/model"
	"locator/internal/repository"
	pkgjwt "locator/pkg/jwt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAuthDisabled       = errors.New("authentication is disabled")
)

type AuthConfig struct {
	Enabled                bool
	JWTSecret              string
	TokenTTL               time.Duration
	BootstrapAdminUsername string
	BootstrapAdminPassword string
}

type LoginInput struct {
	Username string
	Password string
}

type LoginResult struct {
	Token   string      `json:"token"`
	Expires time.Time   `json:"expires_at"`
	User    UserSummary `json:"user"`
}

type UserSummary struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type AuthService struct {
	userRepo *repository.UserRepository
	cfg      AuthConfig
}

func NewAuthService(userRepo *repository.UserRepository, cfg AuthConfig) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

func (s *AuthService) Enabled() bool {
	return s != nil && s.cfg.Enabled
}

func (s *AuthService) ParseToken(token string) (*pkgjwt.Claims, error) {
	if s == nil {
		return nil, ErrAuthDisabled
	}

	if !s.cfg.Enabled {
		return nil, ErrAuthDisabled
	}

	return pkgjwt.ParseHS256(token, s.cfg.JWTSecret, time.Now().UTC())
}

func (s *AuthService) EnsureBootstrapAdmin(ctx context.Context) error {
	if !s.cfg.Enabled {
		return nil
	}

	if strings.TrimSpace(s.cfg.BootstrapAdminUsername) == "" || strings.TrimSpace(s.cfg.BootstrapAdminPassword) == "" {
		return nil
	}

	existing, err := s.userRepo.GetByUsername(ctx, s.cfg.BootstrapAdminUsername)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.BootstrapAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.userRepo.Create(ctx, &model.User{
		Username:     strings.TrimSpace(s.cfg.BootstrapAdminUsername),
		PasswordHash: string(hash),
		Role:         "admin",
	})
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	if !s.cfg.Enabled {
		return nil, ErrAuthDisabled
	}

	username := strings.TrimSpace(input.Username)
	password := input.Password
	if username == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	expires := now.Add(s.cfg.TokenTTL)
	token, err := pkgjwt.GenerateHS256(pkgjwt.Claims{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		IssuedAt:  now.Unix(),
		ExpiresAt: expires.Unix(),
	}, s.cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:   token,
		Expires: expires,
		User: UserSummary{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	}, nil
}
