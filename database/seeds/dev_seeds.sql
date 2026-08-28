-- Development Seed Data for EKA ID
-- Clearly marked for local development and testing only

-- Clean existing data
TRUNCATE users, identities, profiles, organizations, organization_members, verification_requests, verification_results, qr_tokens, credentials, audit_events, duplicate_flags CASCADE;

-- 1. Insert System Admin User (Password: Password123!)
-- Hash generated with standard bcrypt cost 10
INSERT INTO users (id, email, phone, password_hash, role, status)
VALUES (
    'a0000000-0000-0000-0000-000000000001',
    'admin@eka.dev',
    '+919876543210',
    '$2a$10$y5X4XkF8qE6n0V5Q8i2I2.hA21f8aZqE7e0zG/r6I6.1WqfH6WJ.K',
    'SYSTEM_ADMIN',
    'ACTIVE'
);

-- 2. Insert Standard User John Mathew
INSERT INTO users (id, email, phone, password_hash, role, status)
VALUES (
    'b0000000-0000-0000-0000-000000000002',
    'john.mathew@example.com',
    '+919876500001',
    '$2a$10$y5X4XkF8qE6n0V5Q8i2I2.hA21f8aZqE7e0zG/r6I6.1WqfH6WJ.K',
    'USER',
    'ACTIVE'
);

-- 3. Insert Identity for John Mathew with Public EKA ID
INSERT INTO identities (id, eka_id, user_id, status, verification_level, verified_at)
VALUES (
    'c0000000-0000-0000-0000-000000000003',
    'EKA-7K4M-92PX',
    'b0000000-0000-0000-0000-000000000002',
    'ACTIVE',
    'TIER_1_BASIC',
    NOW() - INTERVAL '10 days'
);

-- 4. Insert Profile for John Mathew
INSERT INTO profiles (identity_id, legal_name, date_of_birth, gender, profile_photo_url, phone, email, address_line1, city, state, postal_code, country)
VALUES (
    'c0000000-0000-0000-0000-000000000003',
    'John Mathew',
    '1992-05-14',
    'MALE',
    'https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=300&h=300&fit=crop&crop=faces',
    '+919876500001',
    'john.mathew@example.com',
    '42 Innovation Park, Koramangala',
    'Bengaluru',
    'Karnataka',
    '560034',
    'India'
);

-- 5. Insert Sample Organization
INSERT INTO organizations (id, name, slug, api_key_hash, status, webhook_url)
VALUES (
    'd0000000-0000-0000-0000-000000000004',
    'Acme Technologies Ltd.',
    'acme-tech',
    '$2a$10$y5X4XkF8qE6n0V5Q8i2I2.hA21f8aZqE7e0zG/r6I6.1WqfH6WJ.K',
    'ACTIVE',
    'https://api.acme.example.com/eka-webhook'
);

-- 6. Insert Org Admin User
INSERT INTO users (id, email, phone, password_hash, role, status)
VALUES (
    'e0000000-0000-0000-0000-000000000005',
    'sarah.recruiter@acme.example.com',
    '+919876599999',
    '$2a$10$y5X4XkF8qE6n0V5Q8i2I2.hA21f8aZqE7e0zG/r6I6.1WqfH6WJ.K',
    'ORG_ADMIN',
    'ACTIVE'
);

INSERT INTO organization_members (id, org_id, user_id, role)
VALUES (
    'f0000000-0000-0000-0000-000000000006',
    'd0000000-0000-0000-0000-000000000004',
    'e0000000-0000-0000-0000-000000000005',
    'ADMIN'
);

-- 7. Insert Sample Credential
INSERT INTO credentials (id, identity_id, type, issuer_name, status, issued_at, expires_at, verification_method, metadata)
VALUES (
    'a1111111-1111-1111-1111-111111111111',
    'c0000000-0000-0000-0000-000000000003',
    'EMPLOYMENT',
    'Acme Technologies Ltd.',
    'ACTIVE',
    NOW() - INTERVAL '1 year',
    NOW() + INTERVAL '2 years',
    'CORPORATE_DIGITAL_SIGNATURE',
    '{"title": "Senior Systems Architect", "department": "Platform Engineering"}'::jsonb
);

-- 8. Insert Sample Pending Verification Request
INSERT INTO verification_requests (id, org_id, identity_id, requested_scopes, purpose, status, expires_at)
VALUES (
    'b2222222-2222-2222-2222-222222222222',
    'd0000000-0000-0000-0000-000000000004',
    'c0000000-0000-0000-0000-000000000003',
    '["identity_valid", "name_match", "phone"]'::jsonb,
    'Background verification for senior technical role onboarding',
    'PENDING',
    NOW() + INTERVAL '7 days'
);

-- 9. Insert Sample Audit Events
INSERT INTO audit_events (event_id, actor_id, actor_type, action, resource_type, resource_id, result, ip_address, request_id, metadata)
VALUES
(
    uuid_generate_v4(),
    'b0000000-0000-0000-0000-000000000002',
    'USER',
    'IDENTITY_CREATED',
    'IDENTITY',
    'c0000000-0000-0000-0000-000000000003',
    'SUCCESS',
    '127.0.0.1',
    'req-init-seed-01',
    '{"eka_id": "EKA-7K4M-92PX"}'::jsonb
),
(
    uuid_generate_v4(),
    'b0000000-0000-0000-0000-000000000002',
    'USER',
    'IDENTITY_VERIFIED',
    'IDENTITY',
    'c0000000-0000-0000-0000-000000000003',
    'SUCCESS',
    '127.0.0.1',
    'req-init-seed-02',
    '{"level": "TIER_1_BASIC"}'::jsonb
),
(
    uuid_generate_v4(),
    'e0000000-0000-0000-0000-000000000005',
    'ORG',
    'VERIFICATION_REQUESTED',
    'VERIFICATION_REQUEST',
    'b2222222-2222-2222-2222-222222222222',
    'SUCCESS',
    '198.51.100.4',
    'req-init-seed-03',
    '{"org": "Acme Technologies Ltd."}'::jsonb
);
