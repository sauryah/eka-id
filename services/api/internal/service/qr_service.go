package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
)

var (
	ErrQRTokenNotFound = errors.New("verification unavailable: token is invalid or does not exist")
	ErrQRTokenExpired  = errors.New("verification unavailable: this verification request has expired")
	ErrQRTokenUsed     = errors.New("verification unavailable: this single-use verification token has already been used")
	ErrIdentityInactive = errors.New("verification unavailable: identity is not active")
)

type QRVerificationResult struct {
	Status            string                 `json:"status"`
	EkaID             string                 `json:"eka_id"`
	VerificationLevel string                 `json:"verification_level"`
	VerifiedAt        *time.Time             `json:"verified_at,omitempty"`
	LegalName         string                 `json:"legal_name,omitempty"`
	DisclosedClaims   map[string]interface{} `json:"disclosed_claims"`
	VerificationDate  time.Time              `json:"verification_date"`
}

type QRService struct {
	qrRepo    repository.QRRepository
	identRepo repository.IdentityRepository
	profRepo  repository.ProfileRepository
	auditSvc  *AuditService
	verifyURL string
}

func NewQRService(
	qrRepo repository.QRRepository,
	identRepo repository.IdentityRepository,
	profRepo repository.ProfileRepository,
	auditSvc *AuditService,
	verifyURL string,
) *QRService {
	return &QRService{
		qrRepo:    qrRepo,
		identRepo: identRepo,
		profRepo:  profRepo,
		auditSvc:  auditSvc,
		verifyURL: verifyURL,
	}
}

type CreateQRResponse struct {
	Token         string    `json:"token"`
	VerifyURL     string    `json:"verify_url"`
	ExpiresAt     time.Time `json:"expires_at"`
	AllowedScopes []string  `json:"allowed_scopes"`
}

func (s *QRService) GenerateVerificationToken(
	ctx context.Context,
	identityID uuid.UUID,
	scopes []string,
	duration time.Duration,
	actorID *uuid.UUID,
) (*CreateQRResponse, error) {
	if len(scopes) == 0 {
		scopes = []string{"identity_valid", "legal_name"}
	}
	if duration <= 0 {
		duration = 15 * time.Minute
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}
	token := hex.EncodeToString(randomBytes)

	now := time.Now().UTC()
	expiresAt := now.Add(duration)

	qrToken := &domain.QRToken{
		ID:            uuid.New(),
		Token:         token,
		IdentityID:    identityID,
		AllowedScopes: scopes,
		IsUsed:        false,
		ExpiresAt:     expiresAt,
		CreatedAt:     now,
	}

	if err := s.qrRepo.CreateToken(ctx, qrToken); err != nil {
		return nil, err
	}

	verifyURL := fmt.Sprintf("%s?token=%s", s.verifyURL, token)

	_ = s.auditSvc.Record(ctx, actorID, "USER", "QR_CREATED", "QR_TOKEN", qrToken.ID.String(), "SUCCESS", "", "", "", map[string]interface{}{
		"expires_at": expiresAt,
		"scopes":     scopes,
	})

	return &CreateQRResponse{
		Token:         token,
		VerifyURL:     verifyURL,
		ExpiresAt:     expiresAt,
		AllowedScopes: scopes,
	}, nil
}

func (s *QRService) VerifyToken(ctx context.Context, token, ip, ua, reqID string) (*QRVerificationResult, error) {
	qr, err := s.qrRepo.GetToken(ctx, token)
	if err != nil || qr == nil {
		_ = s.auditSvc.Record(ctx, nil, "ANONYMOUS", "QR_VERIFICATION_FAILED", "QR_TOKEN", token, "FAILURE", ip, ua, reqID, map[string]interface{}{
			"reason": "token not found",
		})
		return nil, ErrQRTokenNotFound
	}

	if time.Now().UTC().After(qr.ExpiresAt) {
		_ = s.auditSvc.Record(ctx, nil, "ANONYMOUS", "QR_VERIFICATION_FAILED", "QR_TOKEN", token, "FAILURE", ip, ua, reqID, map[string]interface{}{
			"reason": "token expired",
		})
		return nil, ErrQRTokenExpired
	}

	identity, err := s.identRepo.GetByID(ctx, qr.IdentityID)
	if err != nil || identity == nil {
		return nil, ErrIdentityNotFound
	}

	if identity.Status != domain.IdentityStatusActive {
		_ = s.auditSvc.Record(ctx, nil, "ANONYMOUS", "QR_VERIFICATION_FAILED", "IDENTITY", identity.EkaID, "FAILURE", ip, ua, reqID, map[string]interface{}{
			"status": identity.Status,
		})
		return nil, ErrIdentityInactive
	}

	profile, _ := s.profRepo.GetByIdentityID(ctx, identity.ID)

	disclosed := make(map[string]interface{})
	var legalName string

	for _, scope := range qr.AllowedScopes {
		switch scope {
		case "identity_valid":
			disclosed["identity_valid"] = true
		case "legal_name":
			if profile != nil {
				legalName = profile.LegalName
				disclosed["legal_name"] = profile.LegalName
			}
		case "dob", "age":
			if profile != nil {
				disclosed["date_of_birth"] = profile.DateOfBirth
			}
		case "photo":
			if profile != nil {
				disclosed["profile_photo_url"] = profile.ProfilePhotoURL
			}
		case "city_state":
			if profile != nil {
				disclosed["city"] = profile.City
				disclosed["state"] = profile.State
			}
		}
	}

	_ = s.auditSvc.Record(ctx, nil, "ANONYMOUS", "QR_VERIFIED", "IDENTITY", identity.EkaID, "SUCCESS", ip, ua, reqID, map[string]interface{}{
		"scopes": qr.AllowedScopes,
	})

	return &QRVerificationResult{
		Status:            "VERIFIED",
		EkaID:             identity.EkaID,
		VerificationLevel: identity.VerificationLevel,
		VerifiedAt:        identity.VerifiedAt,
		LegalName:         legalName,
		DisclosedClaims:   disclosed,
		VerificationDate:  time.Now().UTC(),
	}, nil
}