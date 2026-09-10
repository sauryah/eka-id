# EKA ID — REST API Reference

Base URL: `http://localhost:8080/api/v1`

All responses use standard JSON wrapping. Privileged endpoints require HTTP Bearer authentication:
```http
Authorization: Bearer <JWT_TOKEN>
```

### Rate Limiting Headers
All API responses include standard sliding-window rate limit headers:
- `X-RateLimit-Limit`: Maximum allowed requests within window (e.g. 100).
- `X-RateLimit-Remaining`: Number of remaining requests available.
- `X-RateLimit-Reset`: Duration in seconds until window resets.
- `Retry-After`: Returned on `HTTP 429 Too Many Requests` indicating wait seconds.

---

## Endpoints

### 1. Health & Observability
- `GET /health` — Service liveness check (returns W3C compliance status).
- `GET /ready` — Service readiness probe (checks Redis, DB, and feature engines).
- `GET /api/v1/docs` — Interactive OpenAPI 3.0 documentation.

### 2. Authentication
- `POST /api/v1/auth/request-otp` [Rate Limited: 15 req/min]
  - Body: `{ "target": "user@example.com" }`
  - Returns: `{ "message": "Verification code dispatched", "dev_otp": "123456" }`
- `POST /api/v1/auth/register` [Rate Limited: 15 req/min]
  - Body: `{ "email": "...", "phone": "...", "password": "...", "legal_name": "...", "date_of_birth": "YYYY-MM-DD", "otp_code": "123456" }`
  - Returns: `{ "token": "...", "identity": { "eka_id": "EKA-..." }, "profile": { ... } }`
- `POST /api/v1/auth/login` [Rate Limited: 15 req/min]
  - Body: `{ "email": "...", "password": "..." }`
  - Returns: JWT session token and user details.

### 3. Identities & Decentralized Identifiers (DID)
- `GET /api/v1/identities/me` [Authorized]
  - Retrieves authenticated identity, subject DID (`did:eka:<eka_id>`), profile, and credentials.
- `GET /api/v1/identities/{ekaId}` [Public, 60 req/min]
  - Public lookup returning validity status, subject DID, and verification tier. Zero PII exposed.
- `GET /api/v1/did/{did}` [Public]
  - Resolves `did:eka:<eka_id>`, `did:eka:org:<slug>`, or `did:eka:issuer:platform` to an official W3C DID Core 1.0 Document (`application/did+ld+json`).

### 4. W3C Verifiable Credentials & Ephemeral QR Verification
- `GET /api/v1/credentials/{id}/w3c` [Public]
  - Exports an individual credential as a signed W3C Verifiable Credential (`application/credential+ld+json`).
- `POST /api/v1/credentials/verify-w3c` [Public]
  - Cryptographically verifies a W3C Verifiable Credential payload, checking signatures, issuer DID, and expiration date.
- `POST /api/v1/qr/generate` [Authorized]
  - Body: `{ "scopes": ["identity_valid", "legal_name"], "duration_minutes": 15 }`
  - Returns: `{ "token": "...", "verify_url": "...", "expires_at": "..." }`
- `POST /api/v1/qr/verify` [Public, 60 req/min]
  - Body: `{ "token": "..." }`
  - Returns: `{ "status": "VERIFIED", "did": "did:eka:...", "disclosed_claims": { ... }, "verifiable_presentation": { ... } }`

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
  - List flagged suspicious duplicate records (including biometric facial embedding matches).
- `POST /api/v1/admin/duplicates/{id}/resolve` [System Admin]
  - Body: `{ "status": "RESOLVED_FALSE_POSITIVE" }`
- `GET /api/v1/admin/audit` [System Admin]
  - Immutable audit event log stream.