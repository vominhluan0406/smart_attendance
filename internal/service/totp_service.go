package service

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

const (
	TOTPInterval = 15 // seconds
	TOTPDigits   = 6
)

type TOTPService struct{}

func NewTOTPService() *TOTPService {
	return &TOTPService{}
}

// GenerateCode produces the current TOTP code for a given secret.
func (s *TOTPService) GenerateCode(secret string) (string, int, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return "", 0, fmt.Errorf("decode secret: %w", err)
	}

	now := time.Now()
	counter := uint64(now.Unix()) / TOTPInterval
	code := hotp(key, counter, TOTPDigits)

	// Seconds remaining until next code
	remaining := TOTPInterval - int(now.Unix()%TOTPInterval)

	return code, remaining, nil
}

// ValidateCode checks if the provided code matches the current or previous TOTP window.
func (s *TOTPService) ValidateCode(secret, code string) (bool, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return false, fmt.Errorf("decode secret: %w", err)
	}

	now := time.Now()
	counter := uint64(now.Unix()) / TOTPInterval

	// Check current window and one previous (to handle edge cases)
	for i := uint64(0); i <= 1; i++ {
		expected := hotp(key, counter-i, TOTPDigits)
		if expected == code {
			return true, nil
		}
	}

	return false, nil
}

// hotp generates an HOTP code per RFC 4226.
func hotp(key []byte, counter uint64, digits int) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff

	mod := uint32(math.Pow10(digits))
	otp := truncated % mod

	return fmt.Sprintf("%0*d", digits, otp)
}
