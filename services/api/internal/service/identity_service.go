package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/crypto"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
)

var (
	ErrIdentityNotFound      = errors.New("identity not found")
	ErrMaxCollisionRetries   = errors.New("failed to generate unique EKA ID after max retries")
	ErrInvalidStatusTransition = errors.New("invalid identity status transition")
)

type IdentityService struct {
	identRepo repository.IdentityRepository
	profRepo  repository.ProfileRepository
	auditSvc  *AuditService
}

func NewIdentityService(
	identRepo repository.IdentityRepository,
	profRepo repository.ProfileRepository,
	auditSvc *AuditService,
) *IdentityService {
	return &IdentityService{
		identRepo: identRepo,
		profRepo:  profRepo,
		auditSvc:  auditSvc,
	}
}

// CreateIdentity creates a new internal identity with a collision-resistant public EKA ID
func (s *IdentityService) CreateIdentity(ctx context.Context, userID uuid.UUID, level string) (*domain.Identity, error) {
	const maxRetries = 5
	var uniqueEkaID string
	var err error

	for i := 0; i < maxRetries; i++ {
		candidateID, genErr := crypto.GenerateEkaID()
		if genErr != nil {
			return nil, fmt.Errorf("crypto generator error: %w", genErr)
		}

		// Verify uniqueness in repository
		existing, _ := s.identRepo.GetByEkaID(ctx, candidateID)
		if existing == nil {
			uniqueEkaID = candidateID
			break
		}
	}

	if uniqueEkaID == "" {
		return nil, ErrMaxCollisionRetries
	}

	if level == "" {
		level = domain.VerificationTier1Basic
	}

	now := time.Now().UTC()
	identity := &domain.Identity{
		ID:                uuid.New(),
		EkaID:             uniqueEkaID,
		UserID:            userID,
		Status:            domain.IdentityStatusActive,
		VerificationLevel: level,
		VerifiedAt:        &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err = s.identRepo.Create(ctx, identity); err != nil {
		return nil, fmt.Errorf("failed to persist identity: %w", err)
	}

	_ = s.auditSvc.Record(ctx, &userID, "USER", "IDENTITY_CREATED", "IDENTITY", identity.EkaID, "SUCCESS", "", "", "", map[string]interface{}{
		"eka_id": identity.EkaID,
		"level":  identity.VerificationLevel,
	})

	return identity, nil
}

func (s *IdentityService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Identity, error) {
	ident, err := s.identRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrIdentityNotFound
	}
	return ident, nil
}

func (s *IdentityService) GetByEkaID(ctx context.Context, ekaID string) (*domain.Identity, error) {
	cleanID, err := crypto.NormalizeEkaID(ekaID)
	if err != nil {
		return nil, err
	}

	ident, err := s.identRepo.GetByEkaID(ctx, cleanID)
	if err != nil {
		return nil, ErrIdentityNotFound
	}
	return ident, nil
}

func (s *IdentityService) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Identity, error) {
	ident, err := s.identRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, ErrIdentityNotFound
	}
	return ident, nil
}

func (s *IdentityService) UpdateStatus(ctx context.Context, id uuid.UUID, newStatus string, actorID *uuid.UUID) error {
	validStatuses := map[string]bool{
		domain.IdentityStatusActive:    true,
		domain.IdentityStatusSuspended: true,
		domain.IdentityStatusRevoked:   true,
		domain.IdentityStatusDeceased:  true,
	}

	if !validStatuses[newStatus] {
		return ErrInvalidStatusTransition
	}

	ident, err := s.identRepo.GetByID(ctx, id)
	if err != nil {
		return ErrIdentityNotFound
	}

	if err := s.identRepo.UpdateStatus(ctx, id, newStatus); err != nil {
		return err
	}

	_ = s.auditSvc.Record(ctx, actorID, "ADMIN", "IDENTITY_STATUS_UPDATED", "IDENTITY", ident.EkaID, "SUCCESS", "", "", "", map[string]interface{}{
		"old_status": ident.Status,
		"new_status": newStatus,
	})

	return nil
}

func (s *IdentityService) List(ctx context.Context, limit, offset int) ([]*domain.Identity, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.identRepo.List(ctx, limit, offset)
}