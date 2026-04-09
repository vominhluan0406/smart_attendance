package model

// UserCredential stores WebAuthn/Passkey credentials for biometric authentication.
type UserCredential struct {
	BaseModel
	UserID              string `gorm:"type:uuid;index;not null" json:"user_id"`
	CredentialID        []byte `gorm:"type:bytea;uniqueIndex;not null" json:"credential_id"`
	PublicKey           []byte `gorm:"type:bytea;not null" json:"public_key"`
	AttestationType     string `gorm:"type:varchar(50)" json:"attestation_type"`
	AuthenticatorAAGUID []byte `gorm:"type:bytea;column:authenticator_aaguid" json:"authenticator_aaguid"`
	SignCount           uint32 `gorm:"type:integer" json:"sign_count"`
	BackupEligible      bool   `gorm:"type:boolean" json:"backup_eligible"`
	BackupState         bool   `gorm:"type:boolean" json:"backup_state"`
	IsApproved          bool   `gorm:"type:boolean;default:false" json:"is_approved"`
	Transport           string `gorm:"type:varchar(255)" json:"transport"` // Comma-separated list
}

func (UserCredential) TableName() string {
	return "user_credentials"
}
