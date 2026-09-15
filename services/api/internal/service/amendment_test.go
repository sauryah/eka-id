package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
	"github.com/sauryah/eka-id/services/api/internal/service"
)

func TestAmendmentService_DocumentUploadAndReview(t *testing.T) {
	ctx := context.Background()
	store := repository.NewMemoryStore()

	auditSvc := service.NewAuditService(store.Audit)
	identSvc := service.NewIdentityService(store.Identities, store.Profiles, auditSvc)
	vcSvc := service.NewVCService(store.Identities, store.Profiles, store.Organizations, store.Credentials, auditSvc, "test-secret-key-32-bytes-long!!", "http://localhost:3000/verify")
	amendSvc := service.NewAmendmentService(store.Documents, store.Amendments, store.Identities, store.Profiles, auditSvc, vcSvc)

	// Create test user and profile
	userID := uuid.New()
	identID := uuid.New()
	_ = identSvc

	_ = store.Identities.Create(ctx, &domain.Identity{
		ID:                identID,
		EkaID:             "EKA-TEST-1234",
		UserID:            userID,
		Status:            domain.IdentityStatusActive,
		VerificationLevel: domain.VerificationTier1Basic,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	})

	_ = store.Profiles.Create(ctx, &domain.Profile{
		IdentityID:  identID,
		LegalName:   "Original Name",
		DateOfBirth: "1990-01-01",
		Gender:      "MALE",
		City:        "Mumbai",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	})

	// 1. Upload Supporting Document
	dummyPDF := []byte("%PDF-1.4 simulated passport document bytes")
	doc, err := amendSvc.UploadDocument(ctx, identID, "PASSPORT", "passport_scan.pdf", "application/pdf", dummyPDF, &userID)
	if err != nil {
		t.Fatalf("Failed to upload document: %v", err)
	}
	if doc.SHA256Hash == "" {
		t.Fatalf("Expected non-empty SHA-256 hash")
	}
	if doc.DocumentType != "PASSPORT" {
		t.Fatalf("Expected doc type PASSPORT, got %s", doc.DocumentType)
	}

	// 2. Submit Amendment Request
	requestedChanges := map[string]interface{}{
		"legal_name": "Updated Legal Name",
		"city":       "New Delhi",
	}
	amendReq, err := amendSvc.CreateAmendmentRequest(ctx, identID, requestedChanges, "Official name correction as per Passport", []uuid.UUID{doc.ID}, &userID)
	if err != nil {
		t.Fatalf("Failed to create amendment request: %v", err)
	}
	if amendReq.Status != domain.AmendmentStatusPending {
		t.Fatalf("Expected status PENDING_REVIEW, got %s", amendReq.Status)
	}
	if amendReq.CurrentValues["legal_name"] != "Original Name" {
		t.Fatalf("Expected snapshot of current name 'Original Name', got %v", amendReq.CurrentValues["legal_name"])
	}

	// 3. Admin Review & Approval
	adminID := uuid.New()
	reviewedReq, err := amendSvc.ReviewAmendment(ctx, amendReq.ID, true, "", adminID)
	if err != nil {
		t.Fatalf("Failed to review amendment: %v", err)
	}
	if reviewedReq.Status != domain.AmendmentStatusApproved {
		t.Fatalf("Expected status APPROVED, got %s", reviewedReq.Status)
	}

	// 4. Verify Profile was updated
	updatedProfile, err := store.Profiles.GetByIdentityID(ctx, identID)
	if err != nil {
		t.Fatalf("Failed to fetch updated profile: %v", err)
	}
	if updatedProfile.LegalName != "Updated Legal Name" {
		t.Fatalf("Expected profile LegalName to be 'Updated Legal Name', got %s", updatedProfile.LegalName)
	}
	if updatedProfile.City != "New Delhi" {
		t.Fatalf("Expected profile City to be 'New Delhi', got %s", updatedProfile.City)
	}
}
