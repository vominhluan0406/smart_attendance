package service

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/smart-attendance/organization-service/internal/model"
	"github.com/smart-attendance/organization-service/internal/repository"
	"github.com/smart-attendance/shared/event"
	"gorm.io/gorm"
)

var (
	ErrBranchNotFound = errors.New("branch not found")
)

type BranchService struct {
	branchRepo *repository.BranchRepository
	cache      *cache.Cache
	eventBus   *event.Bus
}

func NewBranchService(branchRepo *repository.BranchRepository, eventBus *event.Bus) *BranchService {
	return &BranchService{
		branchRepo: branchRepo,
		cache:      cache.New(5*time.Minute, 10*time.Minute),
		eventBus:   eventBus,
	}
}

// CreateBranchInput holds the input for creating a branch.
type CreateBranchInput struct {
	Name             string   `json:"name"`
	Address          string   `json:"address"`
	Lat              *float64 `json:"lat"`
	Lng              *float64 `json:"lng"`
	RadiusM          int      `json:"radius_m"`
	AllowedMethods   string   `json:"allowed_methods"`
	WorkStartTime    string   `json:"work_start_time"`
	WorkEndTime      string   `json:"work_end_time"`
	RequireBiometric bool     `json:"require_biometric"`
}

// Create creates a new branch with a generated TOTP secret.
func (s *BranchService) Create(input CreateBranchInput) (*model.Branch, error) {
	if input.AllowedMethods == "" {
		input.AllowedMethods = "qr_totp,ip,location"
	}
	if input.WorkStartTime == "" {
		input.WorkStartTime = "08:00"
	}
	if input.WorkEndTime == "" {
		input.WorkEndTime = "17:00"
	}
	if input.RadiusM <= 0 {
		input.RadiusM = 200
	}

	branch := &model.Branch{
		Name:             input.Name,
		Address:          input.Address,
		Lat:              input.Lat,
		Lng:              input.Lng,
		RadiusM:          input.RadiusM,
		RequireBiometric: input.RequireBiometric,
		AllowedMethods:   input.AllowedMethods,
		TOTPSecret:       generateTOTPSecret(),
		WorkStartTime:    input.WorkStartTime,
		WorkEndTime:      input.WorkEndTime,
		IsActive:         true,
	}

	if err := s.branchRepo.Create(branch); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}

	log.Printf("[service][branch] created branch %s (%s)", branch.ID, branch.Name)
	return branch, nil
}

// GetByID returns a branch by ID with preloaded IP whitelist and locations.
func (s *BranchService) GetByID(id string) (*model.Branch, error) {
	branch, err := s.branchRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBranchNotFound
		}
		return nil, err
	}
	return branch, nil
}

// GetByIDCached returns branch with config from cache (IP whitelist, locations).
func (s *BranchService) GetByIDCached(id string) (*model.Branch, error) {
	cacheKey := "branch:" + id
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*model.Branch), nil
	}

	branch, err := s.branchRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBranchNotFound
		}
		return nil, err
	}

	s.cache.Set(cacheKey, branch, 5*time.Minute)
	return branch, nil
}

// ListAll returns all active branches.
func (s *BranchService) ListAll() ([]model.Branch, error) {
	return s.branchRepo.ListAll()
}

// List returns a paginated list of branches.
func (s *BranchService) List(params repository.BranchListParams) (*repository.BranchListResult, error) {
	return s.branchRepo.List(params)
}

// UpdateBranchInput holds the input for updating a branch.
type UpdateBranchInput struct {
	Name             string   `json:"name"`
	Address          string   `json:"address"`
	Lat              *float64 `json:"lat"`
	Lng              *float64 `json:"lng"`
	RadiusM          int      `json:"radius_m"`
	AllowedMethods   string   `json:"allowed_methods"`
	WorkStartTime    string   `json:"work_start_time"`
	WorkEndTime      string   `json:"work_end_time"`
	IsActive         *bool    `json:"is_active"`
	RequireBiometric *bool    `json:"require_biometric"`
}

