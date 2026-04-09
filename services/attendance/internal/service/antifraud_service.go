package service

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/smart-attendance/attendance-service/internal/model"
	"github.com/smart-attendance/attendance-service/internal/repository"
)

// Anti-fraud error sentinels
var (
	ErrGPSAccuracyTooLow  = fmt.Errorf("do chinh xac GPS qua thap, nghi ngo gia mao vi tri")
	ErrGPSAccuracyTooHigh = fmt.Errorf("do chinh xac GPS qua cao (>150m), vi tri khong dang tin cay")
	ErrTOTPAlreadyUsed    = fmt.Errorf("ma QR da duoc su dung, vui long doi ma moi")
	ErrImpossibleTravel   = fmt.Errorf("phat hien di chuyen bat thuong, vui long thu lai")
	ErrDeviceBlocked      = fmt.Errorf("thiet bi nay da bi chan, vui long lien he quan ly")
	ErrIPLocationMismatch = fmt.Errorf("vi tri IP va GPS khong khop, nghi ngo su dung VPN")
)

const (
	minGPSAccuracyM        = 10.0  // Below this = likely spoofed (too precise)
	maxGPSAccuracyM        = 150.0 // Above this = too imprecise
	maxTravelSpeedKMH      = 150.0 // Max plausible travel speed
	ipLocationMaxDistKM    = 500.0 // Max distance between IP geo and GPS
	totpNonceTTL           = 30 * time.Second
	lastCheckinCacheTTL    = 24 * time.Hour
	anomalyZScoreThreshold = 3.0
)

// lastCheckinData is cached per user to detect impossible travel.
type lastCheckinData struct {
	Lat  float64
	Lng  float64
	Time time.Time
}

type AntiFraudService struct {
	cache      *gocache.Cache
	logRepo    *repository.AttendanceLogRepository
	deviceRepo *repository.UserDeviceRepository
	alertRepo  *repository.FraudAlertRepository
}

func NewAntiFraudService(
	c *gocache.Cache,
	logRepo *repository.AttendanceLogRepository,
	deviceRepo *repository.UserDeviceRepository,
	alertRepo *repository.FraudAlertRepository,
) *AntiFraudService {
	return &AntiFraudService{
		cache:      c,
		logRepo:    logRepo,
		deviceRepo: deviceRepo,
		alertRepo:  alertRepo,
	}
}

// --- Feature 1: GPS Accuracy Check ---

func (s *AntiFraudService) ValidateGPSAccuracy(accuracyM *float64) error {
	if accuracyM == nil {
		return nil // No accuracy data provided, skip check
	}
	if *accuracyM < minGPSAccuracyM {
		log.Printf("[antifraud] GPS accuracy too low: %.1fm (min: %.1fm) -- possible mock GPS", *accuracyM, minGPSAccuracyM)
		return ErrGPSAccuracyTooLow
	}
	if *accuracyM > maxGPSAccuracyM {
		log.Printf("[antifraud] GPS accuracy too high: %.1fm (max: %.1fm) -- unreliable position", *accuracyM, maxGPSAccuracyM)
		return ErrGPSAccuracyTooHigh
	}
	return nil
}

// --- Feature 2: TOTP Single-Use Nonce ---

func (s *AntiFraudService) CheckTOTPNonce(branchID, code string) error {
	if code == "" {
		return nil
	}
	key := fmt.Sprintf("totp_nonce:%s:%s", branchID, code)
	if _, found := s.cache.Get(key); found {
		log.Printf("[antifraud] TOTP code reused: branch=%s code=%s", branchID, code)
		return ErrTOTPAlreadyUsed
	}
	return nil
}

func (s *AntiFraudService) MarkTOTPUsed(branchID, code string) {
	if code == "" {
		return
	}
	key := fmt.Sprintf("totp_nonce:%s:%s", branchID, code)
	s.cache.Set(key, true, totpNonceTTL)
}

// --- Feature 4: Impossible Travel Detection ---

func (s *AntiFraudService) CheckImpossibleTravel(userID string, lat, lng float64, now time.Time) error {
	cacheKey := fmt.Sprintf("last_checkin:%s", userID)

	var last *lastCheckinData

	// Try cache first
	if val, found := s.cache.Get(cacheKey); found {
		last = val.(*lastCheckinData)
	} else {
		// Fallback: query last log with GPS coordinates
		lastLog, err := s.logRepo.FindLastWithLocation(userID)
		if err == nil && lastLog != nil && lastLog.Lat != nil && lastLog.Lng != nil {
			last = &lastCheckinData{
				Lat:  *lastLog.Lat,
				Lng:  *lastLog.Lng,
				Time: lastLog.LoggedAt,
			}
		}
	}

	// Update cache with current position
	s.cache.Set(cacheKey, &lastCheckinData{Lat: lat, Lng: lng, Time: now}, lastCheckinCacheTTL)

	if last == nil {
		return nil // First check-in, nothing to compare
	}

	// Calculate distance and speed
	distM := Haversine(last.Lat, last.Lng, lat, lng)
	elapsed := now.Sub(last.Time)
	if elapsed < time.Second {
		return nil // Same-second log, skip
	}

	speedKMH := (distM / 1000.0) / elapsed.Hours()

	if speedKMH > maxTravelSpeedKMH && distM > 1000 { // Only flag if distance > 1km
		log.Printf("[antifraud] impossible travel: user=%s dist=%.0fm elapsed=%v speed=%.0fkm/h",
			userID, distM, elapsed, speedKMH)
		return ErrImpossibleTravel
	}

	return nil
}

// --- Feature 5: Device Fingerprinting ---

