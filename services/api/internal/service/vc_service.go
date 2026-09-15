package service

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/crypto"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/repository"
	"github.com/sauryah/eka-id/services/api/internal/vc"
)

var (
	ErrInvalidDIDFormat       = errors.New("invalid DID format: must follow did:eka:<eka_id> or did:eka:org:<slug>")
	ErrDIDNotFound            = errors.New("DID entity not found")
	ErrInvalidW3CProof        = errors.New("invalid or tampered W3C cryptographic proof")
	ErrW3CCredentialExpired   = errors.New("W3C verifiable credential has expired")
)

type VCService struct {
	identRepo       repository.IdentityRepository
	profRepo        repository.ProfileRepository
	orgRepo         repository.OrganizationRepository
	credRepo        repository.CredentialRepository
	auditSvc        *AuditService
	jwtSecret       []byte
	verifyURL       string
	platformPubKey  ed25519.PublicKey
	platformPrivKey ed25519.PrivateKey
}

func NewVCService(
	identRepo repository.IdentityRepository,
	profRepo repository.ProfileRepository,
	orgRepo repository.OrganizationRepository,
	credRepo repository.CredentialRepository,
	auditSvc *AuditService,
	jwtSecret string,
	verifyURL string,
) *VCService {
	pub, priv, _ := crypto.DeriveEd25519KeyPair([]byte(jwtSecret))

	return &VCService{
		identRepo:       identRepo,
		profRepo:        profRepo,
		orgRepo:         orgRepo,
		credRepo:        credRepo,
		auditSvc:        auditSvc,
		jwtSecret:       []byte(jwtSecret),
		verifyURL:       verifyURL,
		platformPubKey:  pub,
		platformPrivKey: priv,
	}
}

// ResolveDID resolves a did:eka identifier to a standard W3C DID Document
func (s *VCService) ResolveDID(ctx context.Context, didURI string) (*vc.DIDDocument, error) {
	cleanDID := strings.TrimSpace(didURI)

	// Case 1: Platform Root Authority DID
	if cleanDID == "did:eka:issuer:platform" || cleanDID == "did:eka:platform" {
		keyID := "did:eka:issuer:platform#key-1"
		return &vc.DIDDocument{
			Context: []string{vc.W3CDIDContextV1, vc.W3CEd25519Context2020, vc.W3CSecurityJWS2020},
			ID:      "did:eka:issuer:platform",
			VerificationMethod: []vc.VerificationMethod{
				{
					ID:           keyID,
					Type:         vc.TypeEd25519VerificationKey,
					Controller:   "did:eka:issuer:platform",
					PublicKeyJwk: crypto.PublicKeyToJWK(s.platformPubKey, keyID),
				},
			},
			Authentication:  []string{keyID},
			AssertionMethod: []string{keyID},
			Service: []vc.DIDService{
				{
					ID:              "did:eka:issuer:platform#verification-service",
					Type:            "EkaVerificationService",
					ServiceEndpoint: s.verifyURL,
				},
			},
		}, nil
	}

	// Case 2: Organization DID (did:eka:org:<slug>)
	if strings.HasPrefix(cleanDID, "did:eka:org:") {
		slug := strings.TrimPrefix(cleanDID, "did:eka:org:")
		org, err := s.orgRepo.GetBySlug(ctx, slug)
		if err != nil || org == nil {
			return nil, ErrDIDNotFound
		}

		keyID := fmt.Sprintf("%s#key-1", cleanDID)
		orgSeed := fmt.Sprintf("%s:org:%s", string(s.jwtSecret), org.Slug)
		orgPub, _, _ := crypto.DeriveEd25519KeyPair([]byte(orgSeed))

		return &vc.DIDDocument{
			Context:     []string{vc.W3CDIDContextV1, vc.W3CEd25519Context2020, vc.W3CSecurityJWS2020},
			ID:          cleanDID,
			AlsoKnownAs: []string{fmt.Sprintf("https://id.eka.dev/orgs/%s", org.Slug)},
			VerificationMethod: []vc.VerificationMethod{
				{
					ID:           keyID,
					Type:         vc.TypeEd25519VerificationKey,
					Controller:   cleanDID,
					PublicKeyJwk: crypto.PublicKeyToJWK(orgPub, keyID),
				},
			},
			Authentication:  []string{keyID},
			AssertionMethod: []string{keyID},
			Service: []vc.DIDService{
				{
					ID:              fmt.Sprintf("%s#org-service", cleanDID),
					Type:            "EkaOrganizationVerificationService",
					ServiceEndpoint: s.verifyURL,
				},
			},
		}, nil
	}

	// Case 3: Identity Subject DID (did:eka:EKA-XXXX-XXXX)
	if strings.HasPrefix(cleanDID, "did:eka:") {
		ekaID := strings.TrimPrefix(cleanDID, "did:eka:")
		if !crypto.ValidateEkaID(ekaID) {
			return nil, ErrInvalidDIDFormat
		}

		ident, err := s.identRepo.GetByEkaID(ctx, ekaID)
		if err != nil || ident == nil {
			return nil, ErrDIDNotFound
		}

		keyID := fmt.Sprintf("did:eka:%s#key-1", ident.EkaID)
		subjSeed := fmt.Sprintf("%s:subject:%s", string(s.jwtSecret), ident.EkaID)
		subjPub, _, _ := crypto.DeriveEd25519KeyPair([]byte(subjSeed))

		return &vc.DIDDocument{
			Context: []string{vc.W3CDIDContextV1, vc.W3CEd25519Context2020, vc.W3CSecurityJWS2020},
			ID:      fmt.Sprintf("did:eka:%s", ident.EkaID),
			VerificationMethod: []vc.VerificationMethod{
				{
					ID:           keyID,
					Type:         vc.TypeEd25519VerificationKey,
					Controller:   fmt.Sprintf("did:eka:%s", ident.EkaID),
					PublicKeyJwk: crypto.PublicKeyToJWK(subjPub, keyID),
				},
			},
			Authentication:  []string{keyID},
			AssertionMethod: []string{keyID},
			Service: []vc.DIDService{
				{
					ID:              fmt.Sprintf("did:eka:%s#verification", ident.EkaID),
					Type:            "EkaIdentityAttestationService",
					ServiceEndpoint: s.verifyURL,
				},
			},
		}, nil
	}

	return nil, ErrInvalidDIDFormat
}

