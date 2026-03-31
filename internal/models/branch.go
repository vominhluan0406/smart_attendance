package models

// CheckInMethod represents allowed check-in methods for a branch.
type CheckInMethod string

const (
	MethodQRTOTP   CheckInMethod = "qr_totp"
	MethodIP       CheckInMethod = "ip"
	MethodLocation CheckInMethod = "location"
	MethodFace     CheckInMethod = "face"
)

type Branch struct {
	BaseModel
	Name           string   `gorm:"type:text;not null" json:"name"`
	Address        string   `gorm:"type:text" json:"address"`
	Lat            *float64 `gorm:"type:real" json:"lat,omitempty"`
	Lng            *float64 `gorm:"type:real" json:"lng,omitempty"`
	RadiusM        int      `gorm:"default:200" json:"radius_m"`
	TOTPSecret     string   `gorm:"type:text" json:"-"`
	AllowedMethods string   `gorm:"type:text;not null;default:'qr_totp,ip,location'" json:"allowed_methods"`
	WorkStartTime  string   `gorm:"type:text;default:'08:00'" json:"work_start_time"` // Deprecated: use WorkShift
	WorkEndTime    string   `gorm:"type:text;default:'17:00'" json:"work_end_time"`   // Deprecated: use WorkShift
	IsActive       bool     `gorm:"default:true" json:"is_active"`

	// Relations
	IPWhitelist  []BranchIPWhitelist `gorm:"foreignKey:BranchID" json:"ip_whitelist,omitempty"`
	Locations    []BranchLocation    `gorm:"foreignKey:BranchID" json:"locations,omitempty"`
	Shifts       []WorkShift         `gorm:"foreignKey:BranchID" json:"shifts,omitempty"`
	Departments  []Department        `gorm:"foreignKey:BranchID" json:"departments,omitempty"`
}

type BranchIPWhitelist struct {
	BaseModel
	BranchID string `gorm:"type:text;not null;index" json:"branch_id"`
	IPCIDR   string `gorm:"type:text;not null" json:"ip_cidr"`
	Label    string `gorm:"type:text" json:"label"`
}

type BranchLocation struct {
	BaseModel
	BranchID string  `gorm:"type:text;not null;index" json:"branch_id"`
	Label    string  `gorm:"type:text" json:"label"`
	Lat      float64 `gorm:"type:real;not null" json:"lat"`
	Lng      float64 `gorm:"type:real;not null" json:"lng"`
	RadiusM  int     `gorm:"default:200" json:"radius_m"`
}