// Update updates a branch and invalidates cache.
func (s *BranchService) Update(id string, input UpdateBranchInput) (*model.Branch, error) {
	branch, err := s.branchRepo.FindByIDSimple(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBranchNotFound
		}
		return nil, err
	}

	if input.Name != "" {
		branch.Name = input.Name
	}
	if input.Address != "" {
		branch.Address = input.Address
	}
	if input.Lat != nil {
		branch.Lat = input.Lat
	}
	if input.Lng != nil {
		branch.Lng = input.Lng
	}
	if input.RadiusM > 0 {
		branch.RadiusM = input.RadiusM
	}

	// Always update AllowedMethods from input as it's the full set of methods
	log.Printf("[service][branch] updating branch %s: OldMethods=%s NewMethods=%s", id, branch.AllowedMethods, input.AllowedMethods)
	branch.AllowedMethods = input.AllowedMethods

	if input.WorkStartTime != "" {
		branch.WorkStartTime = input.WorkStartTime
	}
	if input.WorkEndTime != "" {
		branch.WorkEndTime = input.WorkEndTime
	}
	if input.IsActive != nil {
		branch.IsActive = *input.IsActive
	}
	if input.RequireBiometric != nil {
		branch.RequireBiometric = *input.RequireBiometric
	}

	if err := s.branchRepo.Update(branch); err != nil {
		return nil, err
	}

	s.invalidateCache(id)
	s.publishBranchEvent(id, "updated")
	log.Printf("[service][branch] updated branch %s (%s)", branch.ID, branch.Name)
	return branch, nil
}

// Delete soft-deletes a branch and invalidates cache.
func (s *BranchService) Delete(id string) error {
	if err := s.branchRepo.Delete(id); err != nil {
		return err
	}
	s.invalidateCache(id)
	s.publishBranchEvent(id, "deleted")
	log.Printf("[service][branch] deleted branch %s", id)
	return nil
}

func (s *BranchService) publishBranchEvent(branchID, action string) {
	if s.eventBus == nil {
		return
	}
	subject := event.SubjectBranchUpdated
	if action == "deleted" {
		subject = event.SubjectBranchDeleted
	}
	s.eventBus.Publish(subject, event.BranchEvent{
		BranchID:  branchID,
		Action:    action,
		Timestamp: time.Now(),
	})
}

// HasMethod checks if a branch allows a specific check-in method.
func HasMethod(branch *model.Branch, method model.CheckInMethod) bool {
	methods := strings.Split(branch.AllowedMethods, ",")
	for _, m := range methods {
		if strings.TrimSpace(m) == string(method) {
			return true
		}
	}
	return false
}

// --- IP Whitelist ---

// UpdateIPWhitelist replaces all IP whitelist entries for a branch.
func (s *BranchService) UpdateIPWhitelist(branchID string, ips []model.BranchIPWhitelist) error {
	if err := s.branchRepo.ReplaceIPWhitelist(branchID, ips); err != nil {
		return err
	}
	s.invalidateCache(branchID)
	s.publishBranchEvent(branchID, "updated")
	log.Printf("[service][branch] updated IP whitelist for branch %s (%d entries)", branchID, len(ips))
	return nil
}

// --- Locations ---

// UpdateLocations replaces all location entries for a branch.
func (s *BranchService) UpdateLocations(branchID string, locs []model.BranchLocation) error {
	if err := s.branchRepo.ReplaceLocations(branchID, locs); err != nil {
		return err
	}
	s.invalidateCache(branchID)
	s.publishBranchEvent(branchID, "updated")
	log.Printf("[service][branch] updated locations for branch %s (%d entries)", branchID, len(locs))
	return nil
}

// --- TOTP ---

// RegenerateTOTPSecret generates a new TOTP secret for a branch.
func (s *BranchService) RegenerateTOTPSecret(branchID string) (string, error) {
	branch, err := s.branchRepo.FindByIDSimple(branchID)
	if err != nil {
		return "", ErrBranchNotFound
	}
	branch.TOTPSecret = generateTOTPSecret()
	if err := s.branchRepo.Update(branch); err != nil {
		return "", err
	}
	s.invalidateCache(branchID)
	log.Printf("[service][branch] regenerated TOTP secret for branch %s", branchID)
	return branch.TOTPSecret, nil
}

// --- Helpers ---

func (s *BranchService) invalidateCache(branchID string) {
	s.cache.Delete("branch:" + branchID)
}

func generateTOTPSecret() string {
	secret := make([]byte, 20)
	rand.Read(secret)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
}
