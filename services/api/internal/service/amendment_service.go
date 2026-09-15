package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/events"
	"github.com/sauryah/eka-id/services/api/internal/repository"
)

var (
	ErrAmendmentNotFound = errors.New("amendment request not found")
	ErrDocumentNotFound  = errors.New("identity document not found")
	ErrInvalidChanges    = errors.New("no valid profile attributes provided for amendment")
)

type AmendmentService struct {
	docRepo   repository.DocumentRepository
	amendRepo repository.AmendmentRepository
	identRepo repository.IdentityRepository
	profRepo  repository.ProfileRepository
	auditSvc  *AuditService
	vcSvc     *VCService
}

func NewAmendmentService(
	docRepo repository.DocumentRepository,
	amendRepo repository.AmendmentRepository,
	identRepo repository.IdentityRepository,
	profRepo repository.ProfileRepository,
	auditSvc *AuditService,
	vcSvc *VCService,
) *AmendmentService {
	return &AmendmentService{
		docRepo:   docRepo,
		amendRepo: amendRepo,
		identRepo: identRepo,
		profRepo:  profRepo,
		auditSvc:  auditSvc,
		vcSvc:     vcSvc,
	}
}

// UploadDocument processes and stores supporting identity proof documents with SHA-256 integrity hash
func (s *AmendmentService) UploadDocument(
	ctx context.Context,
	identityID uuid.UUID,
	docType string,
	docName string,
	mimeType string,
	data []byte,
	actorID *uuid.UUID,
) (*domain.IdentityDocument, error) {
	if len(data) == 0 {
		return nil, errors.New("document data cannot be empty")
	}

	// Compute Cryptographic SHA-256 Checksum
	h := sha256.New()
	h.Write(data)
	sha256Hex := hex.EncodeToString(h.Sum(nil))

	cleanType := strings.ToUpper(strings.TrimSpace(docType))
	if cleanType == "" {
		cleanType = "OTHER"
	}

	doc := &domain.IdentityDocument{
		ID:           uuid.New(),
		IdentityID:   identityID,
		DocumentType: cleanType,
		DocumentName: docName,
		MimeType:     mimeType,
		FileSize:     int64(len(data)),
		SHA256Hash:   sha256Hex,
		FileContent:  base64.StdEncoding.EncodeToString(data),
		Status:       "PENDING",
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.docRepo.Create(ctx, doc); err != nil {
		return nil, err
	}

	_ = s.auditSvc.Record(ctx, actorID, "USER", "DOCUMENT_UPLOADED", "IDENTITY_DOCUMENT", doc.ID.String(), "SUCCESS", "", "", "", map[string]interface{}{
		"document_type": cleanType,
		"sha256_hash":   sha256Hex,
		"file_size":     len(data),
	})

	return doc, nil
}

// ListDocuments retrieves all uploaded documents for an identity
func (s *AmendmentService) ListDocuments(ctx context.Context, identityID uuid.UUID) ([]*domain.IdentityDocument, error) {
	return s.docRepo.ListByIdentityID(ctx, identityID)
}

// CreateAmendmentRequest registers a formal profile modification request with previous snapshot
func (s *AmendmentService) CreateAmendmentRequest(
	ctx context.Context,
	identityID uuid.UUID,
	requestedChanges map[string]interface{},
	justification string,
	docIDs []uuid.UUID,
	actorID *uuid.UUID,
) (*domain.AmendmentRequest, error) {
	if len(requestedChanges) == 0 {
		return nil, ErrInvalidChanges
	}

	ident, err := s.identRepo.GetByID(ctx, identityID)
	if err != nil || ident == nil {
		return nil, ErrIdentityNotFound
	}

	profile, err := s.profRepo.GetByIdentityID(ctx, identityID)
	if err != nil || profile == nil {
		return nil, errors.New("identity profile not found")
	}

	// Capture immutable snapshot of current values for requested attributes
	currentValues := make(map[string]interface{})
	for field := range requestedChanges {
		switch field {
		case "legal_name":
			currentValues["legal_name"] = profile.LegalName
		case "date_of_birth":
			currentValues["date_of_birth"] = profile.DateOfBirth
		case "gender":
			currentValues["gender"] = profile.Gender
		case "address_line1":
			currentValues["address_line1"] = profile.AddressLine1
		case "city":
			currentValues["city"] = profile.City
		case "state":
			currentValues["state"] = profile.State
		case "postal_code":
			currentValues["postal_code"] = profile.PostalCode
		case "country":
			currentValues["country"] = profile.Country
		}
	}

	now := time.Now().UTC()
	req := &domain.AmendmentRequest{
		ID:               uuid.New(),
		IdentityID:       identityID,
		EkaID:            ident.EkaID,
		RequestedChanges: requestedChanges,
		CurrentValues:    currentValues,
		Justification:    justification,
		DocumentIDs:      docIDs,
		Status:           domain.AmendmentStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.amendRepo.Create(ctx, req); err != nil {
		return nil, err
	}

	_ = s.auditSvc.Record(ctx, actorID, "USER", "AMENDMENT_REQUEST_SUBMITTED", "AMENDMENT_REQUEST", req.ID.String(), "SUCCESS", "", "", "", map[string]interface{}{
		"eka_id":            ident.EkaID,
		"requested_changes": requestedChanges,
		"documents_count":   len(docIDs),
	})

	return req, nil
}

// ListAmendmentsByIdentity retrieves all amendment requests for a user
func (s *AmendmentService) ListAmendmentsByIdentity(ctx context.Context, identityID uuid.UUID) ([]*domain.AmendmentRequest, error) {
	return s.amendRepo.ListByIdentityID(ctx, identityID)
}

// ListPendingAmendments returns all pending amendment requests for admin review
func (s *AmendmentService) ListPendingAmendments(ctx context.Context) ([]*domain.AmendmentRequest, error) {
	return s.amendRepo.ListPending(ctx)
}

// ListAllAmendments returns all amendment requests with pagination
func (s *AmendmentService) ListAllAmendments(ctx context.Context, limit, offset int) ([]*domain.AmendmentRequest, int, error) {
	return s.amendRepo.ListAll(ctx, limit, offset)
}

// ReviewAmendment applies or rejects requested profile changes with audit logging and SSE broadcast
func (s *AmendmentService) ReviewAmendment(
	ctx context.Context,
	requestID uuid.UUID,
	approved bool,
	rejectionReason string,
	reviewerID uuid.UUID,
) (*domain.AmendmentRequest, error) {
	req, err := s.amendRepo.GetByID(ctx, requestID)
	if err != nil || req == nil {
		return nil, ErrAmendmentNotFound
	}

	ident, _ := s.identRepo.GetByID(ctx, req.IdentityID)
	ekaID := ""
	if ident != nil {
		ekaID = ident.EkaID
	}

	now := time.Now().UTC()
	if !approved {
		if err := s.amendRepo.UpdateStatus(ctx, requestID, domain.AmendmentStatusRejected, &reviewerID, rejectionReason); err != nil {
			return nil, err
		}

		_ = s.auditSvc.Record(ctx, &reviewerID, "ADMIN", "PROFILE_AMENDMENT_REJECTED", "AMENDMENT_REQUEST", requestID.String(), "REJECTED", "", "", "", map[string]interface{}{
			"identity_id":      req.IdentityID,
			"rejection_reason": rejectionReason,
		})

		// Notify user via real-time SSE stream
		events.GetBroker().Publish(req.IdentityID.String(), events.Event{
			Type:     events.EventAmendmentStatusChanged,
			TargetID: req.IdentityID.String(),
			Payload: map[string]interface{}{
				"request_id":       requestID,
				"status":           domain.AmendmentStatusRejected,
				"rejection_reason": rejectionReason,
				"reviewed_at":      now,
			},
			Timestamp: now,
		})

		req.Status = domain.AmendmentStatusRejected
		req.RejectionReason = rejectionReason
		req.ReviewedBy = &reviewerID
		req.ReviewedAt = &now
		return req, nil
	}

	// Apply requested changes to the profile
	profile, err := s.profRepo.GetByIdentityID(ctx, req.IdentityID)
	if err != nil || profile == nil {
		return nil, errors.New("identity profile not found for applying amendment")
	}

	for field, val := range req.RequestedChanges {
		strVal, _ := val.(string)
		switch field {
		case "legal_name":
			if strVal != "" {
				profile.LegalName = strVal
			}
		case "date_of_birth":
			if strVal != "" {
				profile.DateOfBirth = strVal
			}
		case "gender":
			if strVal != "" {
				profile.Gender = strVal
			}
		case "address_line1":
			if strVal != "" {
				profile.AddressLine1 = strVal
			}
		case "city":
			if strVal != "" {
				profile.City = strVal
			}
		case "state":
			if strVal != "" {
				profile.State = strVal
			}
		case "postal_code":
			if strVal != "" {
				profile.PostalCode = strVal
			}
		case "country":
			if strVal != "" {
				profile.Country = strVal
			}
		}
	}

	if err := s.profRepo.Update(ctx, profile); err != nil {
		return nil, err
	}

	// Update amendment request status
	if err := s.amendRepo.UpdateStatus(ctx, requestID, domain.AmendmentStatusApproved, &reviewerID, ""); err != nil {
		return nil, err
	}

	// Mark attached proof documents as verified
	for _, docID := range req.DocumentIDs {
		_ = s.docRepo.UpdateStatus(ctx, docID, "VERIFIED")
	}

	// Record immutable audit event with before-and-after diff
	_ = s.auditSvc.Record(ctx, &reviewerID, "ADMIN", "PROFILE_AMENDMENT_APPROVED", "AMENDMENT_REQUEST", requestID.String(), "SUCCESS", "", "", "", map[string]interface{}{
		"identity_id":       req.IdentityID,
		"eka_id":            ekaID,
		"applied_changes":   req.RequestedChanges,
		"previous_values":   req.CurrentValues,
		"verified_docs":     req.DocumentIDs,
	})

	// Broadcast SSE event to user's real-time channels
	payload := map[string]interface{}{
		"request_id":      requestID,
		"identity_id":     req.IdentityID,
		"status":          domain.AmendmentStatusApproved,
		"applied_changes": req.RequestedChanges,
		"reviewed_at":     now,
	}

	events.GetBroker().Publish(req.IdentityID.String(), events.Event{
		Type:      events.EventAmendmentStatusChanged,
		TargetID:  req.IdentityID.String(),
		Payload:   payload,
		Timestamp: now,
	})
	if ekaID != "" {
		events.GetBroker().Publish(ekaID, events.Event{
			Type:      events.EventAmendmentStatusChanged,
			TargetID:  ekaID,
			Payload:   payload,
			Timestamp: now,
		})
	}

	req.Status = domain.AmendmentStatusApproved
	req.ReviewedBy = &reviewerID
	req.ReviewedAt = &now
	return req, nil
}
