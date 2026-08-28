package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sauryah/eka-id/services/api/internal/domain"
)

type MemoryStore struct {
	mu            *sync.RWMutex
	filePath      string
	Users         *MemoryUserRepo
	Identities    *MemoryIdentityRepo
	Profiles      *MemoryProfileRepo
	Organizations *MemoryOrganizationRepo
	Verification  *MemoryVerificationRepo
	QR            *MemoryQRRepo
	Credentials   *MemoryCredentialRepo
	Audit         *MemoryAuditRepo
	Duplicates    *MemoryDuplicateRepo
}

func NewMemoryStore() *MemoryStore {
	sharedMu := &sync.RWMutex{}
	usersMap := make(map[uuid.UUID]*domain.User)
	usersByEmail := make(map[string]*domain.User)
	identitiesMap := make(map[uuid.UUID]*domain.Identity)
	identitiesByEkaID := make(map[string]*domain.Identity)
	identitiesByUserID := make(map[uuid.UUID]*domain.Identity)
	profilesMap := make(map[uuid.UUID]*domain.Profile)
	orgsMap := make(map[uuid.UUID]*domain.Organization)
	verifReqs := make(map[uuid.UUID]*domain.VerificationRequest)
	verifResults := make(map[uuid.UUID]*domain.VerificationResult)
	qrTokens := make(map[string]*domain.QRToken)
	credsMap := make(map[uuid.UUID][]*domain.Credential)
	auditList := make([]*domain.AuditEvent, 0)
	dupFlags := make(map[uuid.UUID]*domain.DuplicateFlag)

	return &MemoryStore{
		mu: sharedMu,
		Users: &MemoryUserRepo{
			mu:           sharedMu,
			users:        usersMap,
			usersByEmail: usersByEmail,
		},
		Identities: &MemoryIdentityRepo{
			mu:                 sharedMu,
			identities:         identitiesMap,
			identitiesByEkaID:  identitiesByEkaID,
			identitiesByUserID: identitiesByUserID,
		},
		Profiles: &MemoryProfileRepo{
			mu:       sharedMu,
			profiles: profilesMap,
		},
		Organizations: &MemoryOrganizationRepo{
			mu:   sharedMu,
			orgs: orgsMap,
		},
		Verification: &MemoryVerificationRepo{
			mu:      sharedMu,
			reqs:    verifReqs,
			results: verifResults,
		},
		QR: &MemoryQRRepo{
			mu:     sharedMu,
			tokens: qrTokens,
		},
		Credentials: &MemoryCredentialRepo{
			mu:          sharedMu,
			credentials: credsMap,
		},
		Audit: &MemoryAuditRepo{
			mu:     sharedMu,
			events: &auditList,
		},
		Duplicates: &MemoryDuplicateRepo{
			mu:    sharedMu,
			flags: dupFlags,
		},
	}
}

// User Repo
type MemoryUserRepo struct {
	mu           *sync.RWMutex
	users        map[uuid.UUID]*domain.User
	usersByEmail map[string]*domain.User
	saveHook     func()
}

func (r *MemoryUserRepo) Create(ctx context.Context, u *domain.User) error {
	r.mu.Lock()
	if _, exists := r.usersByEmail[strings.ToLower(u.Email)]; exists {
		r.mu.Unlock()
		return errors.New("user already exists with email")
	}
	r.users[u.ID] = u
	r.usersByEmail[strings.ToLower(u.Email)] = u
	r.mu.Unlock()
	if r.saveHook != nil {
		go r.saveHook()
	}
	return nil
}

func (r *MemoryUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if u, exists := r.users[id]; exists {
		return u, nil
	}
	return nil, ErrNotFound
}

func (r *MemoryUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if u, exists := r.usersByEmail[strings.ToLower(email)]; exists {
		return u, nil
	}
	return nil, ErrNotFound
}

func (r *MemoryUserRepo) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	r.users[user.ID] = user
	r.usersByEmail[strings.ToLower(user.Email)] = user
	r.mu.Unlock()
	if r.saveHook != nil {
		go r.saveHook()
	}
	return nil
}

// Identity Repo
type MemoryIdentityRepo struct {
	mu                 *sync.RWMutex
	identities         map[uuid.UUID]*domain.Identity
	identitiesByEkaID  map[string]*domain.Identity
	identitiesByUserID map[uuid.UUID]*domain.Identity
	saveHook           func()
}

func (r *MemoryIdentityRepo) Create(ctx context.Context, ident *domain.Identity) error {
	r.mu.Lock()
	if _, exists := r.identitiesByEkaID[ident.EkaID]; exists {
		r.mu.Unlock()
		return errors.New("eka id already exists")
	}
	r.identities[ident.ID] = ident
	r.identitiesByEkaID[ident.EkaID] = ident
	r.identitiesByUserID[ident.UserID] = ident
	r.mu.Unlock()
	if r.saveHook != nil {
		go r.saveHook()
	}
	return nil
}

