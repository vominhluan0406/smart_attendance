package service

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/smart-attendance/smart-attendance/internal/cache"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
)

type WebAuthnService struct {
	wa       *webauthn.WebAuthn
	credRepo *repository.UserCredentialRepository
	userRepo *repository.UserRepository
	cache    *cache.Cache
}

func NewWebAuthnService(
	rpID string,
	origin string,
	credRepo *repository.UserCredentialRepository,
	userRepo *repository.UserRepository,
	c *cache.Cache,
) (*WebAuthnService, error) {
	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "Smart Attendance",
		RPID:          rpID,
		RPOrigins:     []string{origin},
	})
	if err != nil {
		return nil, fmt.Errorf("init webauthn: %w", err)
	}

	return &WebAuthnService{
		wa:       w,
		credRepo: credRepo,
		userRepo: userRepo,
		cache:    c,
	}, nil
}

// BeginRegistration returns the JSON options for navigator.credentials.create
func (s *WebAuthnService) BeginRegistration(user *models.User) (*protocol.CredentialCreation, error) {
	// Pre-load credentials
	creds, err := s.credRepo.FindByUserID(user.ID)
	if err == nil {
		user.Credentials = creds
	}

	options, sessionData, err := s.wa.BeginRegistration(user)
	if err != nil {
		return nil, fmt.Errorf("webauthn begin registration: %w", err)
	}

	// Store session data in cache
	s.cache.Set("webauthn:reg:"+user.ID, sessionData, 5*time.Minute)

	return options, nil
}

// FinishRegistration verifies the response from navigator.credentials.create
func (s *WebAuthnService) FinishRegistration(user *models.User, r *http.Request) error {
	// Retrieve session data
	val, ok := s.cache.Get("webauthn:reg:" + user.ID)
	if !ok {
		return fmt.Errorf("registration session not found or expired")
	}
	sessionData := val.(*webauthn.SessionData)
	s.cache.Delete("webauthn:reg:" + user.ID)

	credential, err := s.wa.FinishRegistration(user, *sessionData, r)
	if err != nil {
		return fmt.Errorf("webauthn finish registration: %w", err)
	}

	// Save to DB
	newCred := &models.UserCredential{
		UserID:              user.ID,
		CredentialID:        credential.ID,
		PublicKey:           credential.PublicKey,
		AttestationType:     credential.AttestationType,
		AuthenticatorAAGUID: credential.Authenticator.AAGUID,
		SignCount:           credential.Authenticator.SignCount,
		BackupEligible:      credential.Flags.BackupEligible,
		BackupState:         credential.Flags.BackupState,
	}

	if err := s.credRepo.Create(newCred); err != nil {
		return fmt.Errorf("save credential: %w", err)
	}

	log.Printf("[service][webauthn] registered new credential for user %s", user.ID)
	return nil
}

// BeginLogin returns the JSON options for navigator.credentials.get
func (s *WebAuthnService) BeginLogin(user *models.User) (*protocol.CredentialAssertion, error) {
	// Pre-load credentials
	creds, err := s.credRepo.FindByUserID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("find credentials: %w", err)
	}
	if len(creds) == 0 {
		return nil, fmt.Errorf("no biometric devices registered")
	}
	user.Credentials = creds

	options, sessionData, err := s.wa.BeginLogin(user)
	if err != nil {
		return nil, fmt.Errorf("webauthn begin login: %w", err)
	}

	// Store session data in cache
	s.cache.Set("webauthn:login:"+user.ID, sessionData, 5*time.Minute)

	return options, nil
}

// FinishLogin verifies the response from navigator.credentials.get
func (s *WebAuthnService) FinishLogin(user *models.User, r *http.Request) error {
	// Retrieve session data
	val, ok := s.cache.Get("webauthn:login:" + user.ID)
	if !ok {
		return fmt.Errorf("login session not found or expired")
	}
	sessionData := val.(*webauthn.SessionData)
	s.cache.Delete("webauthn:login:" + user.ID)

	// Pre-load credentials
	creds, err := s.credRepo.FindByUserID(user.ID)
	if err != nil {
		return fmt.Errorf("find credentials: %w", err)
	}
	user.Credentials = creds

	credential, err := s.wa.FinishLogin(user, *sessionData, r)
	if err != nil {
		return fmt.Errorf("webauthn finish login: %w", err)
	}

	// Update sign count
	for i := range user.Credentials {
		if string(user.Credentials[i].CredentialID) == string(credential.ID) {
			user.Credentials[i].SignCount = credential.Authenticator.SignCount
			user.Credentials[i].BackupEligible = credential.Flags.BackupEligible
			user.Credentials[i].BackupState = credential.Flags.BackupState
			s.credRepo.Update(&user.Credentials[i])
			break
		}
	}

	log.Printf("[service][webauthn] verified assertion for user %s", user.ID)
	return nil
}
