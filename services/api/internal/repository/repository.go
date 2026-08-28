package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
)

var ErrNotFound = errors.New("record not found")

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
}

type IdentityRepository interface {
	Create(ctx context.Context, identity *domain.Identity) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Identity, error)
	GetByEkaID(ctx context.Context, ekaID string) (*domain.Identity, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Identity, error)
	List(ctx context.Context, limit, offset int) ([]*domain.Identity, int, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateVerificationLevel(ctx context.Context, id uuid.UUID, level string, verifiedAt *time.Time) error
}

type ProfileRepository interface {
	Create(ctx context.Context, profile *domain.Profile) error
	GetByIdentityID(ctx context.Context, identityID uuid.UUID) (*domain.Profile, error)
	FindByPhone(ctx context.Context, phone string) ([]*domain.Profile, error)
	FindByEmail(ctx context.Context, email string) ([]*domain.Profile, error)
	FindPotentialDuplicates(ctx context.Context, legalName, dob, phone, email string) ([]*domain.Profile, error)
	Update(ctx context.Context, profile *domain.Profile) error
}

type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Organization, error)
	List(ctx context.Context) ([]*domain.Organization, error)
}

type VerificationRepository interface {
	CreateRequest(ctx context.Context, req *domain.VerificationRequest) error
	GetRequestByID(ctx context.Context, id uuid.UUID) (*domain.VerificationRequest, error)
	ListRequestsByIdentity(ctx context.Context, identityID uuid.UUID) ([]*domain.VerificationRequest, error)
	ListRequestsByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.VerificationRequest, error)
	UpdateRequestStatus(ctx context.Context, id uuid.UUID, status string, approvedAt *time.Time) error
	CreateResult(ctx context.Context, result *domain.VerificationResult) error
	GetResultByRequestID(ctx context.Context, requestID uuid.UUID) (*domain.VerificationResult, error)
}

type QRRepository interface {
	CreateToken(ctx context.Context, qrToken *domain.QRToken) error
	GetToken(ctx context.Context, token string) (*domain.QRToken, error)
	MarkTokenUsed(ctx context.Context, token string) error
}

type CredentialRepository interface {
	Create(ctx context.Context, cred *domain.Credential) error
	ListByIdentityID(ctx context.Context, identityID uuid.UUID) ([]*domain.Credential, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Credential, error)
}

type AuditRepository interface {
	Record(ctx context.Context, event *domain.AuditEvent) error
	List(ctx context.Context, limit, offset int, actorID *uuid.UUID, resourceID string) ([]*domain.AuditEvent, int, error)
}

type DuplicateRepository interface {
	CreateFlag(ctx context.Context, flag *domain.DuplicateFlag) error
	ListPending(ctx context.Context) ([]*domain.DuplicateFlag, error)
	ResolveFlag(ctx context.Context, id uuid.UUID, status string, reviewerID uuid.UUID) error
}