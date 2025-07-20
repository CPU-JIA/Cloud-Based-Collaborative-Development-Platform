package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// SSOProvider represents a Single Sign-On identity provider configuration
type SSOProvider struct {
	ID               uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TenantID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Name             string         `gorm:"size:255;not null" json:"name"`
	Type             string         `gorm:"size:50;not null" json:"type"` // saml, oauth2, oidc
	Status           string         `gorm:"size:20;default:active" json:"status"` // active, inactive, disabled
	Configuration    datatypes.JSON `gorm:"type:jsonb" json:"configuration"`
	Metadata         datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	CreatedByUserID  uuid.UUID      `gorm:"type:uuid" json:"created_by_user_id"`
	ModifiedByUserID *uuid.UUID     `gorm:"type:uuid" json:"modified_by_user_id"`
}

// SSOSession represents an active SSO authentication session
type SSOSession struct {
	ID             uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TenantID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	UserID         *uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	ProviderID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"provider_id"`
	ExternalUserID string     `gorm:"size:255;not null" json:"external_user_id"`
	State          string     `gorm:"size:255;not null;index" json:"state"`
	Nonce          string     `gorm:"size:255" json:"nonce"`
	CodeChallenge  string     `gorm:"size:255" json:"code_challenge"`
	RedirectURI    string     `gorm:"size:1000" json:"redirect_uri"`
	Status         string     `gorm:"size:20;default:pending" json:"status"` // pending, completed, failed, expired
	ExpiresAt      time.Time  `gorm:"not null" json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	
	// Relations
	Provider *SSOProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
	User     *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// SSOUserMapping maps external SSO users to internal users
type SSOUserMapping struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	TenantID       uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ProviderID     uuid.UUID `gorm:"type:uuid;not null;index" json:"provider_id"`
	ExternalUserID string    `gorm:"size:255;not null" json:"external_user_id"`
	ExternalEmail  string    `gorm:"size:255" json:"external_email"`
	ExternalName   string    `gorm:"size:255" json:"external_name"`
	Attributes     datatypes.JSON `gorm:"type:jsonb" json:"attributes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	
	// Relations
	Provider *SSOProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
	User     *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// SSO Configuration Types
type SAMLConfig struct {
	EntityID              string            `json:"entity_id"`
	SSOURL               string            `json:"sso_url"`
	SLOUrl               string            `json:"slo_url"`
	Certificate          string            `json:"certificate"`
	PrivateKey           string            `json:"private_key"`
	NameIDFormat         string            `json:"name_id_format"`
	AttributeMapping     map[string]string `json:"attribute_mapping"`
	SignAuthnRequests    bool              `json:"sign_authn_requests"`
	ValidateAssertions   bool              `json:"validate_assertions"`
	RequiredAssertions   []string          `json:"required_assertions"`
}

type OAuth2Config struct {
	ClientID             string            `json:"client_id"`
	ClientSecret         string            `json:"client_secret"`
	AuthorizationURL     string            `json:"authorization_url"`
	TokenURL             string            `json:"token_url"`
	UserInfoURL          string            `json:"user_info_url"`
	RedirectURL          string            `json:"redirect_url"`
	Scopes               []string          `json:"scopes"`
	AttributeMapping     map[string]string `json:"attribute_mapping"`
	PKCEEnabled          bool              `json:"pkce_enabled"`
	StateParamRequired   bool              `json:"state_param_required"`
}

type OIDCConfig struct {
	ClientID             string            `json:"client_id"`
	ClientSecret         string            `json:"client_secret"`
	IssuerURL            string            `json:"issuer_url"`
	RedirectURL          string            `json:"redirect_url"`
	Scopes               []string          `json:"scopes"`
	AttributeMapping     map[string]string `json:"attribute_mapping"`
	PKCEEnabled          bool              `json:"pkce_enabled"`
	NonceRequired        bool              `json:"nonce_required"`
	ValidateIDToken      bool              `json:"validate_id_token"`
	AllowedIssuers       []string          `json:"allowed_issuers"`
}

// TableName methods for custom table names
func (SSOProvider) TableName() string {
	return "sso_providers"
}

func (SSOSession) TableName() string {
	return "sso_sessions"
}

func (SSOUserMapping) TableName() string {
	return "sso_user_mappings"
}