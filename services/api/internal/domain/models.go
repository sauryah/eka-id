package domain

import (
	"time"

	"github.com/google/uuid"
)

// Identity Lifecycle Statuses
const (
	IdentityStatusPending   = "PENDING"
	IdentityStatusActive    = "ACTIVE"
	IdentityStatusSuspended = "SUSPENDED"
	IdentityStatusRevoked   = "REVOKED"
	IdentityStatusDeceased  = "DECEASED"
)

// Verification Levels
const (
	VerificationTier0Unverified = "TIER_0_UNVERIFIED"
	VerificationTier1Basic      = "TIER_1_BASIC"
	VerificationTier2Verified   = "TIER_2_VERIFIED"
	VerificationTier3Enhanced   = "TIER_3_ENHANCED"
)

// Roles
const (
	RoleUser        = "USER"
	RoleOrgMember   = "ORG_MEMBER"
	RoleOrgAdmin    = "ORG_ADMIN"
	RoleSystemAdmin = "SYSTEM_ADMIN"
)

// Verification Request Statuses
const (
	RequestStatusPending  = "PENDING"
	RequestStatusApproved = "APPROVED"
	RequestStatusDenied   = "DENIED"
	RequestStatusExpired  = "EXPIRED"
)

// User represents an authentication principal
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone,omitempty"`
	PasswordHash string    `json:"-"` // Never serialized
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Identity represents the core universal EKA identity
// Notice: Internal ID (UUID) is strictly separated from Public EkaID (EKA-XXXX-XXXX)
type Identity struct {
	ID                uuid.UUID  `json:"id"`                 // Internal primary key
	EkaID             string     `json:"eka_id"`             // Public identifier (EKA-XXXX-XXXX)
	UserID            uuid.UUID  `json:"user_id"`
	Status            string     `json:"status"`
	VerificationLevel string     `json:"verification_level"`
	VerifiedAt        *time.Time `json:"verified_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Profile holds personal identifiable information (PII)
// Access to this struct is strictly controlled and audited
type Profile struct {
	IdentityID      uuid.UUID              `json:"identity_id"`
	LegalName       string                 `json:"legal_name"`        // PII
	DateOfBirth     string                 `json:"date_of_birth"`     // PII - Format: YYYY-MM-DD
	Gender          string                 `json:"gender,omitempty"`  // PII
	ProfilePhotoURL string                 `json:"profile_photo_url,omitempty"`
	Phone           string                 `json:"phone,omitempty"`   // PII - Restricted
	Email           string                 `json:"email,omitempty"`   // PII - Restricted
	AddressLine1    string                 `json:"address_line1,omitempty"` // PII - Restricted
	City            string                 `json:"city,omitempty"`
	State           string                 `json:"state,omitempty"`
	PostalCode      string                 `json:"postal_code,omitempty"`
	Country         string                 `json:"country,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// Organization represents an entity requesting identity verification
type Organization struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	ApiKeyHash string    `json:"-"`
	Status     string    `json:"status"`
	WebhookURL string    `json:"webhook_url,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// VerificationRequest represents a consent-driven verification interaction
type VerificationRequest struct {
	ID              uuid.UUID  `json:"id"`
	OrgID           uuid.UUID  `json:"org_id"`
	OrgName         string     `json:"org_name,omitempty"`
	IdentityID      uuid.UUID  `json:"identity_id"`
	EkaID           string     `json:"eka_id,omitempty"`
	RequestedScopes []string   `json:"requested_scopes"` // e.g. ["identity_valid", "name_match", "dob", "phone"]
	Purpose         string     `json:"purpose"`
	Status          string     `json:"status"`
	ApprovedAt      *time.Time `json:"approved_at,omitempty"`
	ExpiresAt       time.Time  `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// VerificationResult holds selective disclosure output after user approval
type VerificationResult struct {
	ID                 uuid.UUID              `json:"id"`
	RequestID          uuid.UUID              `json:"request_id"`
	DisclosedClaims    map[string]interface{} `json:"disclosed_claims"`
	Status             string                 `json:"status"`
	VerifiedByActorID  *uuid.UUID             `json:"verified_by_actor_id,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
}

// QRToken represents a short-lived, signed token for QR verification
type QRToken struct {
	ID            uuid.UUID `json:"id"`
	Token         string    `json:"token"`
	IdentityID    uuid.UUID `json:"identity_id"`
	AllowedScopes []string  `json:"allowed_scopes"`
	IsUsed        bool      `json:"is_used"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// Credential represents a verifiable claim issued to an identity
type Credential struct {
	ID                 uuid.UUID              `json:"id"`
	IdentityID         uuid.UUID              `json:"identity_id"`
	Type               string                 `json:"type"` // EDUCATION, EMPLOYMENT, LICENSE, CERTIFICATION, ADDRESS_VERIFIED
	IssuerName         string                 `json:"issuer_name"`
	Status             string                 `json:"status"`
	IssuedAt           time.Time              `json:"issued_at"`
	ExpiresAt          *time.Time             `json:"expires_at,omitempty"`
	VerificationMethod string                 `json:"verification_method"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
}

// AuditEvent represents an immutable record of system actions
type AuditEvent struct {
	EventID      uuid.UUID              `json:"event_id"`
	ActorID      *uuid.UUID             `json:"actor_id,omitempty"`
	ActorType    string                 `json:"actor_type"` // USER, ORG, ADMIN, SYSTEM
	Action       string                 `json:"action"`     // IDENTITY_CREATED, QR_VERIFIED, etc.
	ResourceType string                 `json:"resource_type"`
	ResourceID   string                 `json:"resource_id"`
	Result       string                 `json:"result"` // SUCCESS, FAILURE, DENIED
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// DuplicateFlag tracks potential duplicate identities flagged for review
type DuplicateFlag struct {
	ID                   uuid.UUID              `json:"id"`
	IdentityID           uuid.UUID              `json:"identity_id"`
	SuspectedDuplicateID uuid.UUID              `json:"suspected_duplicate_id"`
	ConfidenceScore      float64                `json:"confidence_score"`
	MatchReasons         []string               `json:"match_reasons"`
	Status               string                 `json:"status"` // PENDING_REVIEW, RESOLVED_FALSE_POSITIVE, RESOLVED_DUPLICATE
	ReviewedBy           *uuid.UUID             `json:"reviewed_by,omitempty"`
	ReviewedAt           *time.Time             `json:"reviewed_at,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	PrimaryEkaID         string                 `json:"primary_eka_id,omitempty"`
	PrimaryName          string                 `json:"primary_name,omitempty"`
	PrimaryPhoto         string                 `json:"primary_photo,omitempty"`
	SuspectedEkaID       string                 `json:"suspected_eka_id,omitempty"`
	SuspectedName        string                 `json:"suspected_name,omitempty"`
	SuspectedPhoto       string                 `json:"suspected_photo,omitempty"`
}