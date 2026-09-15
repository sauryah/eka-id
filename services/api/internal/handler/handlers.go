package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/events"
	"github.com/sauryah/eka-id/services/api/internal/middleware"
	"github.com/sauryah/eka-id/services/api/internal/repository"
	"github.com/sauryah/eka-id/services/api/internal/service"
	"github.com/sauryah/eka-id/services/api/internal/vc"
)

type Handlers struct {
	authSvc   *service.AuthService
	identSvc  *service.IdentityService
	qrSvc     *service.QRService
	verifSvc  *service.VerificationService
	dedupSvc  *service.DeduplicationService
	auditSvc  *service.AuditService
	vcSvc     *service.VCService
	amendSvc  *service.AmendmentService
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
	vcSvc *service.VCService,
	amendSvc *service.AmendmentService,
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
		vcSvc:     vcSvc,
		amendSvc:  amendSvc,
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
		"w3c_vc":    "COMPLIANT_V1_V2",
	})
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]interface{}{
		"status":    "READY",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"features": map[string]string{
			"did_resolver": "ACTIVE",
			"w3c_vc":       "ACTIVE",
			"rate_limiter": "HYBRID_SLIDING_WINDOW",
		},
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
		"message": "Verification code dispatched",
		"dev_otp": otp, // Provided for local development & testing ease
		"target":  body.Target,
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

	// Enrich with W3C DID
	did := fmt.Sprintf("did:eka:%s", identity.EkaID)

	JSON(w, http.StatusOK, map[string]interface{}{
		"did":         did,
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

	// Strictly public view: public status, verification tier, and subject DID
	JSON(w, http.StatusOK, map[string]interface{}{
		"did":                fmt.Sprintf("did:eka:%s", ident.EkaID),
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
		Scopes          []string `json:"scopes"`
		DurationMinutes int      `json:"duration_minutes"`
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

	// Format W3C Verifiable Presentation
	var vp *vc.VerifiablePresentation
	if h.vcSvc != nil {
		vp, _ = h.vcSvc.MintVerifiablePresentation(r.Context(), result.EkaID, result.VerificationLevel, result.DisclosedClaims)
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"status":                  result.Status,
		"did":                     fmt.Sprintf("did:eka:%s", result.EkaID),
		"eka_id":                  result.EkaID,
		"verification_level":      result.VerificationLevel,
		"verified_at":             result.VerifiedAt,
		"legal_name":              result.LegalName,
		"disclosed_claims":        result.DisclosedClaims,
		"verification_date":       result.VerificationDate,
		"verifiable_presentation": vp,
	})
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

	// If OrgID is not specified, default to seeded Acme Org UUID
	if input.OrgID == uuid.Nil {
		input.OrgID = uuid.MustParse("d0000000-0000-0000-0000-000000000004")
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

	// Publish live event to target user topics
	events.GetBroker().Publish(input.EkaID, events.Event{
		Type:     events.EventConsentRequested,
		TargetID: input.EkaID,
		Payload: map[string]interface{}{
			"request_id":       req.ID,
			"eka_id":           req.EkaID,
			"org_id":           req.OrgID,
			"org_name":         req.OrgName,
			"requested_scopes": req.RequestedScopes,
			"purpose":          req.Purpose,
			"created_at":       req.CreatedAt,
		},
		Timestamp: time.Now().UTC(),
	})
	if req.IdentityID != uuid.Nil {
		events.GetBroker().Publish(req.IdentityID.String(), events.Event{
			Type:     events.EventConsentRequested,
			TargetID: req.IdentityID.String(),
			Payload: map[string]interface{}{
				"request_id":       req.ID,
				"eka_id":           req.EkaID,
				"org_id":           req.OrgID,
				"org_name":         req.OrgName,
				"requested_scopes": req.RequestedScopes,
				"purpose":          req.Purpose,
				"created_at":       req.CreatedAt,
			},
			Timestamp: time.Now().UTC(),
		})
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

	// Publish live response event to org / requester
	events.GetBroker().Publish(reqUUID.String(), events.Event{
		Type:     events.EventConsentResponded,
		TargetID: reqUUID.String(),
		Payload: map[string]interface{}{
			"request_id":  reqUUID,
			"approved":    body.Approved,
			"identity_id": claims.IdentityID,
			"result":      result,
		},
		Timestamp: time.Now().UTC(),
	})
	if reqObj, err := h.verifSvc.GetRequestByID(r.Context(), reqUUID); err == nil && reqObj != nil {
		events.GetBroker().Publish(reqObj.OrgID.String(), events.Event{
			Type:     events.EventConsentResponded,
			TargetID: reqObj.OrgID.String(),
			Payload: map[string]interface{}{
				"request_id":  reqUUID,
				"org_id":      reqObj.OrgID,
				"approved":    body.Approved,
				"identity_id": claims.IdentityID,
				"result":      result,
			},
			Timestamp: time.Now().UTC(),
		})
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"status":   "PROCESSED",
		"result":   result,
		"approved": body.Approved,
	})
}

// --- Real-time Server-Sent Events (SSE) Stream ---

func (h *Handlers) EventsStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Valid authentication token required", "")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	broker := events.GetBroker()

	topics := []string{claims.UserID.String()}
	if claims.IdentityID != uuid.Nil {
		topics = append(topics, claims.IdentityID.String())
	}
	if customTopic := r.URL.Query().Get("topic"); customTopic != "" {
		topics = append(topics, customTopic)
	}

	if claims.IdentityID != uuid.Nil {
		if ident, err := h.identSvc.GetByID(r.Context(), claims.IdentityID); err == nil && ident != nil {
			topics = append(topics, ident.EkaID)
		}
	}

	combinedChan := make(chan events.Event, 32)
	var unsubList []struct {
		topic string
		ch    chan events.Event
	}

	for _, topic := range topics {
		ch := broker.Subscribe(topic)
		unsubList = append(unsubList, struct {
			topic string
			ch    chan events.Event
		}{topic: topic, ch: ch})

		go func(c chan events.Event) {
			for ev := range c {
				select {
				case combinedChan <- ev:
				case <-r.Context().Done():
					return
				}
			}
		}(ch)
	}

	defer func() {
		for _, u := range unsubList {
			broker.Unsubscribe(u.topic, u.ch)
		}
	}()

	initMsg := events.Event{
		Type:      "CONNECTED",
		TargetID:  claims.UserID.String(),
		Payload:   map[string]interface{}{"status": "active", "topics": topics},
		Timestamp: time.Now().UTC(),
	}
	_, _ = w.Write(initMsg.ToSSE())
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			_, err := w.Write([]byte(": heartbeat\n\n"))
			if err != nil {
				return
			}
			flusher.Flush()
		case ev := <-combinedChan:
			_, err := w.Write(ev.ToSSE())
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// --- W3C Verifiable Credentials & DID Handlers ---

func (h *Handlers) GetDIDDocument(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	didURI := chi.URLParam(r, "did")
	if didURI == "" {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "DID parameter is required", reqID)
		return
	}

	doc, err := h.vcSvc.ResolveDID(r.Context(), didURI)
	if err != nil {
		if err == service.ErrDIDNotFound {
			ErrorResponse(w, http.StatusNotFound, "DID_NOT_FOUND", "Decentralized Identifier (DID) could not be resolved", reqID)
			return
		}
		ErrorResponse(w, http.StatusBadRequest, "INVALID_DID", err.Error(), reqID)
		return
	}

	w.Header().Set("Content-Type", "application/did+ld+json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(doc)
}

func (h *Handlers) GetCredentialW3C(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	credIDStr := chi.URLParam(r, "id")
	credID, err := uuid.Parse(credIDStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid credential UUID", reqID)
		return
	}

	cred, err := h.credRepo.GetByID(r.Context(), credID)
	if err != nil || cred == nil {
		ErrorResponse(w, http.StatusNotFound, "CREDENTIAL_NOT_FOUND", "Credential not found", reqID)
		return
	}

	ident, err := h.identSvc.GetByID(r.Context(), cred.IdentityID)
	if err != nil || ident == nil {
		ErrorResponse(w, http.StatusNotFound, "IDENTITY_NOT_FOUND", "Subject identity not found", reqID)
		return
	}

	profile, _ := h.profRepo.GetByIdentityID(r.Context(), ident.ID)

	w3cVC, err := h.vcSvc.MintVerifiableCredential(r.Context(), cred, ident, profile)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "VC_MINT_FAILED", "Failed to mint W3C credential", reqID)
		return
	}

	w.Header().Set("Content-Type", "application/credential+ld+json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(w3cVC)
}

func (h *Handlers) VerifyW3CCredential(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	var credential vc.VerifiableCredential
	if err := json.NewDecoder(r.Body).Decode(&credential); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_PAYLOAD", "Malformed W3C Verifiable Credential payload", reqID)
		return
	}

	valid, statusReason, err := h.vcSvc.VerifyW3CCredential(r.Context(), &credential)
	if err != nil || !valid {
		JSON(w, http.StatusOK, map[string]interface{}{
			"valid":         false,
			"status":        "INVALID",
			"reason":        statusReason,
			"verified_at":   time.Now().UTC(),
			"specification": "W3C Verifiable Credentials Data Model v1.1/v2.0",
		})
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"valid":         true,
		"status":        "VERIFIED",
		"issuer":        credential.Issuer,
		"subject":       credential.CredentialSubject.ID,
		"issuance_date": credential.IssuanceDate,
		"verified_at":   time.Now().UTC(),
		"specification": "W3C Verifiable Credentials Data Model v1.1/v2.0",
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

// --- Document & Identity Amendment Handlers ---

func (h *Handlers) UploadDocument(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil || claims.IdentityID == uuid.Nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Identity credentials required", reqID)
		return
	}

	contentType := r.Header.Get("Content-Type")
	var docType, docName, mimeType string
	var fileData []byte

	if strings.Contains(contentType, "multipart/form-data") {
		_ = r.ParseMultipartForm(10 << 20) // 10 MB max
		file, handler, err := r.FormFile("file")
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "FILE_MISSING", "Supporting document file is required", reqID)
			return
		}
		defer file.Close()

		docType = r.FormValue("document_type")
		docName = handler.Filename
		mimeType = handler.Header.Get("Content-Type")
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		fileData, err = io.ReadAll(file)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "READ_ERROR", "Failed to read uploaded file", reqID)
			return
		}
	} else {
		var body struct {
			DocumentType string `json:"document_type"`
			DocumentName string `json:"document_name"`
			MimeType     string `json:"mime_type"`
			FileContent  string `json:"file_content"` // Base64
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.FileContent == "" {
			ErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "Document type, name, and base64 file_content are required", reqID)
			return
		}

		docType = body.DocumentType
		docName = body.DocumentName
		mimeType = body.MimeType
		if mimeType == "" {
			mimeType = "application/pdf"
		}

		// Handle data URL prefix if present
		rawBase64 := body.FileContent
		if commaIdx := strings.Index(rawBase64, ","); commaIdx != -1 {
			rawBase64 = rawBase64[commaIdx+1:]
		}

		decoded, err := base64.StdEncoding.DecodeString(rawBase64)
		if err != nil {
			ErrorResponse(w, http.StatusBadRequest, "DECODE_ERROR", "Invalid base64 document content", reqID)
			return
		}
		fileData = decoded
	}

	doc, err := h.amendSvc.UploadDocument(r.Context(), claims.IdentityID, docType, docName, mimeType, fileData, &claims.UserID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "UPLOAD_FAILED", err.Error(), reqID)
		return
	}

	JSON(w, http.StatusCreated, doc)
}

