package dto

type Branch struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	Address        string           `json:"address,omitempty"`
	Lat            *float64         `json:"lat,omitempty"`
	Lng            *float64         `json:"lng,omitempty"`
	RadiusM        int              `json:"radius_m"`
	TOTPSecret     string           `json:"totp_secret,omitempty"`
	AllowedMethods string           `json:"allowed_methods"`
	WorkStartTime  string           `json:"work_start_time"`
	WorkEndTime    string           `json:"work_end_time"`
	IsActive       bool             `json:"is_active"`
	IPWhitelist    []IPWhitelist    `json:"ip_whitelist,omitempty"`
	Locations      []BranchLocation `json:"locations,omitempty"`
}

type IPWhitelist struct {
	ID    string `json:"id"`
	IPCIDR string `json:"ip_cidr"`
	Label  string `json:"label,omitempty"`
}

type BranchLocation struct {
	ID      string  `json:"id"`
	Label   string  `json:"label,omitempty"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	RadiusM int     `json:"radius_m"`
}