// MintVerifiableCredential creates a signed W3C Verifiable Credential from a domain Credential
func (s *VCService) MintVerifiableCredential(ctx context.Context, cred *domain.Credential, identity *domain.Identity, profile *domain.Profile) (*vc.VerifiableCredential, error) {
	if cred == nil || identity == nil {
		return nil, errors.New("credential and identity are required")
	}

	subjectDID := fmt.Sprintf("did:eka:%s", identity.EkaID)
	issuerDID := "did:eka:issuer:platform"
	if cred.IssuerName != "" {
		slug := strings.ToLower(strings.ReplaceAll(cred.IssuerName, " ", "-"))
		issuerDID = fmt.Sprintf("did:eka:org:%s", slug)
	}

	claims := map[string]interface{}{
		"credentialType":     cred.Type,
		"status":             cred.Status,
		"verificationMethod": cred.VerificationMethod,
	}

	if profile != nil {
		claims["legalName"] = profile.LegalName
	}
	if cred.Metadata != nil {
		for k, v := range cred.Metadata {
			claims[k] = v
		}
	}

	now := time.Now().UTC()
	verifiableCred := &vc.VerifiableCredential{
		Context: []string{
			vc.W3CCredentialsContextV1,
			vc.W3CEd25519Context2020,
			vc.EkaIDContextV1,
		},
		ID:   fmt.Sprintf("urn:uuid:%s", cred.ID.String()),
		Type: []string{vc.TypeVerifiableCredential, vc.TypeEkaIdentityCredential, cred.Type},
		Issuer: vc.Issuer{
			ID:   issuerDID,
			Name: cred.IssuerName,
		},
		IssuanceDate:   cred.IssuedAt,
		ExpirationDate: cred.ExpiresAt,
		CredentialSubject: vc.CredentialSubject{
			ID:     subjectDID,
			Claims: claims,
		},
	}

	// Generate Cryptographic Proof
	proof, err := s.generateProof(verifiableCred, issuerDID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}
	verifiableCred.Proof = proof

	return verifiableCred, nil
}

