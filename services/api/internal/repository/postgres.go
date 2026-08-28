package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
)

type PostgresStore struct {
	db            *sql.DB
	Users         *PostgresUserRepo
	Identities    *PostgresIdentityRepo
	Profiles      *PostgresProfileRepo
	Organizations *PostgresOrganizationRepo
	Verification  *PostgresVerificationRepo
	QR            *PostgresQRRepo
	Credentials   *PostgresCredentialRepo
	Audit         *PostgresAuditRepo
	Duplicates    *PostgresDuplicateRepo
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{
		db:            db,
		Users:         &PostgresUserRepo{db: db},
		Identities:    &PostgresIdentityRepo{db: db},
		Profiles:      &PostgresProfileRepo{db: db},
		Organizations: &PostgresOrganizationRepo{db: db},
		Verification:  &PostgresVerificationRepo{db: db},
		QR:            &PostgresQRRepo{db: db},
		Credentials:   &PostgresCredentialRepo{db: db},
		Audit:         &PostgresAuditRepo{db: db},
		Duplicates:    &PostgresDuplicateRepo{db: db},
	}
}

func (s *PostgresStore) AutoMigrate(ctx context.Context) error {
	schemaSQL := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		email VARCHAR(255) UNIQUE NOT NULL,
		phone VARCHAR(30),
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(30) NOT NULL DEFAULT 'USER',
		status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS identities (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		eka_id VARCHAR(14) UNIQUE NOT NULL,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
		verification_level VARCHAR(30) NOT NULL DEFAULT 'TIER_1_BASIC',
		verified_at TIMESTAMPTZ DEFAULT NOW(),
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_identities_eka_id ON identities(eka_id);
	CREATE INDEX IF NOT EXISTS idx_identities_user_id ON identities(user_id);
	CREATE INDEX IF NOT EXISTS idx_identities_status ON identities(status);

	CREATE TABLE IF NOT EXISTS profiles (
		identity_id UUID PRIMARY KEY REFERENCES identities(id) ON DELETE CASCADE,
		legal_name VARCHAR(150) NOT NULL,
		date_of_birth DATE NOT NULL,
		gender VARCHAR(30),
		profile_photo_url TEXT,
		phone VARCHAR(30),
		email VARCHAR(255),
		address_line1 VARCHAR(255),
		city VARCHAR(100),
		state VARCHAR(100),
		postal_code VARCHAR(30),
		country VARCHAR(50) DEFAULT 'India',
		metadata JSONB DEFAULT '{}'::jsonb,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_profiles_phone ON profiles(phone);
	CREATE INDEX IF NOT EXISTS idx_profiles_email ON profiles(email);
	CREATE INDEX IF NOT EXISTS idx_profiles_legal_name ON profiles(legal_name);

	CREATE TABLE IF NOT EXISTS organizations (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		name VARCHAR(150) NOT NULL,
		slug VARCHAR(100) UNIQUE NOT NULL,
		api_key_hash VARCHAR(255) NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
		webhook_url TEXT,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS organization_members (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		role VARCHAR(30) NOT NULL DEFAULT 'MEMBER',
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		UNIQUE(org_id, user_id)
	);

	CREATE TABLE IF NOT EXISTS verification_requests (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
		identity_id UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
		requested_scopes JSONB NOT NULL,
		purpose TEXT NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
		approved_at TIMESTAMPTZ,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_verif_req_identity ON verification_requests(identity_id);
	CREATE INDEX IF NOT EXISTS idx_verif_req_org ON verification_requests(org_id);
	CREATE INDEX IF NOT EXISTS idx_verif_req_status ON verification_requests(status);

	CREATE TABLE IF NOT EXISTS verification_results (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		request_id UUID NOT NULL REFERENCES verification_requests(id) ON DELETE CASCADE,
		disclosed_claims JSONB NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'VALID',
		verified_by_actor_id UUID,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS qr_tokens (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		token VARCHAR(128) UNIQUE NOT NULL,
		identity_id UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
		allowed_scopes JSONB NOT NULL,
		is_used BOOLEAN NOT NULL DEFAULT FALSE,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_qr_tokens_token ON qr_tokens(token);

	CREATE TABLE IF NOT EXISTS credentials (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		identity_id UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
		type VARCHAR(60) NOT NULL,
		issuer_name VARCHAR(150) NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
		issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMPTZ,
		verification_method VARCHAR(100),
		metadata JSONB DEFAULT '{}'::jsonb,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_credentials_identity ON credentials(identity_id);

	CREATE TABLE IF NOT EXISTS audit_events (
		event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		actor_id UUID,
		actor_type VARCHAR(30) NOT NULL,
		action VARCHAR(60) NOT NULL,
		resource_type VARCHAR(60) NOT NULL,
		resource_id VARCHAR(60) NOT NULL,
		result VARCHAR(30) NOT NULL,
		ip_address VARCHAR(45),
		user_agent TEXT,
		request_id VARCHAR(64),
		metadata JSONB DEFAULT '{}'::jsonb,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_events(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_events(action);
	CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_events(resource_type, resource_id);
	CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_events(actor_type, actor_id);

	CREATE TABLE IF NOT EXISTS duplicate_flags (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		identity_id UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
		suspected_duplicate_id UUID NOT NULL REFERENCES identities(id) ON DELETE CASCADE,
		confidence_score NUMERIC(5, 2) NOT NULL,
		match_reasons JSONB NOT NULL,
		status VARCHAR(30) NOT NULL DEFAULT 'PENDING_REVIEW',
		reviewed_by UUID REFERENCES users(id),
		reviewed_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	`
	_, err := s.db.ExecContext(ctx, schemaSQL)
	return err
}

// --- User Repository ---

type PostgresUserRepo struct { db *sql.DB }

func (r *PostgresUserRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, phone, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.Phone, user.PasswordHash, user.Role, user.Status, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *PostgresUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `SELECT id, email, phone, password_hash, role, status, created_at, updated_at FROM users WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	var u domain.User
	var phone sql.NullString
	if err := row.Scan(&u.ID, &u.Email, &phone, &u.PasswordHash, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	if phone.Valid {
		u.Phone = phone.String
	}
	return &u, nil
}

func (r *PostgresUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, phone, password_hash, role, status, created_at, updated_at FROM users WHERE LOWER(email) = LOWER($1)`
	row := r.db.QueryRowContext(ctx, query, email)
	var u domain.User
	var phone sql.NullString
	if err := row.Scan(&u.ID, &u.Email, &phone, &u.PasswordHash, &u.Role, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	if phone.Valid {
		u.Phone = phone.String
	}
	return &u, nil
}

func (r *PostgresUserRepo) Update(ctx context.Context, user *domain.User) error {
	query := `UPDATE users SET email = $1, phone = $2, role = $3, status = $4, updated_at = NOW() WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, user.Email, user.Phone, user.Role, user.Status, user.ID)
	return err
}

// --- Identity Repository ---

type PostgresIdentityRepo struct { db *sql.DB }

func (r *PostgresIdentityRepo) Create(ctx context.Context, id *domain.Identity) error {
	query := `
		INSERT INTO identities (id, eka_id, user_id, status, verification_level, verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		id.ID, id.EkaID, id.UserID, id.Status, id.VerificationLevel, id.VerifiedAt, id.CreatedAt, id.UpdatedAt,
	)
	return err
}

func (r *PostgresIdentityRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Identity, error) {
	query := `SELECT id, eka_id, user_id, status, verification_level, verified_at, created_at, updated_at FROM identities WHERE id = $1`
	return r.scanIdentity(r.db.QueryRowContext(ctx, query, id))
}

func (r *PostgresIdentityRepo) GetByEkaID(ctx context.Context, ekaID string) (*domain.Identity, error) {
	query := `SELECT id, eka_id, user_id, status, verification_level, verified_at, created_at, updated_at FROM identities WHERE eka_id = $1`
	return r.scanIdentity(r.db.QueryRowContext(ctx, query, ekaID))
}

func (r *PostgresIdentityRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Identity, error) {
	query := `SELECT id, eka_id, user_id, status, verification_level, verified_at, created_at, updated_at FROM identities WHERE user_id = $1`
	return r.scanIdentity(r.db.QueryRowContext(ctx, query, userID))
}

func (r *PostgresIdentityRepo) scanIdentity(row *sql.Row) (*domain.Identity, error) {
	var ident domain.Identity
	var verifiedAt sql.NullTime
	if err := row.Scan(&ident.ID, &ident.EkaID, &ident.UserID, &ident.Status, &ident.VerificationLevel, &verifiedAt, &ident.CreatedAt, &ident.UpdatedAt); err != nil {
		return nil, err
	}
	if verifiedAt.Valid {
		ident.VerifiedAt = &verifiedAt.Time
	}
	return &ident, nil
}

func (r *PostgresIdentityRepo) List(ctx context.Context, limit, offset int) ([]*domain.Identity, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM identities`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT id, eka_id, user_id, status, verification_level, verified_at, created_at, updated_at FROM identities ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []*domain.Identity
	for rows.Next() {
		var ident domain.Identity
		var verifiedAt sql.NullTime
		if err := rows.Scan(&ident.ID, &ident.EkaID, &ident.UserID, &ident.Status, &ident.VerificationLevel, &verifiedAt, &ident.CreatedAt, &ident.UpdatedAt); err != nil {
			return nil, 0, err
		}
		if verifiedAt.Valid {
			ident.VerifiedAt = &verifiedAt.Time
		}
		results = append(results, &ident)
	}
	return results, total, nil
}

func (r *PostgresIdentityRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE identities SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

func (r *PostgresIdentityRepo) UpdateVerificationLevel(ctx context.Context, id uuid.UUID, level string, verifiedAt *time.Time) error {
	query := `UPDATE identities SET verification_level = $1, verified_at = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, level, verifiedAt, id)
	return err
}

// --- Profile Repository ---

type PostgresProfileRepo struct { db *sql.DB }

func (r *PostgresProfileRepo) Create(ctx context.Context, p *domain.Profile) error {
	metaJSON, _ := json.Marshal(p.Metadata)
	query := `
		INSERT INTO profiles (identity_id, legal_name, date_of_birth, gender, profile_photo_url, phone, email, address_line1, city, state, postal_code, country, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`
	_, err := r.db.ExecContext(ctx, query,
		p.IdentityID, p.LegalName, p.DateOfBirth, p.Gender, p.ProfilePhotoURL, p.Phone, p.Email, p.AddressLine1, p.City, p.State, p.PostalCode, p.Country, metaJSON, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

func (r *PostgresProfileRepo) GetByIdentityID(ctx context.Context, identityID uuid.UUID) (*domain.Profile, error) {
	query := `
		SELECT identity_id, legal_name, TO_CHAR(date_of_birth, 'YYYY-MM-DD'), gender, profile_photo_url, phone, email, address_line1, city, state, postal_code, country, metadata, created_at, updated_at
		FROM profiles WHERE identity_id = $1
	`
	row := r.db.QueryRowContext(ctx, query, identityID)
	var p domain.Profile
	var gender, photo, phone, email, addr, city, state, postal, country sql.NullString
	var metaBytes []byte
	err := row.Scan(&p.IdentityID, &p.LegalName, &p.DateOfBirth, &gender, &photo, &phone, &email, &addr, &city, &state, &postal, &country, &metaBytes, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if gender.Valid { p.Gender = gender.String }
	if photo.Valid { p.ProfilePhotoURL = photo.String }
	if phone.Valid { p.Phone = phone.String }
	if email.Valid { p.Email = email.String }
	if addr.Valid { p.AddressLine1 = addr.String }
	if city.Valid { p.City = city.String }
	if state.Valid { p.State = state.String }
	if postal.Valid { p.PostalCode = postal.String }
	if country.Valid { p.Country = country.String }
	if len(metaBytes) > 0 {
		_ = json.Unmarshal(metaBytes, &p.Metadata)
	}
	return &p, nil
}

func (r *PostgresProfileRepo) FindByPhone(ctx context.Context, phone string) ([]*domain.Profile, error) {
	query := `SELECT identity_id, legal_name, TO_CHAR(date_of_birth, 'YYYY-MM-DD'), phone, email FROM profiles WHERE phone = $1`
	rows, err := r.db.QueryContext(ctx, query, phone)
	if err != nil { return nil, err }
	defer rows.Close()
	var results []*domain.Profile
	for rows.Next() {
		var p domain.Profile
		var ph, em sql.NullString
		if err := rows.Scan(&p.IdentityID, &p.LegalName, &p.DateOfBirth, &ph, &em); err == nil {
			if ph.Valid { p.Phone = ph.String }
			if em.Valid { p.Email = em.String }
			results = append(results, &p)
		}
	}
	return results, nil
}

func (r *PostgresProfileRepo) FindByEmail(ctx context.Context, email string) ([]*domain.Profile, error) {
	query := `SELECT identity_id, legal_name, TO_CHAR(date_of_birth, 'YYYY-MM-DD'), phone, email FROM profiles WHERE LOWER(email) = LOWER($1)`
	rows, err := r.db.QueryContext(ctx, query, email)
	if err != nil { return nil, err }
	defer rows.Close()
	var results []*domain.Profile
	for rows.Next() {
		var p domain.Profile
		var ph, em sql.NullString
		if err := rows.Scan(&p.IdentityID, &p.LegalName, &p.DateOfBirth, &ph, &em); err == nil {
			if ph.Valid { p.Phone = ph.String }
			if em.Valid { p.Email = em.String }
			results = append(results, &p)
		}
	}
	return results, nil
}

func (r *PostgresProfileRepo) FindPotentialDuplicates(ctx context.Context, legalName, dob, phone, email string) ([]*domain.Profile, error) {
	query := `
		SELECT identity_id, legal_name, TO_CHAR(date_of_birth, 'YYYY-MM-DD'), phone, email, profile_photo_url, metadata
		FROM profiles
		WHERE (phone = $1 AND $1 != '')
		   OR (email = $2 AND $2 != '')
		   OR (LOWER(legal_name) = LOWER($3) AND date_of_birth = $4::date)
		   OR (metadata ? 'face_embedding')
		   OR (profile_photo_url IS NOT NULL AND profile_photo_url != '')
	`
	rows, err := r.db.QueryContext(ctx, query, phone, email, legalName, dob)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*domain.Profile
	for rows.Next() {
		var p domain.Profile
		var pPhone, pEmail, pPhoto sql.NullString
		var metaBytes []byte
		if err := rows.Scan(&p.IdentityID, &p.LegalName, &p.DateOfBirth, &pPhone, &pEmail, &pPhoto, &metaBytes); err == nil {
			if pPhone.Valid { p.Phone = pPhone.String }
			if pEmail.Valid { p.Email = pEmail.String }
			if pPhoto.Valid { p.ProfilePhotoURL = pPhoto.String }
			if len(metaBytes) > 0 {
				_ = json.Unmarshal(metaBytes, &p.Metadata)
			}
			results = append(results, &p)
		}
	}
	return results, nil
}

func (r *PostgresProfileRepo) Update(ctx context.Context, p *domain.Profile) error {
	metaJSON, _ := json.Marshal(p.Metadata)
	query := `
		UPDATE profiles SET
			legal_name = $1, date_of_birth = $2, gender = $3, profile_photo_url = $4,
			phone = $5, email = $6, address_line1 = $7, city = $8, state = $9,
			postal_code = $10, country = $11, metadata = $12, updated_at = NOW()
		WHERE identity_id = $13
	`
	_, err := r.db.ExecContext(ctx, query,
		p.LegalName, p.DateOfBirth, p.Gender, p.ProfilePhotoURL, p.Phone, p.Email,
		p.AddressLine1, p.City, p.State, p.PostalCode, p.Country, metaJSON, p.IdentityID,
	)
	return err
}

// --- Organization Repository ---

type PostgresOrganizationRepo struct { db *sql.DB }

func (r *PostgresOrganizationRepo) Create(ctx context.Context, org *domain.Organization) error {
	query := `INSERT INTO organizations (id, name, slug, api_key_hash, status, webhook_url, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, org.ID, org.Name, org.Slug, org.ApiKeyHash, org.Status, org.WebhookURL, org.CreatedAt, org.UpdatedAt)
	return err
}

func (r *PostgresOrganizationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	query := `SELECT id, name, slug, api_key_hash, status, webhook_url, created_at, updated_at FROM organizations WHERE id = $1`
	var o domain.Organization
	var hook sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(&o.ID, &o.Name, &o.Slug, &o.ApiKeyHash, &o.Status, &hook, &o.CreatedAt, &o.UpdatedAt)
	if err != nil { return nil, err }
	if hook.Valid { o.WebhookURL = hook.String }
	return &o, nil
}

func (r *PostgresOrganizationRepo) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	query := `SELECT id, name, slug, api_key_hash, status, webhook_url, created_at, updated_at FROM organizations WHERE slug = $1`
	var o domain.Organization
	var hook sql.NullString
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&o.ID, &o.Name, &o.Slug, &o.ApiKeyHash, &o.Status, &hook, &o.CreatedAt, &o.UpdatedAt)
	if err != nil { return nil, err }
	if hook.Valid { o.WebhookURL = hook.String }
	return &o, nil
}

func (r *PostgresOrganizationRepo) List(ctx context.Context) ([]*domain.Organization, error) {
	query := `SELECT id, name, slug, api_key_hash, status, webhook_url, created_at, updated_at FROM organizations`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []*domain.Organization
	for rows.Next() {
		var o domain.Organization
		var hook sql.NullString
		if err := rows.Scan(&o.ID, &o.Name, &o.Slug, &o.ApiKeyHash, &o.Status, &hook, &o.CreatedAt, &o.UpdatedAt); err == nil {
			if hook.Valid { o.WebhookURL = hook.String }
			list = append(list, &o)
		}
	}
	return list, nil
}

// --- QR Token Repository ---

type PostgresQRRepo struct { db *sql.DB }

func (r *PostgresQRRepo) CreateToken(ctx context.Context, qr *domain.QRToken) error {
	scopesJSON, _ := json.Marshal(qr.AllowedScopes)
	query := `
		INSERT INTO qr_tokens (id, token, identity_id, allowed_scopes, is_used, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		qr.ID, qr.Token, qr.IdentityID, scopesJSON, qr.IsUsed, qr.ExpiresAt, qr.CreatedAt,
	)
	return err
}

func (r *PostgresQRRepo) GetToken(ctx context.Context, token string) (*domain.QRToken, error) {
	query := `SELECT id, token, identity_id, allowed_scopes, is_used, expires_at, created_at FROM qr_tokens WHERE token = $1`
	row := r.db.QueryRowContext(ctx, query, token)
	var qr domain.QRToken
	var scopesBytes []byte
	if err := row.Scan(&qr.ID, &qr.Token, &qr.IdentityID, &scopesBytes, &qr.IsUsed, &qr.ExpiresAt, &qr.CreatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(scopesBytes, &qr.AllowedScopes)
	return &qr, nil
}

func (r *PostgresQRRepo) MarkTokenUsed(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE qr_tokens SET is_used = TRUE WHERE token = $1`, token)
	return err
}

// --- Verification Requests & Results ---

type PostgresVerificationRepo struct { db *sql.DB }

func (r *PostgresVerificationRepo) CreateRequest(ctx context.Context, req *domain.VerificationRequest) error {
	scopesJSON, _ := json.Marshal(req.RequestedScopes)
	query := `
		INSERT INTO verification_requests (id, org_id, identity_id, requested_scopes, purpose, status, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		req.ID, req.OrgID, req.IdentityID, scopesJSON, req.Purpose, req.Status, req.ExpiresAt, req.CreatedAt,
	)
	return err
}

func (r *PostgresVerificationRepo) GetRequestByID(ctx context.Context, id uuid.UUID) (*domain.VerificationRequest, error) {
	query := `
		SELECT vr.id, vr.org_id, o.name, vr.identity_id, i.eka_id, vr.requested_scopes, vr.purpose, vr.status, vr.approved_at, vr.expires_at, vr.created_at
		FROM verification_requests vr
		JOIN organizations o ON o.id = vr.org_id
		JOIN identities i ON i.id = vr.identity_id
		WHERE vr.id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)
	var req domain.VerificationRequest
	var approvedAt sql.NullTime
	var scopesBytes []byte
	err := row.Scan(&req.ID, &req.OrgID, &req.OrgName, &req.IdentityID, &req.EkaID, &scopesBytes, &req.Purpose, &req.Status, &approvedAt, &req.ExpiresAt, &req.CreatedAt)
	if err != nil {
		return nil, err
	}
	if approvedAt.Valid {
		req.ApprovedAt = &approvedAt.Time
	}
	_ = json.Unmarshal(scopesBytes, &req.RequestedScopes)
	return &req, nil
}

func (r *PostgresVerificationRepo) ListRequestsByIdentity(ctx context.Context, identityID uuid.UUID) ([]*domain.VerificationRequest, error) {
	query := `
		SELECT vr.id, vr.org_id, o.name, vr.identity_id, i.eka_id, vr.requested_scopes, vr.purpose, vr.status, vr.approved_at, vr.expires_at, vr.created_at
		FROM verification_requests vr
		JOIN organizations o ON o.id = vr.org_id
		JOIN identities i ON i.id = vr.identity_id
		WHERE vr.identity_id = $1
		ORDER BY vr.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, identityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.VerificationRequest
	for rows.Next() {
		var req domain.VerificationRequest
		var approvedAt sql.NullTime
		var scopesBytes []byte
		if err := rows.Scan(&req.ID, &req.OrgID, &req.OrgName, &req.IdentityID, &req.EkaID, &scopesBytes, &req.Purpose, &req.Status, &approvedAt, &req.ExpiresAt, &req.CreatedAt); err == nil {
			if approvedAt.Valid {
				req.ApprovedAt = &approvedAt.Time
			}
			_ = json.Unmarshal(scopesBytes, &req.RequestedScopes)
			list = append(list, &req)
		}
	}
	return list, nil
}

func (r *PostgresVerificationRepo) ListRequestsByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.VerificationRequest, error) {
	query := `
		SELECT vr.id, vr.org_id, o.name, vr.identity_id, i.eka_id, vr.requested_scopes, vr.purpose, vr.status, vr.approved_at, vr.expires_at, vr.created_at
		FROM verification_requests vr
		JOIN organizations o ON o.id = vr.org_id
		JOIN identities i ON i.id = vr.identity_id
		WHERE vr.org_id = $1
		ORDER BY vr.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil { return nil, err }
	defer rows.Close()

	var list []*domain.VerificationRequest
	for rows.Next() {
		var req domain.VerificationRequest
		var approvedAt sql.NullTime
		var scopesBytes []byte
		if err := rows.Scan(&req.ID, &req.OrgID, &req.OrgName, &req.IdentityID, &req.EkaID, &scopesBytes, &req.Purpose, &req.Status, &approvedAt, &req.ExpiresAt, &req.CreatedAt); err == nil {
			if approvedAt.Valid { req.ApprovedAt = &approvedAt.Time }
			_ = json.Unmarshal(scopesBytes, &req.RequestedScopes)
			list = append(list, &req)
		}
	}
	return list, nil
}

func (r *PostgresVerificationRepo) UpdateRequestStatus(ctx context.Context, id uuid.UUID, status string, approvedAt *time.Time) error {
	query := `UPDATE verification_requests SET status = $1, approved_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, status, approvedAt, id)
	return err
}

func (r *PostgresVerificationRepo) CreateResult(ctx context.Context, res *domain.VerificationResult) error {
	claimsJSON, _ := json.Marshal(res.DisclosedClaims)
	query := `
		INSERT INTO verification_results (id, request_id, disclosed_claims, status, verified_by_actor_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		res.ID, res.RequestID, claimsJSON, res.Status, res.VerifiedByActorID, res.CreatedAt,
	)
	return err
}

func (r *PostgresVerificationRepo) GetResultByRequestID(ctx context.Context, requestID uuid.UUID) (*domain.VerificationResult, error) {
	query := `SELECT id, request_id, disclosed_claims, status, verified_by_actor_id, created_at FROM verification_results WHERE request_id = $1`
	var res domain.VerificationResult
	var claimsBytes []byte
	var verBy sql.NullString
	err := r.db.QueryRowContext(ctx, query, requestID).Scan(&res.ID, &res.RequestID, &claimsBytes, &res.Status, &verBy, &res.CreatedAt)
	if err != nil { return nil, err }
	if verBy.Valid {
		p, _ := uuid.Parse(verBy.String)
		res.VerifiedByActorID = &p
	}
	_ = json.Unmarshal(claimsBytes, &res.DisclosedClaims)
	return &res, nil
}

// --- Audit Repository ---

type PostgresAuditRepo struct { db *sql.DB }

func (r *PostgresAuditRepo) Record(ctx context.Context, ev *domain.AuditEvent) error {
	metaJSON, _ := json.Marshal(ev.Metadata)
	query := `
		INSERT INTO audit_events (event_id, actor_id, actor_type, action, resource_type, resource_id, result, ip_address, user_agent, request_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.db.ExecContext(ctx, query,
		ev.EventID, ev.ActorID, ev.ActorType, ev.Action, ev.ResourceType, ev.ResourceID, ev.Result, ev.IPAddress, ev.UserAgent, ev.RequestID, metaJSON, ev.CreatedAt,
	)
	return err
}

func (r *PostgresAuditRepo) List(ctx context.Context, limit, offset int, actorID *uuid.UUID, resourceID string) ([]*domain.AuditEvent, int, error) {
	var total int
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events`).Scan(&total)

	query := `
		SELECT event_id, actor_id, actor_type, action, resource_type, resource_id, result, ip_address, user_agent, request_id, metadata, created_at
		FROM audit_events
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*domain.AuditEvent
	for rows.Next() {
		var ev domain.AuditEvent
		var actor sql.NullString
		var ip, ua, reqID sql.NullString
		var metaBytes []byte
		if err := rows.Scan(&ev.EventID, &actor, &ev.ActorType, &ev.Action, &ev.ResourceType, &ev.ResourceID, &ev.Result, &ip, &ua, &reqID, &metaBytes, &ev.CreatedAt); err == nil {
			if actor.Valid {
				parsed, _ := uuid.Parse(actor.String)
				ev.ActorID = &parsed
			}
			if ip.Valid { ev.IPAddress = ip.String }
			if ua.Valid { ev.UserAgent = ua.String }
			if reqID.Valid { ev.RequestID = reqID.String }
			if len(metaBytes) > 0 {
				_ = json.Unmarshal(metaBytes, &ev.Metadata)
			}
			events = append(events, &ev)
		}
	}
	return events, total, nil
}

// --- Credentials Repository ---

type PostgresCredentialRepo struct { db *sql.DB }

func (r *PostgresCredentialRepo) Create(ctx context.Context, c *domain.Credential) error {
	metaJSON, _ := json.Marshal(c.Metadata)
	query := `
		INSERT INTO credentials (id, identity_id, type, issuer_name, status, issued_at, expires_at, verification_method, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		c.ID, c.IdentityID, c.Type, c.IssuerName, c.Status, c.IssuedAt, c.ExpiresAt, c.VerificationMethod, metaJSON, c.CreatedAt,
	)
	return err
}

func (r *PostgresCredentialRepo) ListByIdentityID(ctx context.Context, identityID uuid.UUID) ([]*domain.Credential, error) {
	query := `SELECT id, identity_id, type, issuer_name, status, issued_at, expires_at, verification_method, metadata, created_at FROM credentials WHERE identity_id = $1 ORDER BY issued_at DESC`
	rows, err := r.db.QueryContext(ctx, query, identityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*domain.Credential
	for rows.Next() {
		var c domain.Credential
		var exp sql.NullTime
		var metaBytes []byte
		if err := rows.Scan(&c.ID, &c.IdentityID, &c.Type, &c.IssuerName, &c.Status, &c.IssuedAt, &exp, &c.VerificationMethod, &metaBytes, &c.CreatedAt); err == nil {
			if exp.Valid {
				c.ExpiresAt = &exp.Time
			}
			_ = json.Unmarshal(metaBytes, &c.Metadata)
			list = append(list, &c)
		}
	}
	return list, nil
}

func (r *PostgresCredentialRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Credential, error) {
	query := `SELECT id, identity_id, type, issuer_name, status, issued_at, expires_at, verification_method, metadata, created_at FROM credentials WHERE id = $1`
	var c domain.Credential
	var exp sql.NullTime
	var metaBytes []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(&c.ID, &c.IdentityID, &c.Type, &c.IssuerName, &c.Status, &c.IssuedAt, &exp, &c.VerificationMethod, &metaBytes, &c.CreatedAt)
	if err != nil { return nil, err }
	if exp.Valid { c.ExpiresAt = &exp.Time }
	_ = json.Unmarshal(metaBytes, &c.Metadata)
	return &c, nil
}

// --- Duplicate Flags ---

type PostgresDuplicateRepo struct { db *sql.DB }

func (r *PostgresDuplicateRepo) CreateFlag(ctx context.Context, flag *domain.DuplicateFlag) error {
	reasonsJSON, _ := json.Marshal(flag.MatchReasons)
	query := `
		INSERT INTO duplicate_flags (id, identity_id, suspected_duplicate_id, confidence_score, match_reasons, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		flag.ID, flag.IdentityID, flag.SuspectedDuplicateID, flag.ConfidenceScore, reasonsJSON, flag.Status, flag.CreatedAt,
	)
	return err
}

func (r *PostgresDuplicateRepo) ListPending(ctx context.Context) ([]*domain.DuplicateFlag, error) {
	query := `SELECT id, identity_id, suspected_duplicate_id, confidence_score, match_reasons, status, reviewed_by, reviewed_at, created_at FROM duplicate_flags WHERE status = 'PENDING_REVIEW' ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var flags []*domain.DuplicateFlag
	for rows.Next() {
		var f domain.DuplicateFlag
		var reasonsBytes []byte
		var revBy sql.NullString
		var revAt sql.NullTime
		if err := rows.Scan(&f.ID, &f.IdentityID, &f.SuspectedDuplicateID, &f.ConfidenceScore, &reasonsBytes, &f.Status, &revBy, &revAt, &f.CreatedAt); err == nil {
			_ = json.Unmarshal(reasonsBytes, &f.MatchReasons)
			if revBy.Valid {
				p, _ := uuid.Parse(revBy.String)
				f.ReviewedBy = &p
			}
			if revAt.Valid {
				f.ReviewedAt = &revAt.Time
			}
			flags = append(flags, &f)
		}
	}
	return flags, nil
}

func (r *PostgresDuplicateRepo) ResolveFlag(ctx context.Context, id uuid.UUID, status string, reviewerID uuid.UUID) error {
	now := time.Now().UTC()
	query := `UPDATE duplicate_flags SET status = $1, reviewed_by = $2, reviewed_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, status, reviewerID, now, id)
	return err
}