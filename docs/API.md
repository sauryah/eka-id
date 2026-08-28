# EKA ID — REST API Reference

Base URL: `http://localhost:8080/api/v1`

All responses use standard JSON wrapping. Privileged endpoints require HTTP Bearer authentication:
```http
Authorization: Bearer <JWT_TOKEN>
```

---

## Endpoints

### 1. Health & Observability
- `GET /health` — Service liveness check.
- `GET /ready` — Service readiness probe.
- `GET /api/v1/docs` — Interactive OpenAPI 3.0 documentation.

### 2. Authentication
- `POST /api/v1/auth/request-otp`
  - Body: `{ "target": "user@example.com" }`
  - Returns: `{ "message": "Verification code dispatched", "dev_otp": "123456" }`
- `POST /api/v1/auth/register`
  - Body: `{ "email": "...", "phone": "...", "password": "...", "legal_name": "...", "date_of_birth": "YYYY-MM-DD", "otp_code": "123456" }`
  - Returns: `{ "token": "...", "identity": { "eka_id": "EKA-..." }, "profile": { ... } }`
- `POST /api/v1/auth/login`
  - Body: `{ "email": "...", "password": "..." }`
  - Returns: JWT session token and user details.

### 3. Identities
- `GET /api/v1/identities/me` [Authorized]
  - Retrieves authenticated identity, profile, and credentials.
- `GET /api/v1/identities/{ekaId}` [Public]
  - Public lookup returning only validity status and verification tier. Zero PII exposed.

### 4. Ephemeral QR Verification
- `POST /api/v1/qr/generate` [Authorized]
  - Body: `{ "scopes": ["identity_valid", "legal_name"], "duration_minutes": 15 }`
  - Returns: `{ "token": "...", "verify_url": "...", "expires_at": "..." }`
- `POST /api/v1/qr/verify` [Public]
  - Body: `{ "token": "..." }`
  - Returns: `{ "status": "VERIFIED", "eka_id": "...", "disclosed_claims": { ... } }`

### 5. Verification Requests (Consent Flow)
- `POST /api/v1/verification-requests` [Authorized Org]
  - Body: `{ "eka_id": "...", "purpose": "...", "requested_scopes": [...] }`
- `GET /api/v1/verification-requests/pending` [Authorized User]
  - Lists pending requests awaiting user consent.
- `POST /api/v1/verification-requests/{id}/respond` [Authorized User]
  - Body: `{ "approved": true }`

### 6. Administration
- `GET /api/v1/admin/identities` [System Admin]
  - Paginated identity list.
- `POST /api/v1/admin/identities/{id}/status` [System Admin]
  - Body: `{ "status": "SUSPENDED" }`
- `GET /api/v1/admin/duplicates` [System Admin]
  - List flagged suspicious duplicate records.
- `POST /api/v1/admin/duplicates/{id}/resolve` [System Admin]
  - Body: `{ "status": "RESOLVED_FALSE_POSITIVE" }`
- `GET /api/v1/admin/audit` [System Admin]
  - Immutable audit event log stream.