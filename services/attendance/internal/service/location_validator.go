package service

import (
	"log"
	"math"

	"github.com/smart-attendance/shared/dto"
)

const earthRadiusM = 6371000.0

type LocationValidator struct{}

func NewLocationValidator() *LocationValidator {
	return &LocationValidator{}
}

// Validate checks if the given lat/lng is within any of the branch's allowed locations.
// Returns true if locations list is empty (no restriction configured -- logs warning).
func (v *LocationValidator) Validate(lat, lng float64, locations []dto.BranchLocation) bool {
	if len(locations) == 0 {
		log.Printf("[service][location-validator] WARNING: Location method enabled but no locations configured -- allowing all positions")
		return true
	}

	for _, loc := range locations {
		dist := haversine(lat, lng, loc.Lat, loc.Lng)
		if dist <= float64(loc.RadiusM) {
			return true
		}
	}

	log.Printf("[service][location-validator] position (%.6f, %.6f) not within any allowed location (%d zones)", lat, lng, len(locations))
	return false
}

// haversine calculates the great-circle distance in meters between two points.
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := toRadians(lat2 - lat1)
	dLng := toRadians(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * c
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180
}
