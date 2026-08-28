package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/middleware"
	"github.com/sauryah/eka-id/services/api/internal/repository"
	"github.com/sauryah/eka-id/services/api/internal/service"
)

type Handlers struct {
	authSvc   *service.AuthService
	identSvc  *service.IdentityService
	qrSvc     *service.QRService
	verifSvc  *service.VerificationService
	dedupSvc  *service.DeduplicationService
	auditSvc  *service.AuditService
	profRepo  repository.ProfileRepository
	credRepo  repository.CredentialRepository
}

func NewHandlers(
	authSvc *service.AuthService,
	identSvc *service.IdentityService,
	qrSvc *service.QRService,
	verifSvc *service.VerificationService,
	dedupSvc *service.DeduplicationService,
	auditSvc *service.AuditService,
	profRepo repository.ProfileRepository,
	credRepo repository.CredentialRepository,
) *Handlers {
	return &Handlers{
		authSvc:   authSvc,
		identSvc:  identSvc,
		qrSvc:     qrSvc,
		verifSvc:  verifSvc,
		dedupSvc:  dedupSvc,
		auditSvc:  auditSvc,
		profRepo:  profRepo,
		credRepo:  credRepo,
	}
}

// Helpers
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func ErrorResponse(w http.ResponseWriter, status int, code, message, reqID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":       code,
			"message":    message,
			"request_id": reqID,
		},
	})
}

// --- Health Handlers ---

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]interface{}{
		"status":    "UP",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]interface{}{
		"status":    "READY",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// --- Auth Handlers ---

func (h *Handlers) RequestOTP(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	var body struct {
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Target == "" {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "Target phone or email is required", reqID)
		return
	}

	otp, err := h.authSvc.RequestOTP(r.Context(), body.Target)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate verification code", reqID)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Verification code dispatched",
		"dev_otp":   otp, // Provided for local development & testing ease
		"target":    body.Target,
	})
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	var input service.RegistrationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "Malformed registration payload", reqID)
		return
	}

	if input.Email == "" || input.Password == "" || input.LegalName == "" || input.DateOfBirth == "" {
		ErrorResponse(w, http.StatusBadRequest, "VALIDATION_FAILED", "Email, password, legal name, and date of birth are required", reqID)
		return
	}

	ip := r.RemoteAddr
	ua := r.UserAgent()

	result, err := h.authSvc.Register(r.Context(), input, ip, ua, reqID)
	if err != nil {
		if err == service.ErrInvalidOTP {
			ErrorResponse(w, http.StatusUnauthorized, "INVALID_OTP", "The verification code is invalid or expired. (Use 123456 for dev)", reqID)
			return
		}
		if err == service.ErrUserAlreadyExists {
			ErrorResponse(w, http.StatusConflict, "USER_EXISTS", "A user is already registered with this email.", reqID)
			return
		}
		ErrorResponse(w, http.StatusInternalServerError, "REGISTRATION_FAILED", err.Error(), reqID)
		return
	}

	JSON(w, http.StatusCreated, result)
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Password == "" {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "Email and password are required", reqID)
		return
	}

	ip := r.RemoteAddr
	ua := r.UserAgent()

	result, err := h.authSvc.Login(r.Context(), body.Email, body.Password, ip, ua, reqID)
	if err != nil {
		ErrorResponse(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Incorrect email or password.", reqID)
		return
	}

	JSON(w, http.StatusOK, result)
}

// --- Identity Handlers ---

func (h *Handlers) GetMyIdentity(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized", reqID)
		return
	}

	identity, err := h.identSvc.GetByUserID(r.Context(), claims.UserID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "IDENTITY_NOT_FOUND", "No identity associated with current user", reqID)
		return
	}

	profile, _ := h.profRepo.GetByIdentityID(r.Context(), identity.ID)
	creds, _ := h.credRepo.ListByIdentityID(r.Context(), identity.ID)

	JSON(w, http.StatusOK, map[string]interface{}{
		"identity":    identity,
		"profile":     profile,
		"credentials": creds,
	})
}

func (h *Handlers) GetPublicIdentity(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	ekaID := chi.URLParam(r, "ekaId")

	ident, err := h.identSvc.GetByEkaID(r.Context(), ekaID)
	if err != nil {
		ErrorResponse(w, http.StatusNotFound, "IDENTITY_NOT_FOUND", "Identity could not be found", reqID)
		return
	}

	// Strictly public view: only public status and verification level
	JSON(w, http.StatusOK, map[string]interface{}{
		"eka_id":             ident.EkaID,
		"status":             ident.Status,
		"verification_level": ident.VerificationLevel,
		"verified_at":        ident.VerifiedAt,
	})
}

// --- QR Handlers ---

