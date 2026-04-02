package models

import "github.com/go-webauthn/webauthn/webauthn"

type UserCredential struct {
	BaseModel
	UserID              string `gorm:"type:text;index;not null" json:"user_id"`
	CredentialID        []byte `gorm:"type:blob;uniqueIndex;not null" json:"credential_id"`
	PublicKey           []byte `gorm:"type:blob;not null" json:"public_key"`
	AttestationType     string `gorm:"type:text" json:"attestation_type"`
	AuthenticatorAAGUID []byte `gorm:"type:blob;column:authenticator_aaguid" json:"authenticator_aaguid"`
	SignCount           uint32 `gorm:"type:integer" json:"sign_count"`
	Transport           string `gorm:"type:text" json:"transport"` // Comma-separated list
}

// ToWebAuthn converts the model to a webauthn.Credential
func (c *UserCredential) ToWebAuthn() webauthn.Credential {
	return webauthn.Credential{
		ID:              c.CredentialID,
		PublicKey:       c.PublicKey,
		AttestationType: c.AttestationType,
		Transport:       nil, // Parse c.Transport if needed
		Authenticator: webauthn.Authenticator{
			AAGUID:    c.AuthenticatorAAGUID,
			SignCount: c.SignCount,
		},
	}
}