// MintVerifiablePresentation packages a verification result into a W3C Verifiable Presentation
func (s *VCService) MintVerifiablePresentation(ctx context.Context, ekaID string, level string, disclosedClaims map[string]interface{}) (*vc.VerifiablePresentation, error) {
	subjectDID := fmt.Sprintf("did:eka:%s", ekaID)
	now := time.Now().UTC()
	exp := now.Add(15 * time.Minute)

	credID := uuid.New().String()
	attestationVC := vc.VerifiableCredential{
		Context: []string{
			vc.W3CCredentialsContextV1,
			vc.W3CEd25519Context2020,
			vc.EkaIDContextV1,
		},
		ID:   fmt.Sprintf("urn:uuid:%s", credID),
		Type: []string{vc.TypeVerifiableCredential, vc.TypeEkaVerifiableAttestation},
		Issuer: vc.Issuer{
			ID:   "did:eka:issuer:platform",
			Name: "EKA Universal Identity Platform",
		},
		IssuanceDate:   now,
		ExpirationDate: &exp,
		CredentialSubject: vc.CredentialSubject{
			ID: subjectDID,
			Claims: map[string]interface{}{
				"verificationLevel": level,
				"disclosedClaims":   disclosedClaims,
			},
		},
	}

	proof, err := s.generateProof(&attestationVC, "did:eka:issuer:platform", now)
	if err != nil {
		return nil, err
	}
	attestationVC.Proof = proof

	pres := &vc.VerifiablePresentation{
		Context: []string{
			vc.W3CCredentialsContextV1,
			vc.W3CEd25519Context2020,
			vc.EkaIDContextV1,
		},
		ID:                   fmt.Sprintf("urn:uuid:%s", uuid.New().String()),
		Type:                 []string{vc.TypeVerifiablePresentation},
		Holder:               subjectDID,
		VerifiableCredential: []vc.VerifiableCredential{attestationVC},
	}

	// Sign presentation envelope
	presProof, err := s.generateProof(pres, subjectDID, now)
	if err == nil {
		pres.Proof = presProof
	}

	return pres, nil
}

// VerifyW3CCredential validates a W3C Verifiable Credential's cryptographic integrity and expiration
func (s *VCService) VerifyW3CCredential(ctx context.Context, cred *vc.VerifiableCredential) (bool, string, error) {
	if cred == nil || cred.Proof == nil {
		return false, "Missing proof", ErrInvalidW3CProof
	}

	// 1. Expiration check
	if cred.ExpirationDate != nil && time.Now().UTC().After(*cred.ExpirationDate) {
		return false, "Credential expired", ErrW3CCredentialExpired
	}

	// 2. Compute canonical payload
	payloadBytes := s.computeCanonicalPayload(cred, cred.Proof.Created)

	// 3. Try Ed25519 Asymmetric Verification first
	if crypto.VerifyEd25519(s.platformPubKey, payloadBytes, cred.Proof.ProofValue) {
		return true, "VALID", nil
	}

	// 4. Fallback to HMAC verification for legacy credentials
	mac := hmac.New(sha256.New, s.jwtSecret)
	mac.Write(payloadBytes)
	legacyExpected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if hmac.Equal([]byte(cred.Proof.ProofValue), []byte(legacyExpected)) {
		return true, "VALID", nil
	}

	return false, "Cryptographic proof mismatch", ErrInvalidW3CProof
}

func (s *VCService) generateProof(target interface{}, issuerDID string, timestamp time.Time) (*vc.Proof, error) {
	payloadBytes := s.computeCanonicalPayload(target, timestamp)
	proofVal := crypto.SignEd25519(s.platformPrivKey, payloadBytes)

	return &vc.Proof{
		Type:               vc.TypeEd25519Signature2020,
		Created:            timestamp,
		VerificationMethod: fmt.Sprintf("%s#key-1", issuerDID),
		ProofPurpose:       "assertionMethod",
		ProofValue:         proofVal,
	}, nil
}

func (s *VCService) computeCanonicalPayload(target interface{}, timestamp time.Time) []byte {
	var rawData []byte
	switch v := target.(type) {
	case *vc.VerifiableCredential:
		copyVC := *v
		copyVC.Proof = nil
		rawData, _ = json.Marshal(copyVC)
	case *vc.VerifiablePresentation:
		copyVP := *v
		copyVP.Proof = nil
		rawData, _ = json.Marshal(copyVP)
	case vc.VerifiableCredential:
		copyVC := v
		copyVC.Proof = nil
		rawData, _ = json.Marshal(copyVC)
	default:
		rawData, _ = json.Marshal(target)
	}

	var combined []byte
	combined = append(combined, rawData...)
	combined = append(combined, []byte(timestamp.Format(time.RFC3339Nano))...)
	return combined
}
