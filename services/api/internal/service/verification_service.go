package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
)

var (
	ErrVerificationRequestNotFound = errors.New("verification request not found")
	ErrRequestAlreadyProcessed     = errors.New("verification request has already been approved or denied")
	ErrUnauthorizedConsent         = errors.New("unauthorized: cannot consent for another identity")
)

type VerificationService struct {
	verifRepo repository.VerificationRepository
	identRepo repository.IdentityRepository
	profRepo  repository.ProfileRepository
	auditSvc  *AuditService
}

func NewVerificationService(
	verifRepo repository.VerificationRepository,
	identRepo repository.IdentityRepository,
	profRepo repository.ProfileRepository,
	auditSvc *AuditService,
) *VerificationService {
	return &VerificationService{
		verifRepo: verifRepo,
		identRepo: identRepo,
		profRepo:  profRepo,
		auditSvc:  auditSvc,
	}
}

type CreateVerificationRequestInput struct {
	OrgID           uuid.UUID `json:"org_id"`
	EkaID           string    `json:"eka_id"`
	RequestedScopes []string  `json:"requested_scopes"`
	Purpose         string    `json:"purpose"`
	DurationDays    int       `json:"duration_days"`
}

func (s *VerificationService) CreateRequest(ctx context.Context, in CreateVerificationRequestInput, actorID *uuid.UUID) (*domain.VerificationRequest, error) {
	ident, err := s.identRepo.GetByEkaID(ctx, in.EkaID)
	if err != nil {
		return nil, ErrIdentityNotFound
	}

	if in.DurationDays <= 0 {
		in.DurationDays = 7
	}

	now := time.Now().UTC()
	req := &domain.VerificationRequest{
		ID:              uuid.New(),
		OrgID:           in.OrgID,
		IdentityID:      ident.ID,
		RequestedScopes: in.RequestedScopes,
		Purpose:         in.Purpose,
		Status:          domain.RequestStatusPending,
		ExpiresAt:       now.Add(time.Duration(in.DurationDays) * 24 * time.Hour),
		CreatedAt:       now,
	}

	if err := s.verifRepo.CreateRequest(ctx, req); err != nil {
		return nil, err
	}

	_ = s.auditSvc.Record(ctx, actorID, "ORG", "VERIFICATION_REQUESTED", "VERIFICATION_REQUEST", req.ID.String(), "SUCCESS", "", "", "", map[string]interface{}{
		"eka_id": ident.EkaID,
		"scopes": req.RequestedScopes,
	})

	return req, nil
}

func (s *VerificationService) RespondRequest(ctx context.Context, requestID, identityID uuid.UUID, approved bool, actorID *uuid.UUID) (*domain.VerificationResult, error) {
	req, err := s.verifRepo.GetRequestByID(ctx, requestID)
	if err != nil {
		return nil, ErrVerificationRequestNotFound
	}

	if req.IdentityID != identityID {
		return nil, ErrUnauthorizedConsent
	}

	if req.Status != domain.RequestStatusPending {
		return nil, ErrRequestAlreadyProcessed
	}

	now := time.Now().UTC()
	if !approved {
		if err := s.verifRepo.UpdateRequestStatus(ctx, requestID, domain.RequestStatusDenied, &now); err != nil {
			return nil, err
		}
		_ = s.auditSvc.Record(ctx, actorID, "USER", "VERIFICATION_DENIED", "VERIFICATION_REQUEST", requestID.String(), "DENIED", "", "", "", nil)
		return nil, nil
	}

	profile, _ := s.profRepo.GetByIdentityID(ctx, identityID)
	ident, _ := s.identRepo.GetByID(ctx, identityID)

	disclosed := make(map[string]interface{})
	for _, scope := range req.RequestedScopes {
		switch scope {
		case "identity_valid":
			disclosed["identity_valid"] = (ident != nil && ident.Status == domain.IdentityStatusActive)
		case "name_match", "legal_name":
			if profile != nil {
				disclosed["legal_name"] = profile.LegalName
			}
		case "dob", "date_of_birth":
			if profile != nil {
				disclosed["date_of_birth"] = profile.DateOfBirth
			}
		case "phone":
			if profile != nil {
				disclosed["phone"] = profile.Phone
			}
		case "email":
			if profile != nil {
				disclosed["email"] = profile.Email
			}
		case "address":
			if profile != nil {
				disclosed["address_line1"] = profile.AddressLine1
				disclosed["city"] = profile.City
				disclosed["state"] = profile.State
				disclosed["postal_code"] = profile.PostalCode
				disclosed["country"] = profile.Country
			}
		}
	}

	res := &domain.VerificationResult{
		ID:                uuid.New(),
		RequestID:         requestID,
		DisclosedClaims:   disclosed,
		Status:            "VALID",
		VerifiedByActorID: actorID,
		CreatedAt:         now,
	}

	if err := s.verifRepo.CreateResult(ctx, res); err != nil {
		return nil, err
	}

	if err := s.verifRepo.UpdateRequestStatus(ctx, requestID, domain.RequestStatusApproved, &now); err != nil {
		return nil, err
	}

	_ = s.auditSvc.Record(ctx, actorID, "USER", "VERIFICATION_APPROVED", "VERIFICATION_REQUEST", requestID.String(), "SUCCESS", "", "", "", map[string]interface{}{
		"disclosed_scopes": req.RequestedScopes,
	})

	return res, nil
}

func (s *VerificationService) ListPendingByIdentity(ctx context.Context, identityID uuid.UUID) ([]*domain.VerificationRequest, error) {
	return s.verifRepo.ListRequestsByIdentity(ctx, identityID)
}