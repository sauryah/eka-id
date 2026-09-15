package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
	"github.com/sauryah/eka-id/services/api/internal/service"
	"github.com/sauryah/eka-id/services/api/internal/vc"
)

func TestEd25519AsymmetricW3CCredential(t *testing.T) {
	memStore := repository.NewMemoryStore()
	auditSvc := service.NewAuditService(memStore.Audit)
	secret := "test_ed25519_secret_32bytes_key_2026_dev"
	verifyURL := "http://localhost:3000/verify"

	vcSvc := service.NewVCService(memStore.Identities, memStore.Profiles, memStore.Organizations, memStore.Credentials, auditSvc, secret, verifyURL)
	ctx := context.Background()

	// 1. Resolve Platform DID and verify Ed25519 JWK format
	doc, err := vcSvc.ResolveDID(ctx, "did:eka:issuer:platform")
	if err != nil {
		t.Fatalf("failed to resolve platform DID: %v", err)
	}

	if len(doc.VerificationMethod) == 0 {
		t.Fatalf("expected verification methods in DID doc")
	}

	vm := doc.VerificationMethod[0]
	if vm.Type != vc.TypeEd25519VerificationKey {
		t.Errorf("expected type %s, got %s", vc.TypeEd25519VerificationKey, vm.Type)
	}

	if vm.PublicKeyJwk["kty"] != "OKP" || vm.PublicKeyJwk["crv"] != "Ed25519" {
		t.Errorf("expected OKP/Ed25519 JWK, got: %v", vm.PublicKeyJwk)
	}

	// 2. Mint Verifiable Credential with Ed25519 Proof
	identID := uuid.New()
	ident := &domain.Identity{
		ID:                identID,
		EkaID:             "EKA-7K4M-92PX",
		UserID:            uuid.New(),
		Status:            "ACTIVE",
		VerificationLevel: "TIER_1_BASIC",
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
	profile := &domain.Profile{
		IdentityID:  identID,
		LegalName:   "John Mathew",
		DateOfBirth: "1992-05-14",
	}
	cred := &domain.Credential{
		ID:                 uuid.New(),
		IdentityID:         identID,
		Type:               "EMPLOYMENT",
		IssuerName:         "Acme Technologies Ltd.",
		Status:             "ACTIVE",
		IssuedAt:           time.Now().UTC(),
		VerificationMethod: "CORPORATE_DIGITAL_SIGNATURE",
	}

	w3cVC, err := vcSvc.MintVerifiableCredential(ctx, cred, ident, profile)
	if err != nil {
		t.Fatalf("failed to mint W3C VC: %v", err)
	}

	if w3cVC.Proof == nil || w3cVC.Proof.ProofValue == "" {
		t.Fatalf("expected cryptographic proof in VC")
	}

	if w3cVC.Proof.Type != vc.TypeEd25519Signature2020 {
		t.Errorf("expected proof type %s, got %s", vc.TypeEd25519Signature2020, w3cVC.Proof.Type)
	}

	// 3. Verify valid VC
	valid, status, err := vcSvc.VerifyW3CCredential(ctx, w3cVC)
	if err != nil || !valid {
		t.Fatalf("expected VC verification to succeed: %v, status: %s", err, status)
	}

	// 4. Tamper Detection Test
	tamperedVC := *w3cVC
	tamperedVC.CredentialSubject.Claims = map[string]interface{}{
		"legalName": "Tampered Attacker",
	}

	tamperedValid, _, _ := vcSvc.VerifyW3CCredential(ctx, &tamperedVC)
	if tamperedValid {
		t.Fatalf("expected tampered credential to FAIL verification")
	}
}
