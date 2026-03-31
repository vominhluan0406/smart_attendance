package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/smart-attendance/smart-attendance/internal/config"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailExists        = errors.New("email already registered")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrAccountDisabled    = errors.New("account is disabled")
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type Claims struct {
	UserID   string      `json:"user_id"`
	Email    string      `json:"email"`
	Role     models.Role `json:"role"`
	BranchID *string     `json:"branch_id,omitempty"`
	jwt.RegisteredClaims
}

type AuthService struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

type RegisterInput struct {
	Email    string      `json:"email"`
	Password string      `json:"password"`
	FullName string      `json:"full_name"`
	Role     models.Role `json:"role"`
}

func (s *AuthService) Register(input RegisterInput) (*models.User, error) {
	if _, err := s.userRepo.FindByEmail(input.Email); err == nil {
		return nil, ErrEmailExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	role := input.Role
	if role == "" {
		role = models.RoleEmployee
	}

	user := &models.User{
		Email:        input.Email,
		PasswordHash: string(hash),
		FullName:     input.FullName,
		Role:         role,
		IsActive:     true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *AuthService) Login(input LoginInput) (*TokenPair, *models.User, error) {
	user, err := s.userRepo.FindByEmail(input.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}

	if !user.IsActive {
		return nil, nil, ErrAccountDisabled
	}

	if user.PasswordHash == "" {
		return nil, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	tokens, err := s.generateTokenPair(user)
	if err != nil {
		return nil, nil, err
	}

	return tokens, user, nil
}

// OAuthUserInfo holds user info extracted from an OAuth provider.
type OAuthUserInfo struct {
	Provider string
	OAuthID  string
	Email    string
	FullName string
}

// LoginOrCreateFromOAuth finds an existing user by OAuth ID or email,
// or creates a new one. Then generates a JWT token pair.
func (s *AuthService) LoginOrCreateFromOAuth(info OAuthUserInfo) (*TokenPair, *models.User, error) {
	// 1. Try to find by OAuth ID
	user, err := s.userRepo.FindByOAuthID(info.Provider, info.OAuthID)
	if err == nil {
		if !user.IsActive {
			return nil, nil, ErrAccountDisabled
		}
		tokens, err := s.generateTokenPair(user)
		return tokens, user, err
	}

	// 2. Try to find by email (link existing account)
	user, err = s.userRepo.FindByEmail(info.Email)
	if err == nil {
		if !user.IsActive {
			return nil, nil, ErrAccountDisabled
		}
		// Link OAuth to existing account
		user.OAuthProvider = info.Provider
		user.OAuthID = info.OAuthID
		if err := s.userRepo.Update(user); err != nil {
			return nil, nil, fmt.Errorf("link oauth to user: %w", err)
		}
		tokens, err := s.generateTokenPair(user)
		return tokens, user, err
	}

	// 3. Create new user (no password, OAuth-only)
	user = &models.User{
		Email:         info.Email,
		FullName:      info.FullName,
		Role:          models.RoleEmployee,
		IsActive:      true,
		OAuthProvider: info.Provider,
		OAuthID:       info.OAuthID,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, nil, fmt.Errorf("create oauth user: %w", err)
	}

	tokens, err := s.generateTokenPair(user)
	return tokens, user, err
}

func (s *AuthService) RefreshToken(refreshToken string) (*TokenPair, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	return s.generateTokenPair(user)
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *AuthService) generateTokenPair(user *models.User) (*TokenPair, error) {
	now := time.Now()

	// Access token
	accessClaims := &Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Role:     user.Role,
		BranchID: user.BranchID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.cfg.JWTExpireMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID,
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}
 
	// Refresh token
	refreshClaims := &Claims{
		UserID:   user.ID,
		Email:    user.Email,
		Role:     user.Role,
		BranchID: user.BranchID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.cfg.JWTRefreshHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   user.ID,
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    s.cfg.JWTExpireMinutes * 60,
	}, nil
}
