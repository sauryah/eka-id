package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/sauryah/eka-id/services/api/internal/config"
	"github.com/sauryah/eka-id/services/api/internal/domain"
	"github.com/sauryah/eka-id/services/api/internal/handler"
	"github.com/sauryah/eka-id/services/api/internal/middleware"
	"github.com/sauryah/eka-id/services/api/internal/repository"
	"github.com/sauryah/eka-id/services/api/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()
	log.Printf("[INFO] Starting EKA ID Core API server in %s mode...", cfg.Environment)

	// Repositories initialization
	var userRepo repository.UserRepository
	var identRepo repository.IdentityRepository
	var profRepo repository.ProfileRepository
	var orgRepo repository.OrganizationRepository
	var verifRepo repository.VerificationRepository
	var qrRepo repository.QRRepository
	var credRepo repository.CredentialRepository
	var auditRepo repository.AuditRepository
	var dedupRepo repository.DuplicateRepository

	// Attempt PostgreSQL connection
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)
	db, err := sql.Open("postgres", dsn)
	var dbConnected bool
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if pingErr := db.PingContext(ctx); pingErr == nil {
			dbConnected = true
			log.Printf("[INFO] Connected to PostgreSQL on %s:%s", cfg.DBHost, cfg.DBPort)
		}
		cancel()
	}

	if dbConnected {
		pgStore := repository.NewPostgresStore(db)
		// We use PostgresStore directly
		userRepo = pgStore.Users
		identRepo = pgStore.Identities
		profRepo = pgStore.Profiles
		orgRepo = pgStore.Organizations
		verifRepo = pgStore.Verification
		qrRepo = pgStore.QR
		credRepo = pgStore.Credentials
		auditRepo = pgStore.Audit
		dedupRepo = pgStore.Duplicates
	} else {
		log.Printf("[WARN] PostgreSQL not available (%v). Initializing resilient in-memory store with demo seeds...", err)
		memStore := repository.NewMemoryStore()
		seedMemoryStore(memStore)
		userRepo = memStore.Users
		identRepo = memStore.Identities
		profRepo = memStore.Profiles
		orgRepo = memStore.Organizations
		verifRepo = memStore.Verification
		qrRepo = memStore.QR
		credRepo = memStore.Credentials
		auditRepo = memStore.Audit
		dedupRepo = memStore.Duplicates
	}

	_ = orgRepo // preserve for future multi-tenant org routing

	// Service Layer
	auditSvc := service.NewAuditService(auditRepo)
	identSvc := service.NewIdentityService(identRepo, profRepo, auditSvc)
	dedupSvc := service.NewDeduplicationService(profRepo, dedupRepo, auditSvc)
	authSvc := service.NewAuthService(userRepo, identSvc, profRepo, dedupSvc, auditSvc, cfg.JWTSecret)
	qrSvc := service.NewQRService(qrRepo, identRepo, profRepo, auditSvc, cfg.VerifyURLPrefix)
	verifSvc := service.NewVerificationService(verifRepo, identRepo, profRepo, auditSvc)

	// Handler Layer
	h := handler.NewHandlers(authSvc, identSvc, qrSvc, verifSvc, dedupSvc, auditSvc, profRepo, credRepo)

	// Router Setup
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS(cfg.CorsAllowedOrigins))

	rateLimiter := middleware.NewRateLimiter(100, time.Minute)
	r.Use(rateLimiter.Limit)

	// Health & System
	r.Get("/health", h.Health)
	r.Get("/ready", h.Ready)
	r.Get("/api/v1/docs", serveOpenAPIDocs)

	// Public API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Auth
		r.Post("/auth/request-otp", h.RequestOTP)
		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)

		// Public Verification & Identity Lookup
		r.Get("/identities/{ekaId}", h.GetPublicIdentity)
		r.Post("/qr/verify", h.VerifyQR)

		// Authenticated User Endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(authSvc))

			r.Get("/identities/me", h.GetMyIdentity)
			r.Post("/qr/generate", h.GenerateQR)
			r.Post("/verification-requests", h.CreateVerificationRequest)
			r.Get("/verification-requests/pending", h.ListPendingVerificationRequests)
			r.Post("/verification-requests/{id}/respond", h.RespondVerificationRequest)
		})

		// Admin Privileged Endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(authSvc))
			r.Use(middleware.RequireRole(domain.RoleSystemAdmin))

			r.Get("/admin/identities", h.AdminListIdentities)
			r.Post("/admin/identities/{id}/status", h.AdminUpdateIdentityStatus)
			r.Get("/admin/duplicates", h.AdminListDuplicates)
			r.Post("/admin/duplicates/{id}/resolve", h.AdminResolveDuplicate)
			r.Get("/admin/audit", h.AdminListAudit)
		})
	})

	serverAddr := fmt.Sprintf("%s:%s", cfg.ServerHost, cfg.ServerPort)
	log.Printf("[INFO] EKA ID Platform API listening at http://%s", serverAddr)
	log.Printf("[INFO] Interactive OpenAPI Docs available at http://%s/api/v1/docs", serverAddr)

	if err := http.ListenAndServe(serverAddr, r); err != nil {
		log.Fatalf("[FATAL] Server terminated: %v", err)
	}
}