func (r *MemoryIdentityRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Identity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ident, exists := r.identities[id]; exists {
		return ident, nil
	}
	return nil, ErrNotFound
}

func (r *MemoryIdentityRepo) GetByEkaID(ctx context.Context, ekaID string) (*domain.Identity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ident, exists := r.identitiesByEkaID[ekaID]; exists {
		return ident, nil
	}
	return nil, ErrNotFound
}

func (r *MemoryIdentityRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Identity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if ident, exists := r.identitiesByUserID[userID]; exists {
		return ident, nil
	}
	return nil, ErrNotFound
}

func (r *MemoryIdentityRepo) List(ctx context.Context, limit, offset int) ([]*domain.Identity, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var all []*domain.Identity
	for _, ident := range r.identities {
		all = append(all, ident)
	}
	total := len(all)
	if offset >= total {
		return []*domain.Identity{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (r *MemoryIdentityRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	r.mu.Lock()
	if ident, exists := r.identities[id]; exists {
		ident.Status = status
		ident.UpdatedAt = time.Now().UTC()
		r.mu.Unlock()
		if r.saveHook != nil {
			go r.saveHook()
		}
		return nil
	}
	r.mu.Unlock()
	return ErrNotFound
}

func (r *MemoryIdentityRepo) UpdateVerificationLevel(ctx context.Context, id uuid.UUID, level string, verifiedAt *time.Time) error {
	r.mu.Lock()
	if ident, exists := r.identities[id]; exists {
		ident.VerificationLevel = level
		ident.VerifiedAt = verifiedAt
		ident.UpdatedAt = time.Now().UTC()
		r.mu.Unlock()
		if r.saveHook != nil {
			go r.saveHook()
		}
		return nil
	}
	r.mu.Unlock()
	return ErrNotFound
}

// Profile Repo
type MemoryProfileRepo struct {
	mu       *sync.RWMutex
	profiles map[uuid.UUID]*domain.Profile
	saveHook func()
}

func (r *MemoryProfileRepo) Create(ctx context.Context, p *domain.Profile) error {
	r.mu.Lock()
	r.profiles[p.IdentityID] = p
	r.mu.Unlock()
	if r.saveHook != nil {
		go r.saveHook()
	}
	return nil
}

func (r *MemoryProfileRepo) GetByIdentityID(ctx context.Context, id uuid.UUID) (*domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if p, exists := r.profiles[id]; exists {
		return p, nil
	}
	return nil, ErrNotFound
}

func (r *MemoryProfileRepo) FindByPhone(ctx context.Context, phone string) ([]*domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []*domain.Profile
	for _, p := range r.profiles {
		if p.Phone == phone {
			res = append(res, p)
		}
	}
	return res, nil
}

func (r *MemoryProfileRepo) FindByEmail(ctx context.Context, email string) ([]*domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []*domain.Profile
	for _, p := range r.profiles {
		if strings.EqualFold(p.Email, email) {
			res = append(res, p)
		}
	}
	return res, nil
}

func (r *MemoryProfileRepo) FindPotentialDuplicates(ctx context.Context, legalName, dob, phone, email string) ([]*domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var matches []*domain.Profile
	for _, p := range r.profiles {
		hasBio := p.ProfilePhotoURL != "" || (p.Metadata != nil && p.Metadata["face_embedding"] != nil)
		if (phone != "" && p.Phone == phone) ||
			(email != "" && strings.EqualFold(p.Email, email)) ||
			(strings.EqualFold(p.LegalName, legalName) && p.DateOfBirth == dob) ||
			hasBio {
			matches = append(matches, p)
		}
	}
	return matches, nil
}

func (r *MemoryProfileRepo) Update(ctx context.Context, p *domain.Profile) error {
	r.mu.Lock()
	r.profiles[p.IdentityID] = p
	r.mu.Unlock()
	if r.saveHook != nil {
		go r.saveHook()
	}
	return nil
}

// Organization Repo
type MemoryOrganizationRepo struct {
	mu   *sync.RWMutex
	orgs map[uuid.UUID]*domain.Organization
}

func (r *MemoryOrganizationRepo) Create(ctx context.Context, org *domain.Organization) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orgs[org.ID] = org
	return nil
}

func (r *MemoryOrganizationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if o, exists := r.orgs[id]; exists {
		return o, nil
	}
	return nil, ErrNotFound
}

func (r *MemoryOrganizationRepo) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, o := range r.orgs {
		if o.Slug == slug {
			return o, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryOrganizationRepo) List(ctx context.Context) ([]*domain.Organization, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*domain.Organization
	for _, o := range r.orgs {
		list = append(list, o)
	}
	return list, nil
}

// QR Repo
type MemoryQRRepo struct {
	mu     *sync.RWMutex
	tokens map[string]*domain.QRToken
}

func (r *MemoryQRRepo) CreateToken(ctx context.Context, qr *domain.QRToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[qr.Token] = qr
	return nil
}

func (r *MemoryQRRepo) GetToken(ctx context.Context, token string) (*domain.QRToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if qr, exists := r.tokens[token]; exists {
		return qr, nil
	}
	return nil, ErrNotFound
}

func (r *MemoryQRRepo) MarkTokenUsed(ctx context.Context, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if qr, exists := r.tokens[token]; exists {
		qr.IsUsed = true
		return nil
	}
	return ErrNotFound
}

// Verification Repo
type MemoryVerificationRepo struct {
	mu      *sync.RWMutex
	reqs    map[uuid.UUID]*domain.VerificationRequest
	results map[uuid.UUID]*domain.VerificationResult
}

func (r *MemoryVerificationRepo) CreateRequest(ctx context.Context, req *domain.VerificationRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reqs[req.ID] = req
	return nil
}

func (r *MemoryVerificationRepo) GetRequestByID(ctx context.Context, id uuid.UUID) (*domain.VerificationRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if req, exists := r.reqs[id]; exists {
		return req, nil
	}
	return nil, ErrNotFound
}

func (r *MemoryVerificationRepo) ListRequestsByIdentity(ctx context.Context, identityID uuid.UUID) ([]*domain.VerificationRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*domain.VerificationRequest
	for _, req := range r.reqs {
		if req.IdentityID == identityID {
			list = append(list, req)
		}
	}
	return list, nil
}

func (r *MemoryVerificationRepo) ListRequestsByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.VerificationRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*domain.VerificationRequest
	for _, req := range r.reqs {
		if req.OrgID == orgID {
			list = append(list, req)
		}
	}
	return list, nil
}

func (r *MemoryVerificationRepo) UpdateRequestStatus(ctx context.Context, id uuid.UUID, status string, approvedAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if req, exists := r.reqs[id]; exists {
		req.Status = status
		req.ApprovedAt = approvedAt
		return nil
	}
	return ErrNotFound
}

func (r *MemoryVerificationRepo) CreateResult(ctx context.Context, res *domain.VerificationResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results[res.ID] = res
	return nil
}

func (r *MemoryVerificationRepo) GetResultByRequestID(ctx context.Context, requestID uuid.UUID) (*domain.VerificationResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, res := range r.results {
		if res.RequestID == requestID {
			return res, nil
		}
	}
	return nil, ErrNotFound
}

// Credential Repo
type MemoryCredentialRepo struct {
	mu          *sync.RWMutex
	credentials map[uuid.UUID][]*domain.Credential
	saveHook    func()
}

func (r *MemoryCredentialRepo) Create(ctx context.Context, c *domain.Credential) error {
	r.mu.Lock()
	r.credentials[c.IdentityID] = append(r.credentials[c.IdentityID], c)
	r.mu.Unlock()
	if r.saveHook != nil {
		go r.saveHook()
	}
	return nil
}

func (r *MemoryCredentialRepo) ListByIdentityID(ctx context.Context, identityID uuid.UUID) ([]*domain.Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.credentials[identityID], nil
}

func (r *MemoryCredentialRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Credential, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, creds := range r.credentials {
		for _, c := range creds {
			if c.ID == id {
				return c, nil
			}
		}
	}
	return nil, ErrNotFound
}

// Audit Repo
type MemoryAuditRepo struct {
	mu       *sync.RWMutex
	events   *[]*domain.AuditEvent
	saveHook func()
}

func (r *MemoryAuditRepo) Record(ctx context.Context, ev *domain.AuditEvent) error {
	r.mu.Lock()
	*r.events = append([]*domain.AuditEvent{ev}, *r.events...)
	r.mu.Unlock()
	if r.saveHook != nil {
		go r.saveHook()
	}
	return nil
}

func (r *MemoryAuditRepo) List(ctx context.Context, limit, offset int, actorID *uuid.UUID, resourceID string) ([]*domain.AuditEvent, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var filtered []*domain.AuditEvent
	for _, ev := range *r.events {
		if actorID != nil && (ev.ActorID == nil || *ev.ActorID != *actorID) {
			continue
		}
		if resourceID != "" && ev.ResourceID != resourceID {
			continue
		}
		filtered = append(filtered, ev)
	}
	total := len(filtered)
	if offset >= total {
		return []*domain.AuditEvent{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return filtered[offset:end], total, nil
}

// Duplicate Repo
type MemoryDuplicateRepo struct {
	mu       *sync.RWMutex
	flags    map[uuid.UUID]*domain.DuplicateFlag
	saveHook func()
}

func (r *MemoryDuplicateRepo) CreateFlag(ctx context.Context, flag *domain.DuplicateFlag) error {
	r.mu.Lock()
	r.flags[flag.ID] = flag
	r.mu.Unlock()
	if r.saveHook != nil {
		go r.saveHook()
	}
	return nil
}

func (r *MemoryDuplicateRepo) ListPending(ctx context.Context) ([]*domain.DuplicateFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []*domain.DuplicateFlag
	for _, f := range r.flags {
		if f.Status == "PENDING_REVIEW" {
			list = append(list, f)
		}
	}
	return list, nil
}

func (r *MemoryDuplicateRepo) ResolveFlag(ctx context.Context, id uuid.UUID, status string, reviewerID uuid.UUID) error {
	r.mu.Lock()
	if f, exists := r.flags[id]; exists {
		f.Status = status
		f.ReviewedBy = &reviewerID
		now := time.Now().UTC()
		f.ReviewedAt = &now
		r.mu.Unlock()
		if r.saveHook != nil {
			go r.saveHook()
		}
		return nil
	}
	r.mu.Unlock()
	return ErrNotFound
}

// -------------------------------------------------------------
// Disk Persistence Support for Standalone Offline Operation
// -------------------------------------------------------------

type PersistedState struct {
	Users          []*domain.User          `json:"users"`
	Identities     []*domain.Identity      `json:"identities"`
	Profiles       []*domain.Profile       `json:"profiles"`
	Organizations  []*domain.Organization  `json:"organizations"`
	Credentials    []*domain.Credential    `json:"credentials"`
	DuplicateFlags []*domain.DuplicateFlag `json:"duplicate_flags"`
	AuditEvents    []*domain.AuditEvent    `json:"audit_events"`
}

func (m *MemoryStore) SetPersistenceFile(path string) {
	m.filePath = path
	hook := func() {
		_ = m.SaveToFile()
	}
	m.Users.saveHook = hook
	m.Identities.saveHook = hook
	m.Profiles.saveHook = hook
	m.Credentials.saveHook = hook
	m.Duplicates.saveHook = hook
	m.Audit.saveHook = hook
}

func (m *MemoryStore) SaveToFile() error {
	if m.filePath == "" {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	state := PersistedState{
		Users:          make([]*domain.User, 0, len(m.Users.users)),
		Identities:     make([]*domain.Identity, 0, len(m.Identities.identities)),
		Profiles:       make([]*domain.Profile, 0, len(m.Profiles.profiles)),
		Organizations:  make([]*domain.Organization, 0, len(m.Organizations.orgs)),
		Credentials:    make([]*domain.Credential, 0),
		DuplicateFlags: make([]*domain.DuplicateFlag, 0, len(m.Duplicates.flags)),
		AuditEvents:    make([]*domain.AuditEvent, 0, len(*m.Audit.events)),
	}

	for _, u := range m.Users.users {
		state.Users = append(state.Users, u)
	}
	for _, i := range m.Identities.identities {
		state.Identities = append(state.Identities, i)
	}
	for _, p := range m.Profiles.profiles {
		state.Profiles = append(state.Profiles, p)
	}
	for _, o := range m.Organizations.orgs {
		state.Organizations = append(state.Organizations, o)
	}
	for _, credList := range m.Credentials.credentials {
		state.Credentials = append(state.Credentials, credList...)
	}
	for _, f := range m.Duplicates.flags {
		state.DuplicateFlags = append(state.DuplicateFlags, f)
	}
	state.AuditEvents = append(state.AuditEvents, (*m.Audit.events)...)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpFile := fmt.Sprintf("%s.tmp.%d", m.filePath, time.Now().UnixNano())
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, m.filePath)
}

func (m *MemoryStore) LoadFromFile(path string) (bool, error) {
	m.filePath = path
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	var state PersistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return false, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, u := range state.Users {
		m.Users.users[u.ID] = u
		m.Users.usersByEmail[strings.ToLower(u.Email)] = u
	}
	for _, i := range state.Identities {
		m.Identities.identities[i.ID] = i
		m.Identities.identitiesByEkaID[i.EkaID] = i
		m.Identities.identitiesByUserID[i.UserID] = i
	}
	for _, p := range state.Profiles {
		m.Profiles.profiles[p.IdentityID] = p
	}
	for _, o := range state.Organizations {
		m.Organizations.orgs[o.ID] = o
	}
	for _, c := range state.Credentials {
		m.Credentials.credentials[c.IdentityID] = append(m.Credentials.credentials[c.IdentityID], c)
	}
	for _, f := range state.DuplicateFlags {
		m.Duplicates.flags[f.ID] = f
	}
	*m.Audit.events = append(*m.Audit.events, state.AuditEvents...)

	return true, nil
}