func (s *AntiFraudService) CheckDevice(userID, fingerprint, userAgent string) (isNew bool, err error) {
	if fingerprint == "" {
		return false, nil
	}

	hash := hashFingerprint(fingerprint)

	device, err := s.deviceRepo.FindByUserAndFingerprint(userID, hash)
	if err == nil && device != nil {
		// Known device
		if device.IsBlocked {
			log.Printf("[antifraud] blocked device: user=%s fp=%s", userID, hash[:8])
			return false, ErrDeviceBlocked
		}
		// Update last seen
		device.LastSeenAt = time.Now()
		s.deviceRepo.Update(device)
		return false, nil
	}

	// New device -- auto-trust it but flag as new
	newDevice := &model.UserDevice{
		UserID:          userID,
		FingerprintHash: hash,
		UserAgent:       truncateStr(userAgent, 500),
		DeviceName:      parseDeviceName(userAgent),
		LastSeenAt:      time.Now(),
		IsTrusted:       true, // Auto-trust, alert only
	}
	s.deviceRepo.Create(newDevice)

	log.Printf("[antifraud] new device detected: user=%s device=%s", userID, hash[:8])
	return true, nil
}

// --- Feature 6: IP-Location Cross-Check ---
// Uses a simple heuristic: if IP is a private address, skip the check.
// For public IPs, we compare against known branch location as a proxy.

func (s *AntiFraudService) CheckIPLocationConsistency(ip string, lat, lng float64, branchLat, branchLng float64) error {
	// Skip for private/local IPs
	if isPrivateIP(ip) {
		return nil
	}

	// Simple heuristic: if GPS claims to be at branch but IP is clearly not local,
	// we compare the claimed GPS distance to branch vs a threshold.
	distGPSToBranch := Haversine(lat, lng, branchLat, branchLng)
	if distGPSToBranch > ipLocationMaxDistKM*1000 {
		log.Printf("[antifraud] IP-location mismatch: ip=%s gps=(%.4f,%.4f) branch=(%.4f,%.4f) dist=%.0fkm",
			ip, lat, lng, branchLat, branchLng, distGPSToBranch/1000)
		return ErrIPLocationMismatch
	}
	return nil
}

// --- Feature 8: Check-in Pattern Anomaly Detection ---

func (s *AntiFraudService) CheckTimeAnomaly(userID string, checkInTime time.Time) (isAnomaly bool, score float64) {
	cacheKey := fmt.Sprintf("anomaly_stats:%s", userID)

	type stats struct {
		Mean   float64
		StdDev float64
		Count  int
	}

	var s2 *stats
	if val, found := s.cache.Get(cacheKey); found {
		s2 = val.(*stats)
	} else {
		// Compute from recent logs (last 30 days)
		logs, err := s.logRepo.FindRecentFirstLogs(userID, 30)
		if err != nil || len(logs) < 5 {
			return false, 0 // Not enough data
		}

		// Convert to minutes since midnight
		var minutes []float64
		for _, l := range logs {
			m := float64(l.LoggedAt.Hour()*60 + l.LoggedAt.Minute())
			minutes = append(minutes, m)
		}

		mean := average(minutes)
		stddev := stdDev(minutes, mean)
		s2 = &stats{Mean: mean, StdDev: stddev, Count: len(minutes)}
		s.cache.Set(cacheKey, s2, 1*time.Hour)
	}

	if s2.StdDev < 1 || s2.Count < 5 {
		return false, 0 // Not enough variance
	}

	currentMinutes := float64(checkInTime.Hour()*60 + checkInTime.Minute())
	zScore := math.Abs(currentMinutes-s2.Mean) / s2.StdDev

	if zScore > anomalyZScoreThreshold {
		log.Printf("[antifraud] time anomaly: user=%s time=%s mean=%.0fmin stddev=%.0fmin zscore=%.1f",
			userID, checkInTime.Format("15:04"), s2.Mean, s2.StdDev, zScore)
		return true, zScore
	}

	return false, zScore
}

// --- Feature: Create Fraud Alert ---

func (s *AntiFraudService) CreateAlert(userID, branchID string, alertType model.FraudAlertType, severity, description string, details map[string]interface{}, ip string, lat, lng *float64) {
	detailsJSON, _ := json.Marshal(details)
	alert := &model.FraudAlert{
		UserID:      userID,
		BranchID:    branchID,
		AlertType:   alertType,
		Severity:    severity,
		Description: description,
		Details:     string(detailsJSON),
		IPAddress:   ip,
		Lat:         lat,
		Lng:         lng,
	}
	if err := s.alertRepo.Create(alert); err != nil {
		log.Printf("[antifraud] ERROR creating alert: %v", err)
	}
}

// --- Helpers ---

// Haversine calculates great-circle distance in meters (exported for reuse).
func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusMeter = 6371000.0
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeter * c
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}

func hashFingerprint(fp string) string {
	h := sha256.Sum256([]byte(fp))
	return fmt.Sprintf("%x", h)
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

func parseDeviceName(ua string) string {
	if len(ua) > 100 {
		return ua[:100]
	}
	return ua
}

func isPrivateIP(ip string) bool {
	// Simple check for common private IP ranges
	return ip == "" ||
		ip == "127.0.0.1" ||
		ip == "::1" ||
		(len(ip) >= 3 && ip[:3] == "10.") ||
		(len(ip) >= 8 && ip[:8] == "192.168.") ||
		(len(ip) >= 4 && ip[:4] == "172.")
}

func average(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func stdDev(data []float64, mean float64) float64 {
	if len(data) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range data {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(data)-1))
}