func (h *Handlers) GenerateQR(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil || claims.IdentityID == uuid.Nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Valid identity required to generate QR", reqID)
		return
	}

	var body struct {
		Scopes         []string `json:"scopes"`
		DurationMinutes int     `json:"duration_minutes"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	duration := 15 * time.Minute
	if body.DurationMinutes > 0 {
		duration = time.Duration(body.DurationMinutes) * time.Minute
	}

	resp, err := h.qrSvc.GenerateVerificationToken(r.Context(), claims.IdentityID, body.Scopes, duration, &claims.UserID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "QR_GENERATION_FAILED", "Failed to generate QR token", reqID)
		return
	}

	JSON(w, http.StatusOK, resp)
}

func (h *Handlers) VerifyQR(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "Verification token is required", reqID)
		return
	}

	ip := r.RemoteAddr
	ua := r.UserAgent()

	result, err := h.qrSvc.VerifyToken(r.Context(), body.Token, ip, ua, reqID)
	if err != nil {
		if err == service.ErrQRTokenExpired {
			ErrorResponse(w, http.StatusGone, "TOKEN_EXPIRED", "This verification request has expired.", reqID)
			return
		}
		if err == service.ErrIdentityInactive {
			ErrorResponse(w, http.StatusForbidden, "IDENTITY_INACTIVE", "Identity is not active or has been suspended.", reqID)
			return
		}
		ErrorResponse(w, http.StatusNotFound, "TOKEN_INVALID", "This verification request is no longer valid or does not exist.", reqID)
		return
	}

	JSON(w, http.StatusOK, result)
}

// --- Verification Request Handlers ---

func (h *Handlers) CreateVerificationRequest(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())

	var input service.CreateVerificationRequestInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.EkaID == "" {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "Target EKA ID and requested claims are required", reqID)
		return
	}

	// For dev mock: if orgID is nil, generate one
	if input.OrgID == uuid.Nil {
		input.OrgID = uuid.New()
	}

	var actorID *uuid.UUID
	if claims != nil {
		actorID = &claims.UserID
	}

	req, err := h.verifSvc.CreateRequest(r.Context(), input, actorID)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "REQUEST_CREATION_FAILED", err.Error(), reqID)
		return
	}

	JSON(w, http.StatusCreated, req)
}

func (h *Handlers) ListPendingVerificationRequests(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil || claims.IdentityID == uuid.Nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Identity required", reqID)
		return
	}

	list, err := h.verifSvc.ListPendingByIdentity(r.Context(), claims.IdentityID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve verification requests", reqID)
		return
	}

	JSON(w, http.StatusOK, list)
}

func (h *Handlers) RespondVerificationRequest(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil || claims.IdentityID == uuid.Nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Identity required", reqID)
		return
	}

	reqUUIDStr := chi.URLParam(r, "id")
	reqUUID, err := uuid.Parse(reqUUIDStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid request UUID", reqID)
		return
	}

	var body struct {
		Approved bool `json:"approved"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	result, err := h.verifSvc.RespondRequest(r.Context(), reqUUID, claims.IdentityID, body.Approved, &claims.UserID)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "ACTION_FAILED", err.Error(), reqID)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"status":  "PROCESSED",
		"result":  result,
		"approved": body.Approved,
	})
}

// --- Admin Handlers ---

func (h *Handlers) AdminListIdentities(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	list, total, err := h.identSvc.List(r.Context(), limit, offset)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list identities", reqID)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"identities": list,
		"total":      total,
	})
}

func (h *Handlers) AdminUpdateIdentityStatus(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid identity UUID", reqID)
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Status == "" {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "New status is required", reqID)
		return
	}

	var actorID *uuid.UUID
	if claims != nil {
		actorID = &claims.UserID
	}

	if err := h.identSvc.UpdateStatus(r.Context(), id, strings.ToUpper(body.Status), actorID); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), reqID)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Identity status updated successfully"})
}

func (h *Handlers) AdminListDuplicates(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	flags, err := h.dedupSvc.ListPendingFlags(r.Context())
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list duplicate flags", reqID)
		return
	}

	JSON(w, http.StatusOK, flags)
}

func (h *Handlers) AdminResolveDuplicate(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	flagIDStr := chi.URLParam(r, "id")
	flagID, err := uuid.Parse(flagIDStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid flag UUID", reqID)
		return
	}

	var body struct {
		Status string `json:"status"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	reviewerID := uuid.Nil
	if claims != nil {
		reviewerID = claims.UserID
	}

	if err := h.dedupSvc.ResolveFlag(r.Context(), flagID, body.Status, reviewerID); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "RESOLVE_FAILED", err.Error(), reqID)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Duplicate flag resolved"})
}

func (h *Handlers) AdminListAudit(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	events, total, err := h.auditSvc.ListEvents(r.Context(), limit, offset, nil, "")
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list audit events", reqID)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  total,
	})
}