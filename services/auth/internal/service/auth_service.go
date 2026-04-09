package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/smart-attendance/auth-service/internal/config"
	"github.com/smart-attendance/auth-service/internal/model"
	"github.com/smart-attendance/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const maxActiveSessions = 3

// Claims represents the JWT token claims.
type Claims struct {
	jwt.RegisteredClaims
	UserID   string  `json:"user_id"`
	Email    string  `json:"email"`
	FullName string  `json:"full_name"`
	Role     string  `json:"role"`
	BranchID *string `json:"branch_id,omitempty"`
}

type AuthService struct {
	cfg       *config.Config
	userRepo  *repository.UserRepository
	sessRepo  *repository.SessionRepository
}

func NewAuthService(cfg *config.Config, userRepo *repository.UserRepository, sessRepo *repository.SessionRepository) *AuthService {
	return &AuthService{
		cfg:      cfg,
		userRepo: userRepo,
		sessRepo: sessRepo,
	}
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	User         *model.User
}

// Login authenticates a user by email and password, returning JWT tokens.
func (s *AuthService) Login(email, password, ipAddress, userAgent string) (*LoginResult, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		log.Printf("[auth][service][auth] login failed: user not found, email=%s", email)
		return nil, errors.New("invalid email or password")
	}

	if !user.IsActive {
		log.Printf("[auth][service][auth] login failed: user inactive, email=%s", email)
		return nil, errors.New("account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		log.Printf("[auth][service][auth] login failed: invalid password, email=%s", email)
		return nil, errors.New("invalid email or password")
	}

	// Enforce concurrent session limit
	activeCount, err := s.sessRepo.CountActive(user.ID)
	if err != nil {
		log.Printf("[auth][service][auth] login warning: count sessions failed, user_id=%s, err=%v", user.ID, err)
	}
	if activeCount >= maxActiveSessions {
		log.Printf("[auth][service][auth] session limit reached, revoking oldest: user_id=%s, active=%d", user.ID, activeCount)
		if err := s.sessRepo.RevokeOldest(user.ID); err != nil {
			log.Printf("[auth][service][auth] failed to revoke oldest session: %v", err)
		}
	}

	// Generate access token
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		log.Printf("[auth][service][auth] login failed: generate access token, user_id=%s, err=%v", user.ID, err)
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		log.Printf("[auth][service][auth] login failed: generate refresh token, user_id=%s, err=%v", user.ID, err)
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Store session
	tokenHash := hashToken(refreshToken)
	session := &model.UserSession{
		UserID:       user.ID,
		TokenHash:    tokenHash,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		ExpiresAt:    time.Now().Add(time.Duration(s.cfg.JWTRefreshHours) * time.Hour),
		LastActiveAt: time.Now(),
	}
	if err := s.sessRepo.Create(session); err != nil {
		log.Printf("[auth][service][auth] login warning: create session failed, user_id=%s, err=%v", user.ID, err)
	}

	log.Printf("[auth][service][auth] login success: user_id=%s, email=%s, role=%s", user.ID, user.Email, user.Role)

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

// ValidateToken validates a JWT access token and returns the claims.
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// RefreshToken generates a new access token from a valid refresh token.
func (s *AuthService) RefreshToken(refreshToken string) (string, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		log.Printf("[auth][service][auth] refresh failed: invalid token")
		return "", errors.New("invalid refresh token")
	}

	// Check session exists and is not revoked
	tokenHash := hashToken(refreshToken)
	session, err := s.sessRepo.FindByToken(tokenHash)
	if err != nil {
		log.Printf("[auth][service][auth] refresh failed: session not found or revoked")
		return "", errors.New("session expired or revoked")
	}

	// Update last active
	_ = s.sessRepo.UpdateLastActive(session.TokenHash)

	// Fetch fresh user data
	user, err := s.userRepo.FindByID(claims.UserID)
	if err != nil {
		log.Printf("[auth][service][auth] refresh failed: user not found, user_id=%s", claims.UserID)
		return "", errors.New("user not found")
	}

	if !user.IsActive {
		log.Printf("[auth][service][auth] refresh failed: user inactive, user_id=%s", claims.UserID)
		return "", errors.New("account is deactivated")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}

	log.Printf("[auth][service][auth] token refreshed: user_id=%s", claims.UserID)
	return accessToken, nil
}

// Logout revokes the session associated with the given refresh token.
func (s *AuthService) Logout(refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	if err := s.sessRepo.RevokeByToken(tokenHash); err != nil {
		log.Printf("[auth][service][auth] logout failed: err=%v", err)
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

// LogoutAll revokes all sessions for a user.
func (s *AuthService) LogoutAll(userID string) error {
	if err := s.sessRepo.RevokeByUser(userID); err != nil {
		log.Printf("[auth][service][auth] logout all failed: user_id=%s, err=%v", userID, err)
		return fmt.Errorf("logout all: %w", err)
	}
	return nil
}

func (s *AuthService) generateAccessToken(user *model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.cfg.JWTExpireMinutes) * time.Minute)),
			Issuer:    "auth-service",
		},
		UserID:   user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		Role:     string(user.Role),
		BranchID: user.BranchID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *AuthService) generateRefreshToken(user *model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.cfg.JWTRefreshHours) * time.Hour)),
			Issuer:    "auth-service-refresh",
		},
		UserID:   user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		Role:     string(user.Role),
		BranchID: user.BranchID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", h)
}
