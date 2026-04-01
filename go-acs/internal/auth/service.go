package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/novaacs/go-acs/internal/tenant"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInsufficientRole   = errors.New("insufficient permissions")
)

type Config struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type Service struct {
	users   Repository
	tenants tenant.Repository
	cfg     Config
	log     *zap.Logger
}

func NewService(users Repository, tenants tenant.Repository, cfg Config, log *zap.Logger) *Service {
	return &Service{users: users, tenants: tenants, cfg: cfg, log: log}
}

func (s *Service) Login(ctx context.Context, tenantSlug, email, password string) (*TokenPair, error) {
	t, err := s.tenants.GetBySlug(ctx, tenantSlug)
	if err != nil || !t.Active {
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.GetByEmail(ctx, t.ID, email)
	if err != nil {
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$12$dummy"), []byte(password))
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	s.log.Info("user login",
		zap.String("tenant", tenantSlug),
		zap.String("email", email),
		zap.String("role", string(user.Role)),
	)

	return s.issueTokenPair(user)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.validateToken(refreshToken, s.cfg.RefreshSecret)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.users.GetByID(ctx, claims.TenantID, claims.UserID)
	if err != nil || !user.Active {
		return nil, ErrInvalidToken
	}

	return s.issueTokenPair(user)
}

func (s *Service) ValidateAccessToken(tokenStr string) (*Claims, error) {
	return s.validateToken(tokenStr, s.cfg.AccessSecret)
}

func (s *Service) issueTokenPair(user *User) (*TokenPair, error) {
	now := time.Now()

	accessClaims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTTL)),
		},
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Role:     user.Role,
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.AccessSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshClaims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.RefreshTTL)),
		},
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Role:     user.Role,
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.RefreshSecret))
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.AccessTTL.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *Service) validateToken(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(b), err
}
