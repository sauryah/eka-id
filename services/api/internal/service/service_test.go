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

func setupTestServices() (*service.AuthService, *service.IdentityService, *service.QRService, *service.VerificationService, *service.AuditService, *repository.MemoryStore) {
	mem := repository.NewMemoryStore()
	auditSvc := service.NewAuditService(mem.Audit)
	idSvc := service.NewIdentityService(mem.Identities, mem.Profiles, auditSvc)
	dedupSvc := service.NewDeduplicationService(mem.Profiles, mem.Duplicates, mem.Identities, auditSvc)
	authSvc := service.NewAuthService(mem.Users, idSvc, mem.Profiles, dedupSvc, auditSvc, "test-jwt-secret-32-chars-long!!")
	qrSvc := service.NewQRService(mem.QR, mem.Identities, mem.Profiles, auditSvc, "https://id.eka.dev/verify")
	verifSvc := service.NewVerificationService(mem.Verification, mem.Identities, mem.Profiles, auditSvc)

	return authSvc, idSvc, qrSvc, verifSvc, auditSvc, mem
}

func TestRegistrationAndEkaIDGeneration(t *testing.T) {
	authSvc, _, _, _, _, _ := setupTestServices()
	ctx := context.Background()

	input := service.RegistrationInput{
		Email:       "alice@example.com",
		Phone:       "+919876543210",
		Password:    "SecurePass123!",
		LegalName:   "Alice Sharma",
		DateOfBirth: "1995-08-20",
		Gender:      "FEMALE",
		OTPCode:     "123456",
	}

	result, err := authSvc.Register(ctx, input, "127.0.0.1", "test-agent", "req-1")
	if err != nil {
		t.Fatalf("Registration failed: %v", err)
	}

	if result.Identity.EkaID == "" {
		t.Fatal("Expected non-empty EKA ID")
	}

	if len(result.Identity.EkaID) != 13 {
		t.Fatalf("Expected EKA ID length 13, got: %s", result.Identity.EkaID)
	}

	if result.Profile.LegalName != "Alice Sharma" {
		t.Fatalf("Expected name Alice Sharma, got: %s", result.Profile.LegalName)
	}
}

func TestQRVerification_ZeroPIIAndScopes(t *testing.T) {
	authSvc, _, qrSvc, _, _, _ := setupTestServices()
	ctx := context.Background()

	reg, err := authSvc.Register(ctx, service.RegistrationInput{
		Email:       "bob@example.com",
		Phone:       "+919876543211",
		Password:    "SecurePass123!",
		LegalName:   "Robert Chen",
		DateOfBirth: "1988-12-05",
		OTPCode:     "123456",
	}, "127.0.0.1", "test-agent", "req-2")
	if err != nil {
		t.Fatalf("Reg error: %v", err)
	}

	qrResp, err := qrSvc.GenerateVerificationToken(ctx, reg.Identity.ID, []string{"identity_valid", "legal_name"}, 10*time.Minute, &reg.User.ID)
	if err != nil {
		t.Fatalf("Generate QR error: %v", err)
	}

	if qrResp.Token == "" {
		t.Fatal("Expected valid token")
	}

	verifResult, err := qrSvc.VerifyToken(ctx, qrResp.Token, "192.168.1.50", "mobile-scanner", "req-3")
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	if verifResult.Status != "VERIFIED" {
		t.Fatalf("Expected status VERIFIED, got: %s", verifResult.Status)
	}

	if verifResult.LegalName != "Robert Chen" {
		t.Fatalf("Expected name Robert Chen, got: %s", verifResult.LegalName)
	}

	if _, exists := verifResult.DisclosedClaims["date_of_birth"]; exists {
		t.Fatal("Date of birth was disclosed despite not being in QR scope!")
	}
	if _, exists := verifResult.DisclosedClaims["phone"]; exists {
		t.Fatal("Phone was disclosed despite not being in QR scope!")
	}
}

func TestVerificationRequest_ConsentFlow(t *testing.T) {
	authSvc, _, _, verifSvc, _, _ := setupTestServices()
	ctx := context.Background()

	reg, _ := authSvc.Register(ctx, service.RegistrationInput{
		Email:       "charlie@example.com",
		Password:    "SecurePass123!",
		LegalName:   "Charlie Davis",
		DateOfBirth: "1990-01-01",
		Phone:       "+919876543212",
		OTPCode:     "123456",
	}, "127.0.0.1", "agent", "req-4")

	orgID := uuid.New()
	req, err := verifSvc.CreateRequest(ctx, service.CreateVerificationRequestInput{
		OrgID:           orgID,
		EkaID:           reg.Identity.EkaID,
		RequestedScopes: []string{"identity_valid", "name_match", "phone"},
		Purpose:         "Job application verification",
		DurationDays:    7,
	}, &orgID)
	if err != nil {
		t.Fatalf("CreateRequest error: %v", err)
	}

	if req.Status != domain.RequestStatusPending {
		t.Fatalf("Expected PENDING, got: %s", req.Status)
	}

	res, err := verifSvc.RespondRequest(ctx, req.ID, reg.Identity.ID, true, &reg.User.ID)
	if err != nil {
		t.Fatalf("Approve request failed: %v", err)
	}

	if res.Status != "VALID" {
		t.Fatalf("Expected result status VALID, got: %s", res.Status)
	}

	if res.DisclosedClaims["phone"] != "+919876543212" {
		t.Fatalf("Expected phone disclosed, got: %v", res.DisclosedClaims["phone"])
	}
}

