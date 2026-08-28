export const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export interface User {
  id: string;
  email: string;
  phone?: string;
  role: string;
  status: string;
  created_at: string;
}

export interface Identity {
  id: string;
  eka_id: string;
  user_id: string;
  status: string;
  verification_level: string;
  verified_at?: string;
  created_at: string;
}

export interface Profile {
  identity_id: string;
  legal_name: string;
  date_of_birth: string;
  gender?: string;
  profile_photo_url?: string;
  phone?: string;
  email?: string;
  address_line1?: string;
  city?: string;
  state?: string;
  postal_code?: string;
  country?: string;
}

export interface Credential {
  id: string;
  identity_id: string;
  type: string;
  issuer_name: string;
  status: string;
  issued_at: string;
  expires_at?: string;
  verification_method: string;
  metadata?: Record<string, any>;
}

export interface VerificationRequest {
  id: string;
  org_id: string;
  org_name?: string;
  identity_id: string;
  eka_id?: string;
  requested_scopes: string[];
  purpose: string;
  status: string;
  approved_at?: string;
  expires_at: string;
  created_at: string;
}

export interface QRVerificationResult {
  status: string;
  eka_id: string;
  verification_level: string;
  verified_at?: string;
  legal_name?: string;
  disclosed_claims: Record<string, any>;
  verification_date: string;
}

export interface AuditEvent {
  event_id: string;
  actor_id?: string;
  actor_type: string;
  action: string;
  resource_type: string;
  resource_id: string;
  result: string;
  ip_address?: string;
  user_agent?: string;
  request_id?: string;
  metadata?: Record<string, any>;
  created_at: string;
}

export async function requestOTP(target: string) {
  const res = await fetch(`${API_BASE}/api/v1/auth/request-otp`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ target }),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error?.message || 'Failed to request OTP');
  }
  return res.json();
}

export async function registerUser(payload: any) {
  const res = await fetch(`${API_BASE}/api/v1/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error?.message || 'Registration failed');
  }
  return res.json();
}

export async function loginUser(email: string, password: string) {
  const res = await fetch(`${API_BASE}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error?.message || 'Login failed');
  }
  return res.json();
}

export async function getMyIdentity(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/identities/me`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error?.message || 'Failed to fetch identity');
  }
  return res.json();
}

export async function generateQRToken(token: string, scopes: string[], durationMinutes: number) {
  const res = await fetch(`${API_BASE}/api/v1/qr/generate`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ scopes, duration_minutes: durationMinutes }),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error?.message || 'Failed to generate QR');
  }
  return res.json();
}

export async function verifyQRToken(qrToken: string) {
  const res = await fetch(`${API_BASE}/api/v1/qr/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ token: qrToken }),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error?.message || 'Verification failed');
  }
  return res.json();
}

export async function getPendingVerificationRequests(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/verification-requests/pending`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error('Failed to fetch pending requests');
  return res.json();
}

export async function respondVerificationRequest(token: string, requestId: string, approved: boolean) {
  const res = await fetch(`${API_BASE}/api/v1/verification-requests/${requestId}/respond`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ approved }),
  });
  if (!res.ok) throw new Error('Failed to process consent response');
  return res.json();
}

export async function createVerificationRequest(token: string, payload: any) {
  const res = await fetch(`${API_BASE}/api/v1/verification-requests`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    const err = await res.json();
    throw new Error(err.error?.message || 'Failed to submit verification request');
  }
  return res.json();
}

export async function adminListIdentities(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/admin/identities?limit=50`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error('Failed to load identities');
  return res.json();
}

export async function adminUpdateStatus(token: string, identityId: string, status: string) {
  const res = await fetch(`${API_BASE}/api/v1/admin/identities/${identityId}/status`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ status }),
  });
  if (!res.ok) throw new Error('Failed to update status');
  return res.json();
}

export async function adminListDuplicates(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/admin/duplicates`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error('Failed to load duplicate queue');
  return res.json();
}

export async function adminResolveDuplicate(token: string, flagId: string, status: string) {
  const res = await fetch(`${API_BASE}/api/v1/admin/duplicates/${flagId}/resolve`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ status }),
  });
  if (!res.ok) throw new Error('Failed to resolve duplicate flag');
  return res.json();
}

export async function adminListAudit(token: string) {
  const res = await fetch(`${API_BASE}/api/v1/admin/audit?limit=50`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) throw new Error('Failed to load audit events');
  return res.json();
}