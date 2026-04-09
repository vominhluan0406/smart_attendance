package service

import (
	"log"
	"net"

	"github.com/smart-attendance/shared/dto"
)

type IPValidator struct{}

func NewIPValidator() *IPValidator {
	return &IPValidator{}
}

// Validate checks if the given IP address matches any of the branch's IP whitelist entries.
// Returns true if whitelist is empty (no restriction configured -- logs warning).
func (v *IPValidator) Validate(ipStr string, whitelist []dto.IPWhitelist) bool {
	if len(whitelist) == 0 {
		log.Printf("[service][ip-validator] WARNING: IP method enabled but whitelist is empty -- allowing all IPs")
		return true
	}

	// Strip port if present
	host, _, err := net.SplitHostPort(ipStr)
	if err == nil {
		ipStr = host
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Printf("[service][ip-validator] invalid IP: %s", ipStr)
		return false
	}

	for _, entry := range whitelist {
		// Try as CIDR
		_, network, err := net.ParseCIDR(entry.IPCIDR)
		if err == nil {
			if network.Contains(ip) {
				return true
			}
			continue
		}

		// Try as single IP
		entryIP := net.ParseIP(entry.IPCIDR)
		if entryIP != nil && entryIP.Equal(ip) {
			return true
		}
	}

	log.Printf("[service][ip-validator] IP %s not in whitelist (%d entries)", ipStr, len(whitelist))
	return false
}
