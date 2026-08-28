# EKA ID — Security & Cryptographic Rationale

## Threat Model & Security Controls

### 1. Public Identifier Enumeration Attacks
- **Mitigation**: EKA IDs are 8-character Crockford Base32 strings sourced from `crypto/rand`.
- Permutation space exceeds 1.09 trillion combinations.
- Rate limiting on `/identities/{ekaId}` (100 requests per minute per IP) prevents dictionary sweeps.

### 2. Physical & Optical QR Harvesting
- **Mitigation**: Static QR codes that print personal data directly are strictly banned.
- All QR codes encode ephemeral HTTPS URLs with signed tokens.
- Tokens expire within 5 to 60 minutes and enforce selective disclosure.

### 3. Password & Credential Security
- Passwords are encrypted using standard **bcrypt** with default cost factor 10.
- Secrets, passwords, OTPs, and auth tokens are stripped and redacted (`[REDACTED]`) before being recorded into audit ledgers.

### 4. Role-Based Access Control (RBAC)
- Strict boundary between `USER`, `ORG_ADMIN`, and `SYSTEM_ADMIN`.
- System admin routes enforce `RequireRole(domain.RoleSystemAdmin)`.
- Organizations can never query identities without explicit owner consent or active token presentation.

### 5. Vulnerability Reporting
Please report any potential vulnerabilities to security@eka.dev.