func seedMemoryStore(m *repository.MemoryStore) {
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)

	// Admin
	adminID := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	_ = m.Users.Create(ctx, &domain.User{
		ID:           adminID,
		Email:        "admin@eka.dev",
		PasswordHash: string(hash),
		Role:         domain.RoleSystemAdmin,
		Status:       domain.IdentityStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	})

	// John Mathew
	johnUserID := uuid.MustParse("b0000000-0000-0000-0000-000000000002")
	_ = m.Users.Create(ctx, &domain.User{
		ID:           johnUserID,
		Email:        "john.mathew@example.com",
		Phone:        "+919876500001",
		PasswordHash: string(hash),
		Role:         domain.RoleUser,
		Status:       domain.IdentityStatusActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	})

	johnIdentID := uuid.MustParse("c0000000-0000-0000-0000-000000000003")
	verifiedAt := time.Now().UTC().Add(-240 * time.Hour)
	_ = m.Identities.Create(ctx, &domain.Identity{
		ID:                johnIdentID,
		EkaID:             "EKA-7K4M-92PX",
		UserID:            johnUserID,
		Status:            domain.IdentityStatusActive,
		VerificationLevel: domain.VerificationTier1Basic,
		VerifiedAt:        &verifiedAt,
		CreatedAt:         verifiedAt,
		UpdatedAt:         time.Now().UTC(),
	})

	_ = m.Profiles.Create(ctx, &domain.Profile{
		IdentityID:      johnIdentID,
		LegalName:       "John Mathew",
		DateOfBirth:     "1992-05-14",
		Gender:          "MALE",
		ProfilePhotoURL: "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=300&h=300&fit=crop&crop=faces",
		Phone:           "+919876500001",
		Email:           "john.mathew@example.com",
		AddressLine1:    "42 Innovation Park, Koramangala",
		City:            "Bengaluru",
		State:           "Karnataka",
		PostalCode:      "560034",
		Country:         "India",
		CreatedAt:       verifiedAt,
		UpdatedAt:       time.Now().UTC(),
	})

	// Sample Credential
	_ = m.Credentials.Create(ctx, &domain.Credential{
		ID:                 uuid.New(),
		IdentityID:         johnIdentID,
		Type:               "EMPLOYMENT",
		IssuerName:         "Acme Technologies Ltd.",
		Status:             "ACTIVE",
		IssuedAt:           time.Now().Add(-365 * 24 * time.Hour),
		VerificationMethod: "CORPORATE_DIGITAL_SIGNATURE",
		Metadata: map[string]interface{}{
			"title":      "Senior Systems Architect",
			"department": "Platform Engineering",
		},
		CreatedAt: time.Now().UTC(),
	})

	// Sample Pending Verification Request from Org
	orgID := uuid.MustParse("d0000000-0000-0000-0000-000000000004")
	_ = m.Organizations.Create(ctx, &domain.Organization{
		ID:         orgID,
		Name:       "Acme Technologies Ltd.",
		Slug:       "acme-tech",
		ApiKeyHash: string(hash),
		Status:     "ACTIVE",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	})

	_ = m.Verification.CreateRequest(ctx, &domain.VerificationRequest{
		ID:              uuid.MustParse("b2222222-2222-2222-2222-222222222222"),
		OrgID:           orgID,
		OrgName:         "Acme Technologies Ltd.",
		IdentityID:      johnIdentID,
		EkaID:           "EKA-7K4M-92PX",
		RequestedScopes: []string{"identity_valid", "name_match", "phone"},
		Purpose:         "Senior Technical Role Background Onboarding",
		Status:          domain.RequestStatusPending,
		ExpiresAt:       time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:       time.Now().UTC(),
	})

	// Initial Audit Events
	_ = m.Audit.Record(ctx, &domain.AuditEvent{
		EventID:      uuid.New(),
		ActorID:      &johnUserID,
		ActorType:    "USER",
		Action:       "IDENTITY_CREATED",
		ResourceType: "IDENTITY",
		ResourceID:   "EKA-7K4M-92PX",
		Result:       "SUCCESS",
		IPAddress:    "127.0.0.1",
		RequestID:    "init-seed-01",
		Metadata:     map[string]interface{}{"eka_id": "EKA-7K4M-92PX"},
		CreatedAt:    verifiedAt,
	})
}

func serveOpenAPIDocs(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
  <title>EKA ID Platform â€” OpenAPI Documentation</title>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: 'https://raw.githubusercontent.com/sauryah/eka-id/main/docs/openapi.yaml',
        dom_id: '#swagger-ui',
      });
    };
  </script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(html))
}