func (h *Handlers) ListMyDocuments(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil || claims.IdentityID == uuid.Nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Identity credentials required", reqID)
		return
	}

	docs, err := h.amendSvc.ListDocuments(r.Context(), claims.IdentityID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve documents", reqID)
		return
	}

	JSON(w, http.StatusOK, docs)
}

func (h *Handlers) CreateAmendmentRequest(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil || claims.IdentityID == uuid.Nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Identity credentials required", reqID)
		return
	}

	var body struct {
		RequestedChanges map[string]interface{} `json:"requested_changes"`
		Justification    string                 `json:"justification"`
		DocumentIDs      []uuid.UUID            `json:"document_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.RequestedChanges) == 0 {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "Target change attributes are required", reqID)
		return
	}

	req, err := h.amendSvc.CreateAmendmentRequest(r.Context(), claims.IdentityID, body.RequestedChanges, body.Justification, body.DocumentIDs, &claims.UserID)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "AMENDMENT_FAILED", err.Error(), reqID)
		return
	}

	JSON(w, http.StatusCreated, req)
}

func (h *Handlers) ListMyAmendments(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil || claims.IdentityID == uuid.Nil {
		ErrorResponse(w, http.StatusUnauthorized, "UNAUTHORIZED", "Identity credentials required", reqID)
		return
	}

	list, err := h.amendSvc.ListAmendmentsByIdentity(r.Context(), claims.IdentityID)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retrieve amendments", reqID)
		return
	}

	JSON(w, http.StatusOK, list)
}

func (h *Handlers) AdminListAmendments(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	list, total, err := h.amendSvc.ListAllAmendments(r.Context(), limit, offset)
	if err != nil {
		ErrorResponse(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list amendments", reqID)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"amendments": list,
		"total":      total,
	})
}

func (h *Handlers) AdminReviewAmendment(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	claims := middleware.GetUserClaims(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_ID", "Invalid amendment UUID", reqID)
		return
	}

	var body struct {
		Approved        bool   `json:"approved"`
		RejectionReason string `json:"rejection_reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		ErrorResponse(w, http.StatusBadRequest, "INVALID_INPUT", "Review decision required", reqID)
		return
	}

	reviewerID := uuid.Nil
	if claims != nil {
		reviewerID = claims.UserID
	}

	req, err := h.amendSvc.ReviewAmendment(r.Context(), id, body.Approved, body.RejectionReason, reviewerID)
	if err != nil {
		ErrorResponse(w, http.StatusBadRequest, "REVIEW_FAILED", err.Error(), reqID)
		return
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Amendment review processed",
		"amendment": req,
	})
}