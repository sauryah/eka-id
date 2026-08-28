# EKA ID — Technical Architecture

## 1. Architectural Mission
EKA ID establishes a universal digital identity framework founded on the core axiom:
> **One person → One EKA ID → One verifiable digital identity**

EKA ID is an **independent, private/platform-level identity system**. It is strictly decoupled from government registries (such as Aadhaar), avoiding any dependency on external centralized databases while providing cryptographic attestation, zero-PII sharing, and explicit consent mechanisms.

---

## 2. Core Architectural Pillars

### 2.1 Cryptographic Separation of Identifiers
- **Internal Database Identifier**: UUIDv7 / UUIDv4. Used strictly for foreign key integrity, row clustering, and internal service routing. Never exposed in URLs, client state, or public verification screens.
- **Public EKA ID**: Formatted as `EKA-CCCC-CCCC` (e.g. `EKA-7K4M-92PX`).
  - Generated using `crypto/rand` random entropy.
  - Sourced from the **Crockford Base32** alphabet (`0123456789ABCDEFGHJKMNPQRSTVWXYZ`), omitting `I`, `L`, `O`, and `U` to prevent visual ambiguity and accidental profanity.
  - Provides $32^8 \approx 1.1 \times 10^{12}$ (1.1 trillion) distinct permutations.
  - Guaranteed zero personal information encoded inside (no DOB, initials, phone hash, or state codes).

```text
Database Internal:
018e6e5f-d4b2-7000-8431-a87f7397b912 (UUID)
               │
               ▼  (One-way mapping with unique constraint)
Public EKA ID:
EKA-7K4M-92PX
```

### 2.2 Ephemeral, Zero-PII QR Verification
QR codes rendered on digital identity cards or mobile apps contain **no raw PII**.
The QR code encodes strictly an HTTPS URL pointing to a short-lived, signed token:
```text
https://id.eka.dev/verify?token=7f9b8c2d1e4a5061...
```
- **Token Validity**: Default 15 minutes (configurable down to 5 minutes).
- **Scope Restriction**: Only the identity owner’s pre-authorized claims (e.g. `identity_valid`, `legal_name`) are resolved. Fields like phone number or address are physically omitted from the response unless explicitly granted.
- **Single-Use Capability**: Tokens can be invalidated upon first scan to prevent replay attacks.

### 2.3 Consent-Driven Verification Requests
When an organization (employer, bank, platform) requests verification:
1. Organization submits a `VerificationRequest` specifying target `eka_id`, `purpose`, and requested claims.
2. The user reviews the request in their dashboard.
3. The user explicitly clicks **Approve with Consent** or **Deny**.
4. Upon approval, an immutable `VerificationResult` is generated with only the approved claims.
5. All operations are logged to the audit log.

### 2.4 Duplicate Identity Protection (`IdentityDeduplicationService`)
EKA ID implements a pluggable duplicate detection service that computes similarity heuristics:
- Exact Phone Number Match: +50% confidence
- Exact Email Match: +45% confidence
- Exact Legal Name + DOB Match: +55% confidence
If the cumulative confidence exceeds 70%, the record is flagged in `duplicate_flags` for human administrative resolution rather than corrupting existing profiles.

### 2.5 Immutable Audit System
Every critical security event is recorded in the `audit_events` ledger:
- `IDENTITY_CREATED`
- `IDENTITY_VERIFIED`
- `QR_CREATED`
- `QR_VERIFIED`
- `VERIFICATION_REQUESTED`
- `VERIFICATION_APPROVED`
- `VERIFICATION_DENIED`
- `ADMIN_SUSPENDED_IDENTITY`
- `LOGIN_SUCCESS`, `LOGIN_FAILED`

Sensitive credentials (passwords, OTP codes, private keys, auth tokens) are automatically redacted prior to database persistence.