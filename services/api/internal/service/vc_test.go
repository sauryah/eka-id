package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
	"github.com/sauryah/eka-id/services/api/internal/service"
)

func setupVCTest() (*service.VCService, *domain.Identity, *domain.Profile, *domain.Credential, *repository.MemoryStore) {
	mem := repository.NewMemoryStore()
	auditSvc := service.NewAuditService(mem.Audit)
	secret := "test-jwt-secret-32-chars-long!!"
	vcSvc := service.NewVCService(mem.Identities, mem.Profiles, mem.Organizations, mem.Credentials, auditSvc, secret, "https://id.eka.dev/verify")

	ctx := context.Background()

	// Create test user and identity
	identID := uuid.New()
	userID := uuid.New()
	ekaID := "EKA-7K4M-92PX"
	ident := &domain.Identity{
		ID:                identID,
		EkaID:             ekaID,
		UserID:            userID,
		Status:            domain.IdentityStatusActive,
		VerificationLevel: domain.VerificationTier1Basic,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	_ = mem.Identities.Create(ctx, ident)

	prof := &domain.Profile{
		IdentityID:  identID,
		LegalName:   "John Mathew",
		DateOfBirth: "1992-05-14",
		Email:       "john.mathew@example.com",
		Phone:       "+919876500001",
		Country:     "India",
	}
	_ = mem.Profiles.Create(ctx, prof)

	credID := uuid.New()
	cred := &domain.Credential{
		ID:                 credID,
		IdentityID:         identID,
		Type:               "EMPLOYMENT",
		IssuerName:         "Acme Technologies Ltd.",
		Status:             "ACTIVE",
		IssuedAt:           time.Now().UTC().Add(-24 * time.Hour),
		VerificationMethod: "CORPORATE_DIGITAL_SIGNATURE",
		Metadata: map[string]interface{}{
			"title":      "Senior Systems Architect",
			"department": "Platform Engineering",
		},
		CreatedAt: time.Now().UTC(),
	}
	_ = mem.Credentials.Create(ctx, cred)

	org := &domain.Organization{
		ID:         uuid.New(),
		Name:       "Acme Technologies Ltd.",
		Slug:       "acme-tech",
		ApiKeyHash: "dummy-hash",
		Status:     "ACTIVE",
	}
	_ = mem.Organizations.Create(ctx, org)

	return vcSvc, ident, prof, cred, mem
}

func TestResolveDID_UserIdentity(t *testing.T) {
	vcSvc, ident, _, _, _ := setupVCTest()
	ctx := context.Background()

	didURI := "did:eka:" + ident.EkaID
	doc, err := vcSvc.ResolveDID(ctx, didURI)
	if err != nil {
		t.Fatalf("Failed to resolve identity DID: %v", err)
	}

	if doc.ID != didURI {
		t.Fatalf("Expected DID %s, got: %s", didURI, doc.ID)
	}

	if len(doc.VerificationMethod) == 0 {
		t.Fatal("Expected at least one verification method in DID document")
	}

	if !strings.HasPrefix(doc.VerificationMethod[0].ID, didURI) {
		t.Fatalf("Expected verificationMethod ID prefix %s, got: %s", didURI, doc.VerificationMethod[0].ID)
	}

	if len(doc.Authentication) == 0 || doc.Authentication[0] != doc.VerificationMethod[0].ID {
		t.Fatalf("Expected authentication key reference %s", doc.VerificationMethod[0].ID)
	}
}

func TestResolveDID_Organization(t *testing.T) {
	vcSvc, _, _, _, _ := setupVCTest()
	ctx := context.Background()

	orgDID := "did:eka:org:acme-tech"
	doc, err := vcSvc.ResolveDID(ctx, orgDID)
	if err != nil {
		t.Fatalf("Failed to resolve organization DID: %v", err)
	}

	if doc.ID != orgDID {
		t.Fatalf("Expected org DID %s, got: %s", orgDID, doc.ID)
	}
}

func TestResolveDID_PlatformIssuer(t *testing.T) {
	vcSvc, _, _, _, _ := setupVCTest()
	ctx := context.Background()

	platformDID := "did:eka:issuer:platform"
	doc, err := vcSvc.ResolveDID(ctx, platformDID)
	if err != nil {
		t.Fatalf("Failed to resolve platform issuer DID: %v", err)
	}

	if doc.ID != platformDID {
		t.Fatalf("Expected platform DID %s, got: %s", platformDID, doc.ID)
	}
}

func TestMintAndVerifyVerifiableCredential(t *testing.T) {
	vcSvc, ident, prof, cred, _ := setupVCTest()
	ctx := context.Background()

	// 1. Mint W3C VC
	w3cVC, err := vcSvc.MintVerifiableCredential(ctx, cred, ident, prof)
	if err != nil {
		t.Fatalf("Failed to mint W3C VC: %v", err)
	}

	if w3cVC.CredentialSubject.ID != "did:eka:"+ident.EkaID {
		t.Fatalf("Expected subject DID did:eka:%s, got: %s", ident.EkaID, w3cVC.CredentialSubject.ID)
	}

	if w3cVC.Proof == nil || w3cVC.Proof.ProofValue == "" {
		t.Fatal("Expected cryptographic proof object with non-empty signature")
	}

	// 2. Verify Valid Credential
	valid, reason, err := vcSvc.VerifyW3CCredential(ctx, w3cVC)
	if err != nil || !valid {
		t.Fatalf("Expected valid credential verification, got valid=%v, reason=%s, err=%v", valid, reason, err)
	}

	// 3. Verify Tampered Credential (tamper claim)
	tamperedVC := *w3cVC
	tamperedVC.CredentialSubject.Claims = map[string]interface{}{
		"title": "CEO & Founder", // Altered!
	}
	validTampered, _, _ := vcSvc.VerifyW3CCredential(ctx, &tamperedVC)
	if validTampered {
		t.Fatal("Expected tampered credential verification to fail, but it passed!")
	}
}

func TestMintVerifiablePresentation(t *testing.T) {
	vcSvc, ident, _, _, _ := setupVCTest()
	ctx := context.Background()

	disclosed := map[string]interface{}{
		"identity_valid": true,
		"legal_name":     "John Mathew",
	}

	vp, err := vcSvc.MintVerifiablePresentation(ctx, ident.EkaID, ident.VerificationLevel, disclosed)
	if err != nil {
		t.Fatalf("Failed to mint VP: %v", err)
	}

	if vp.Holder != "did:eka:"+ident.EkaID {
		t.Fatalf("Expected holder did:eka:%s, got: %s", ident.EkaID, vp.Holder)
	}

	if len(vp.VerifiableCredential) == 0 {
		t.Fatal("Expected VP to contain at least 1 verifiable credential")
	}

	attestation := vp.VerifiableCredential[0]
	if attestation.Proof == nil || attestation.Proof.ProofValue == "" {
		t.Fatal("Expected attestation credential in VP to have valid proof")
	}

	valid, _, err := vcSvc.VerifyW3CCredential(ctx, &attestation)
	if err != nil || !valid {
		t.Fatalf("Failed to verify attestation inside VP: %v", err)
	}
}
