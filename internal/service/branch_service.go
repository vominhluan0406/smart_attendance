package service

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/smart-attendance/smart-attendance/internal/cache"
	"github.com/smart-attendance/smart-attendance/internal/models"
	"github.com/smart-attendance/smart-attendance/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrBranchNotFound = errors.New("branch not found")
)

type BranchService struct {
	branchRepo *repository.BranchRepository
	userRepo   *repository.UserRepository
	cache      *cache.Cache
}

func NewBranchService(branchRepo *repository.BranchRepository, userRepo *repository.UserRepository, cache *cache.Cache) *BranchService {
	return &BranchService{branchRepo: branchRepo, userRepo: userRepo, cache: cache}
}

type CreateBranchInput struct {
	Name           string  `json:"name"`
	Address        string  `json:"address"`
	Lat            *float64 `json:"lat"`
	Lng            *float64 `json:"lng"`
	RadiusM        int     `json:"radius_m"`
	AllowedMethods string  `json:"allowed_methods"`
	WorkStartTime  string  `json:"work_start_time"`
	WorkEndTime    string  `json:"work_end_time"`
}

func (s *BranchService) Create(input CreateBranchInput) (*models.Branch, error) {
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

	branch := &models.Branch{
		Name:           input.Name,
		Address:        input.Address,
		Lat:            input.Lat,
		Lng:            input.Lng,
		RadiusM:        input.RadiusM,
		TOTPSecret:     generateTOTPSecret(),
		AllowedMethods: input.AllowedMethods,
		WorkStartTime:  input.WorkStartTime,
		WorkEndTime:    input.WorkEndTime,
		IsActive:       true,
	}

	if err := s.branchRepo.Create(branch); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}

	log.Printf("[service][branch] created branch %s (%s)", branch.ID, branch.Name)
	return branch, nil
}

func (s *BranchService) GetByID(id string) (*models.Branch, error) {
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
func (s *BranchService) GetByIDCached(id string) (*models.Branch, error) {
	cacheKey := "branch:" + id
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*models.Branch), nil
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

func (s *BranchService) List(params repository.BranchListParams) (*repository.BranchListResult, error) {
	return s.branchRepo.List(params)
}

func (s *BranchService) ListAll() ([]models.Branch, error) {
	return s.branchRepo.ListAll()
}

type UpdateBranchInput struct {
	Name           string   `json:"name"`
	Address        string   `json:"address"`
	Lat            *float64 `json:"lat"`
	Lng            *float64 `json:"lng"`
	RadiusM        int      `json:"radius_m"`
	AllowedMethods string   `json:"allowed_methods"`
	WorkStartTime  string   `json:"work_start_time"`
	WorkEndTime    string   `json:"work_end_time"`
	IsActive       *bool    `json:"is_active"`
}

func (s *BranchService) Update(id string, input UpdateBranchInput) (*models.Branch, error) {
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
	log.Printf("[service][branch] Updating branch %s: OldMethods=%s NewMethods=%s", id, branch.AllowedMethods, input.AllowedMethods)
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

	if err := s.branchRepo.Update(branch); err != nil {
		return nil, err
	}

	s.invalidateCache(id)
	log.Printf("[service][branch] updated branch %s (%s)", branch.ID, branch.Name)
	return branch, nil
}

func (s *BranchService) Delete(id string) error {
	if err := s.branchRepo.Delete(id); err != nil {
		return err
	}
	s.invalidateCache(id)
	log.Printf("[service][branch] deleted branch %s", id)
	return nil
}

// --- IP Whitelist ---

func (s *BranchService) UpdateIPWhitelist(branchID string, ips []models.BranchIPWhitelist) error {
	if err := s.branchRepo.ReplaceIPWhitelist(branchID, ips); err != nil {
		return err
	}
	s.invalidateCache(branchID)
	log.Printf("[service][branch] updated IP whitelist for branch %s (%d entries)", branchID, len(ips))
	return nil
}

// --- Locations ---

func (s *BranchService) UpdateLocations(branchID string, locs []models.BranchLocation) error {
	if err := s.branchRepo.ReplaceLocations(branchID, locs); err != nil {
		return err
	}
	s.invalidateCache(branchID)
	log.Printf("[service][branch] updated locations for branch %s (%d entries)", branchID, len(locs))
	return nil
}

// --- Employee assignment ---

func (s *BranchService) AssignEmployee(userID, branchID string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}
	user.BranchID = &branchID
	if err := s.userRepo.Update(user); err != nil {
		return err
	}
	log.Printf("[service][branch] assigned user %s to branch %s", userID, branchID)
	return nil
}

func (s *BranchService) UnassignEmployee(userID string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}
	user.BranchID = nil
	return s.userRepo.Update(user)
}

func (s *BranchService) GetEmployeeCount(branchID string) (int64, error) {
	return s.branchRepo.GetEmployeeCount(branchID)
}

// --- TOTP ---

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

// HasMethod checks if a branch allows a specific check-in method.
func (s *BranchService) HasMethod(branch *models.Branch, method models.CheckInMethod) bool {
	methods := strings.Split(branch.AllowedMethods, ",")
	for _, m := range methods {
		if strings.TrimSpace(m) == string(method) {
			return true
		}
	}
	return false
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