func TestAuditLog_Redaction(t *testing.T) {
	_, _, _, _, auditSvc, mem := setupTestServices()
	ctx := context.Background()

	userID := uuid.New()
	err := auditSvc.Record(ctx, &userID, "USER", "TEST_ACTION", "RESOURCE", "123", "SUCCESS", "127.0.0.1", "agent", "req-5", map[string]interface{}{
		"safe_field":   "visible_data",
		"password":     "SuperSecretPlaintext!",
		"jwt_token":    "ey...",
		"otp_code":     "654321",
	})
	if err != nil {
		t.Fatalf("Audit record failed: %v", err)
	}

	events, _, err := mem.Audit.List(ctx, 10, 0, &userID, "")
	if err != nil || len(events) == 0 {
		t.Fatalf("Failed to retrieve audit events: %v", err)
	}

	meta := events[0].Metadata
	if meta["password"] != "[REDACTED]" {
		t.Fatalf("Password was not redacted in audit log! Got: %v", meta["password"])
	}
	if meta["jwt_token"] != "[REDACTED]" {
		t.Fatalf("JWT Token was not redacted in audit log! Got: %v", meta["jwt_token"])
	}
	if meta["otp_code"] != "[REDACTED]" {
		t.Fatalf("OTP code was not redacted in audit log! Got: %v", meta["otp_code"])
	}
}

func TestBiometricFacialDeduplication(t *testing.T) {
	authSvc, _, _, _, _, mem := setupTestServices()
	ctx := context.Background()

	// 1. Register Person 1 with Face Embedding Vector A
	faceVecA := []float64{0.15, -0.08, 0.42, -0.19, 0.22, 0.35, -0.11, 0.05, 0.28, -0.14, 0.31, -0.03}
	_, err := authSvc.Register(ctx, service.RegistrationInput{
		Email:       "original.user@example.com",
		Password:    "Password123!",
		LegalName:   "Original User",
		DateOfBirth: "1990-01-01",
		Phone:       "+919876500010",
		OTPCode:     "123456",
		Metadata: map[string]interface{}{
			"face_embedding": faceVecA,
		},
	}, "127.0.0.1", "test-agent", "req-bio-1")
	if err != nil {
		t.Fatalf("Registration of user 1 failed: %v", err)
	}

	// 2. Register Person 2 with completely different name, email, phone, and DOB
	// But WITH an identical or 99% similar face embedding vector!
	faceVecB := []float64{0.151, -0.079, 0.419, -0.191, 0.221, 0.349, -0.109, 0.051, 0.281, -0.139, 0.309, -0.031}
	_, err = authSvc.Register(ctx, service.RegistrationInput{
		Email:       "fraudulent.clone@example.com",
		Password:    "Password123!",
		LegalName:   "Completely Different Name",
		DateOfBirth: "2000-12-31",
		Phone:       "+919876599999",
		OTPCode:     "123456",
		Metadata: map[string]interface{}{
			"face_embedding": faceVecB,
		},
	}, "127.0.0.1", "test-agent", "req-bio-2")
	if err != nil {
		t.Fatalf("Registration of user 2 failed: %v", err)
	}

	// 3. Verify that Deduplication engine flagged the duplicate due to biometric face match!
	flags, err := mem.Duplicates.ListPending(ctx)
	if err != nil {
		t.Fatalf("Failed to retrieve pending flags: %v", err)
	}

	if len(flags) == 0 {
		t.Fatal("Expected biometric duplicate flag to be generated, but none was found!")
	}

	flag := flags[0]
	if flag.ConfidenceScore < 85.0 {
		t.Fatalf("Expected confidence score >= 85.0, got: %f", flag.ConfidenceScore)
	}

	foundBioReason := false
	for _, reason := range flag.MatchReasons {
		if strings.Contains(reason, "Biometric Face Match") {
			foundBioReason = true
			break
		}
	}
	if !foundBioReason {
		t.Fatalf("Expected Biometric Face Match reason in %v", flag.MatchReasons)
	}
}