package service

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/patrickmn/go-cache"
	"github.com/smart-attendance/auth-service/internal/config"
	"github.com/smart-attendance/auth-service/internal/model"
	"github.com/smart-attendance/auth-service/internal/repository"
)

type WebAuthnService struct {
	web       *webauthn.WebAuthn
	userRepo  *repository.UserRepository
	credRepo  *repository.CredentialRepository
	cache     *cache.Cache
}

func NewWebAuthnService(cfg *config.Config, userRepo *repository.UserRepository, credRepo *repository.CredentialRepository) (*WebAuthnService, error) {
	// Default values for development
	rpID := "localhost"
	rpOrigin := "http://localhost:3000"
	
	// Can be extended via config later
	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Smart Attendance",
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn: %w", err)
	}

	return &WebAuthnService{
		web:      w,
		userRepo: userRepo,
		credRepo: credRepo,
		cache:    cache.New(10*time.Minute, 10*time.Minute),
	}, nil
}

// Internal wrapper to satisfy webauthn.User interface
type webauthnUser struct {
	user  *model.User
	creds []model.UserCredential
}

func (u *webauthnUser) WebAuthnID() []byte {
	return []byte(u.user.ID)
}

func (u *webauthnUser) WebAuthnName() string {
	return u.user.Email
}

func (u *webauthnUser) WebAuthnDisplayName() string {
	return u.user.FullName
}

func (u *webauthnUser) WebAuthnIcon() string {
	return ""
}

func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	var result []webauthn.Credential
	for _, c := range u.creds {
		transports := []protocol.AuthenticatorTransport{}
		if c.Transport != "" {
			parts := strings.Split(c.Transport, ",")
			for _, p := range parts {
				transports = append(transports, protocol.AuthenticatorTransport(p))
			}
		}

		result = append(result, webauthn.Credential{
			ID:              c.CredentialID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Transport:       transports,
			Authenticator: webauthn.Authenticator{
				AAGUID:    c.AuthenticatorAAGUID,
				SignCount: c.SignCount,
			},
		})
	}
	return result
}

func (s *WebAuthnService) BeginRegistration(userID string) (*protocol.CredentialCreation, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	creds, _ := s.credRepo.ListByUserID(userID)
	wUser := &webauthnUser{user: user, creds: creds}

	options, sessionData, err := s.web.BeginRegistration(wUser)
	if err != nil {
		log.Printf("[auth][webauthn] error starting registration for user %s: %v", userID, err)
		return nil, err
	}

	// Store session data in cache for 10 minutes
	s.cache.Set("reg_"+userID, sessionData, cache.DefaultExpiration)

	return options, nil
}

func (s *WebAuthnService) FinishRegistration(userID string, r *http.Request) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Retrieve session data from cache
	sessionKey := "reg_" + userID
	val, found := s.cache.Get(sessionKey)
	if !found {
		return errors.New("registration session expired or not found")
	}
	sessionData := val.(*webauthn.SessionData)
	defer s.cache.Delete(sessionKey)

	// Wrap user
	creds, _ := s.credRepo.ListByUserID(userID)
	wUser := &webauthnUser{user: user, creds: creds}

	// Prepare request for parsing
	parsedResponse, err := protocol.ParseCredentialCreationResponse(r)
	if err != nil {
		return fmt.Errorf("failed to parse credential response: %w", err)
	}

	credential, err := s.web.CreateCredential(wUser, *sessionData, parsedResponse)
	if err != nil {
		log.Printf("[auth][webauthn] error finishing registration for user %s: %v", userID, err)
		return err
	}

	// Save credential to DB
	transports := []string{}
	for _, t := range credential.Transport {
		transports = append(transports, string(t))
	}

	dbCred := &model.UserCredential{
		UserID:              userID,
		CredentialID:        credential.ID,
		PublicKey:           credential.PublicKey,
		AttestationType:     credential.AttestationType,
		AuthenticatorAAGUID: credential.Authenticator.AAGUID,
		SignCount:           credential.Authenticator.SignCount,
		Transport:           strings.Join(transports, ","),
		IsApproved:          true, // Auto-approve for now
	}

	if err := s.credRepo.Create(dbCred); err != nil {
		return fmt.Errorf("failed to save credential: %w", err)
	}

	log.Printf("[auth][webauthn] registration successful for user %s", userID)
	return nil
}
