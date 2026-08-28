'use client';

import React, { useEffect, useState } from 'react';
import {
  Shield, QrCode, CheckCircle2, Award, Clock, Eye, Sliders,
  Check, X, AlertCircle, Lock, Download, Printer, UserCheck, RefreshCw, Key
} from 'lucide-react';
import DigitalEkaCard from '@/components/DigitalEkaCard';
import {
  getMyIdentity, generateQRToken, getPendingVerificationRequests,
  respondVerificationRequest, Identity, Profile, Credential, VerificationRequest
} from '@/lib/api';

export default function DashboardPage() {
  const [activeTab, setActiveTab] = useState<'identity' | 'card' | 'qr' | 'requests' | 'credentials' | 'activity' | 'privacy'>('identity');
  const [loading, setLoading] = useState(true);
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [requests, setRequests] = useState<VerificationRequest[]>([]);
  const [error, setError] = useState<string | null>(null);

  // Dynamic QR Studio State
  const [qrDuration, setQrDuration] = useState<number>(15);
  const [qrScopes, setQrScopes] = useState<string[]>(['identity_valid', 'legal_name']);
  const [generatedQR, setGeneratedQR] = useState<{ token: string; verify_url: string; expires_at: string } | null>(null);
  const [qrGenerating, setQrGenerating] = useState(false);

  // Request response feedback
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);

  useEffect(() => {
    loadDashboardData();
  }, []);

  const loadDashboardData = async () => {
    const token = localStorage.getItem('eka_token');
    if (!token) {
      window.location.href = '/login';
      return;
    }

    try {
      setLoading(true);
      setError(null);
      const data = await getMyIdentity(token);
      setIdentity(data.identity);
      setProfile(data.profile);
      setCredentials(data.credentials || []);

      // Load pending verification requests
      try {
        const reqs = await getPendingVerificationRequests(token);
        setRequests(reqs || []);
      } catch (e) {
        console.warn('Could not load pending requests:', e);
      }

      // Initial QR generation
      try {
        const qr = await generateQRToken(token, ['identity_valid', 'legal_name'], 15);
        setGeneratedQR(qr);
      } catch (e) {
        console.warn('QR auto-generation skipped:', e);
      }
    } catch (err: any) {
      setError(err.message || 'Failed to load identity dashboard.');
    } finally {
      setLoading(false);
    }
  };

  const handleGenerateCustomQR = async () => {
    const token = localStorage.getItem('eka_token');
    if (!token) return;
    setQrGenerating(true);
    try {
      const qr = await generateQRToken(token, qrScopes, qrDuration);
      setGeneratedQR(qr);
      setActionSuccess('Fresh verification QR code generated!');
      setTimeout(() => setActionSuccess(null), 4000);
    } catch (err: any) {
      setError(err.message || 'Failed to generate QR');
    } finally {
      setQrGenerating(false);
    }
  };

  const handleConsent = async (requestId: string, approved: boolean) => {
    const token = localStorage.getItem('eka_token');
    if (!token) return;

    try {
      await respondVerificationRequest(token, requestId, approved);
      setActionSuccess(approved ? 'Verification request approved with selective claims.' : 'Verification request denied.');
      // Refresh requests list
      const updated = await getPendingVerificationRequests(token);
      setRequests(updated || []);
      setTimeout(() => setActionSuccess(null), 4000);
    } catch (err: any) {
      setError(err.message || 'Failed to process request');
    }
  };

  const toggleScope = (scope: string) => {
    if (qrScopes.includes(scope)) {
      setQrScopes(qrScopes.filter((s) => s !== scope));
    } else {
      setQrScopes([...qrScopes, scope]);
    }
  };

  if (loading) {
    return (
      <div className="min-h-[70vh] flex items-center justify-center">
        <div className="text-center space-y-3">
          <RefreshCw className="w-8 h-8 text-teal-700 animate-spin mx-auto" />
          <p className="text-sm font-medium text-slate-600">Loading secure identity credentials...</p>
        </div>
      </div>
    );
  }

  if (error && !identity) {
    return (
      <div className="max-w-2xl mx-auto my-12 p-6 bg-white rounded-2xl border border-slate-200 shadow-sm text-center">
        <AlertCircle className="w-10 h-10 text-rose-600 mx-auto mb-3" />
        <h3 className="text-lg font-bold text-slate-900">Session Error</h3>
        <p className="text-sm text-slate-600 mt-1 mb-4">{error}</p>
        <button
          onClick={() => {
            localStorage.removeItem('eka_token');
            window.location.href = '/login';
          }}
          className="px-4 py-2 bg-teal-700 text-white text-sm font-semibold rounded-lg shadow"
        >
          Return to Sign In
        </button>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      {/* Top Banner Overview */}
      <div className="bg-gradient-to-r from-slate-900 via-slate-800 to-teal-900 rounded-2xl p-6 sm:p-8 text-white shadow-lg mb-8">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-6">
          <div className="flex items-center space-x-4">
            <div className="w-16 h-16 rounded-full p-0.5 bg-teal-400">
              {profile?.profile_photo_url ? (
                <img
                  src={profile.profile_photo_url}
                  alt={profile.legal_name}
                  className="w-full h-full rounded-full object-cover"
                />
              ) : (
                <div className="w-full h-full rounded-full bg-slate-800 flex items-center justify-center font-bold text-teal-300 text-xl">
                  {profile?.legal_name?.charAt(0) || 'U'}
                </div>
              )}
            </div>

            <div>
              <div className="flex items-center space-x-2">
                <h1 className="text-2xl font-extrabold tracking-tight">{profile?.legal_name}</h1>
                <span className="flex items-center space-x-1 px-2 py-0.5 bg-emerald-500/20 border border-emerald-400/40 text-emerald-300 rounded-full text-xs font-semibold">
                  <CheckCircle2 className="w-3 h-3" />
                  <span>{identity?.status}</span>
                </span>
              </div>
              <p className="text-slate-300 text-xs mt-0.5">{profile?.email} • {profile?.phone}</p>
            </div>
          </div>

          <div className="flex items-center space-x-4 bg-slate-950/60 border border-slate-700/80 p-3.5 rounded-xl">
            <div>
              <p className="text-[10px] text-teal-400 uppercase tracking-widest font-semibold">Public EKA ID</p>
              <p className="font-mono text-lg sm:text-xl font-bold tracking-wider text-white">
                {identity?.eka_id}
              </p>
            </div>
            <div className="border-l border-slate-700 pl-4">
              <p className="text-[10px] text-slate-400 uppercase tracking-widest font-semibold">Verification</p>
              <p className="text-xs font-bold text-emerald-400 flex items-center space-x-1 mt-0.5">
                <CheckCircle2 className="w-3.5 h-3.5" />
                <span>{identity?.verification_level?.replace(/_/g, ' ')}</span>
              </p>
            </div>
          </div>
        </div>
      </div>

      {actionSuccess && (
        <div className="mb-6 p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-sm flex items-center space-x-2">
          <CheckCircle2 className="w-5 h-5 flex-shrink-0" />
          <span>{actionSuccess}</span>
        </div>
      )}

      {/* Navigation Tabs */}
      <div className="flex overflow-x-auto space-x-2 border-b border-slate-200 pb-2 mb-8">
        {[
          { id: 'identity', label: 'My Identity', icon: Shield },
          { id: 'card', label: 'Digital Card', icon: Award },
          { id: 'qr', label: 'QR Studio', icon: QrCode },
          { id: 'requests', label: `Consent Requests (${requests.length})`, icon: UserCheck },
          { id: 'credentials', label: `Credentials (${credentials.length})`, icon: Key },
          { id: 'privacy', label: 'Privacy & Rights', icon: Eye },
        ].map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`flex items-center space-x-2 px-4 py-2 rounded-lg text-xs sm:text-sm font-semibold whitespace-nowrap transition ${
                isActive
                  ? 'bg-teal-700 text-white shadow-sm'
                  : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
              }`}
            >
              <Icon className="w-4 h-4" />
              <span>{tab.label}</span>
            </button>
          );
        })}
      </div>

      {/* TAB CONTENT 1: My Identity Overview */}
      {activeTab === 'identity' && identity && profile && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2 space-y-6">
            <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-5">
              <h3 className="font-bold text-slate-900 text-base flex items-center space-x-2">
                <Shield className="w-5 h-5 text-teal-700" />
                <span>Identity Attributes</span>
              </h3>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
                <div className="p-3.5 rounded-xl bg-slate-50 border border-slate-100">
                  <span className="text-slate-400 uppercase tracking-wider font-semibold text-[10px]">Legal Name</span>
                  <p className="text-sm font-bold text-slate-900 mt-1">{profile.legal_name}</p>
                </div>
                <div className="p-3.5 rounded-xl bg-slate-50 border border-slate-100">
                  <span className="text-slate-400 uppercase tracking-wider font-semibold text-[10px]">Date of Birth</span>
                  <p className="text-sm font-bold text-slate-900 mt-1">{profile.date_of_birth}</p>
                </div>
                <div className="p-3.5 rounded-xl bg-slate-50 border border-slate-100">
                  <span className="text-slate-400 uppercase tracking-wider font-semibold text-[10px]">Primary Email</span>
                  <p className="text-sm font-bold text-slate-900 mt-1">{profile.email}</p>
                </div>
                <div className="p-3.5 rounded-xl bg-slate-50 border border-slate-100">
                  <span className="text-slate-400 uppercase tracking-wider font-semibold text-[10px]">Registered Phone</span>
                  <p className="text-sm font-bold text-slate-900 mt-1">{profile.phone}</p>
                </div>
                <div className="sm:col-span-2 p-3.5 rounded-xl bg-slate-50 border border-slate-100">
                  <span className="text-slate-400 uppercase tracking-wider font-semibold text-[10px]">Residential Address</span>
                  <p className="text-sm font-bold text-slate-900 mt-1">
                    {profile.address_line1 ? `${profile.address_line1}, ${profile.city}, ${profile.state} - ${profile.postal_code}, ${profile.country}` : 'Not provided'}
                  </p>
                </div>
              </div>
            </div>

            {/* Verification Status Card */}
            <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-3">
              <h3 className="font-bold text-slate-900 text-base">Verification Level: Basic Tier 1</h3>
              <p className="text-xs text-slate-600 leading-relaxed">
                Your email and telephone contact credentials have been verified through OTP challenge.
                To upgrade to Tier 2 (Corporate Verified) or Tier 3 (Enhanced Verification), request a credential from an authorized organization.
              </p>
              <div className="flex items-center space-x-2 pt-2 text-xs font-semibold text-teal-700">
                <CheckCircle2 className="w-4 h-4" />
                <span>Verified since {identity.verified_at ? new Date(identity.verified_at).toLocaleDateString() : 'Activation'}</span>
              </div>
            </div>
          </div>

          <div>
            <DigitalEkaCard identity={identity} profile={profile} qrUrl={generatedQR?.verify_url} />
          </div>
        </div>
      )}

      {/* TAB CONTENT 2: Digital Card Studio */}
      {activeTab === 'card' && identity && profile && (
        <div className="flex flex-col items-center py-4">
          <div className="text-center max-w-lg mb-6">
            <h3 className="text-xl font-bold text-slate-900">Your Official Digital EKA ID Card</h3>
            <p className="text-xs text-slate-500 mt-1">
              Download, print, or present this credential for secure platform verification.
            </p>
          </div>
          <DigitalEkaCard identity={identity} profile={profile} qrUrl={generatedQR?.verify_url} />
        </div>
      )}

      {/* TAB CONTENT 3: Dynamic QR Studio */}
      {activeTab === 'qr' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-6">
            <div>
              <h3 className="font-bold text-slate-900 text-base">Dynamic QR Generator</h3>
              <p className="text-xs text-slate-500 mt-1">
                Configure duration and which claims verifiers are authorized to see when scanning this QR.
              </p>
            </div>

            {/* Scope Selection */}
            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-2">
                Selective Disclosure Scopes
              </label>
              <div className="space-y-2">
                {[
                  { id: 'identity_valid', label: 'Identity Validity Status (Required)', locked: true },
                  { id: 'legal_name', label: 'Full Legal Name', locked: false },
                  { id: 'dob', label: 'Date of Birth & Age Check', locked: false },
                  { id: 'photo', label: 'Profile Photograph Match', locked: false },
                  { id: 'city_state', label: 'City & State Only (No Street Address)', locked: false },
                ].map((item) => (
                  <label
                    key={item.id}
                    className="flex items-center space-x-3 p-2.5 rounded-lg border border-slate-200 hover:bg-slate-50 cursor-pointer text-xs"
                  >
                    <input
                      type="checkbox"
                      disabled={item.locked}
                      checked={item.locked || qrScopes.includes(item.id)}
                      onChange={() => toggleScope(item.id)}
                      className="rounded text-teal-700 focus:ring-teal-600 w-4 h-4"
                    />
                    <span className="font-medium text-slate-800">{item.label}</span>
                  </label>
                ))}
              </div>
            </div>

            {/* Duration */}
            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-2">
                Token Expiration Window
              </label>
              <div className="grid grid-cols-3 gap-2">
                {[5, 15, 60].map((mins) => (
                  <button
                    key={mins}
                    type="button"
                    onClick={() => setQrDuration(mins)}
                    className={`py-2 text-xs font-semibold rounded-lg border transition ${
                      qrDuration === mins
                        ? 'bg-teal-700 text-white border-teal-700 shadow-sm'
                        : 'bg-white text-slate-700 border-slate-200 hover:bg-slate-50'
                    }`}
                  >
                    {mins} Minutes
                  </button>
                ))}
              </div>
            </div>

            <button
              onClick={handleGenerateCustomQR}
              disabled={qrGenerating}
              className="w-full py-2.5 bg-teal-700 hover:bg-teal-800 text-white text-xs font-bold rounded-lg shadow transition"
            >
              {qrGenerating ? 'Generating Signed Token...' : 'Generate New Verification QR'}
            </button>
          </div>

          {/* QR Display Preview */}
          <div className="bg-slate-900 text-white p-6 rounded-2xl border border-slate-800 flex flex-col items-center justify-center text-center">
            <h4 className="font-bold text-sm text-teal-400 mb-1">Live Verification QR</h4>
            <p className="text-[11px] text-slate-400 mb-4 max-w-xs">
              Valid for {qrDuration} minutes. Scan with any standard camera or EKA verifier.
            </p>

            {generatedQR && identity && (
              <div className="p-4 bg-white rounded-xl shadow-xl">
                <DigitalEkaCard identity={identity} profile={profile!} qrUrl={generatedQR.verify_url} />
              </div>
            )}
          </div>
        </div>
      )}

      {/* TAB CONTENT 4: Consent & Verification Requests */}
      {activeTab === 'requests' && (
        <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-6">
          <div>
            <h3 className="font-bold text-slate-900 text-base">Incoming Verification Requests</h3>
            <p className="text-xs text-slate-500 mt-1">
              Third-party organizations can only access your attributes when you explicitly approve the request.
            </p>
          </div>

          {requests.length === 0 ? (
            <div className="p-12 text-center text-slate-400 text-sm">
              <CheckCircle2 className="w-8 h-8 mx-auto mb-2 text-emerald-500" />
              No pending verification requests at this time.
            </div>
          ) : (
            <div className="space-y-4">
              {requests.map((req) => (
                <div key={req.id} className="p-4 rounded-xl border border-slate-200 bg-slate-50 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                  <div className="space-y-1 text-xs">
                    <div className="flex items-center space-x-2">
                      <span className="font-bold text-slate-900 text-sm">{req.org_name || 'Acme Technologies Ltd.'}</span>
                      <span className="px-2 py-0.5 rounded bg-amber-100 text-amber-800 text-[10px] font-semibold uppercase">Pending Consent</span>
                    </div>
                    <p className="text-slate-600"><strong>Purpose:</strong> {req.purpose}</p>
                    <p className="text-slate-500">
                      <strong>Requested Claims:</strong> {req.requested_scopes?.join(', ')}
                    </p>
                    <p className="text-slate-400 text-[11px]">Expires: {new Date(req.expires_at).toLocaleDateString()}</p>
                  </div>

                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => handleConsent(req.id, false)}
                      className="px-3.5 py-1.5 bg-white border border-slate-300 text-rose-600 hover:bg-rose-50 text-xs font-semibold rounded-lg transition"
                    >
                      Deny
                    </button>
                    <button
                      onClick={() => handleConsent(req.id, true)}
                      className="px-4 py-1.5 bg-teal-700 hover:bg-teal-800 text-white text-xs font-semibold rounded-lg shadow-sm transition"
                    >
                      Approve with Consent
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* TAB CONTENT 5: Credentials */}
      {activeTab === 'credentials' && (
        <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-6">
          <div>
            <h3 className="font-bold text-slate-900 text-base">Verifiable Credentials</h3>
            <p className="text-xs text-slate-500 mt-1">
              Cryptographically anchored claims issued by trusted employers, universities, and institutions.
            </p>
          </div>

          {credentials.length === 0 ? (
            <div className="p-12 text-center text-slate-400 text-sm">
              No credentials registered yet.
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {credentials.map((cred) => (
                <div key={cred.id} className="p-4 rounded-xl border border-slate-200 bg-slate-50 space-y-2 text-xs">
                  <div className="flex items-center justify-between">
                    <span className="font-bold text-slate-900 uppercase tracking-wider text-[11px]">{cred.type}</span>
                    <span className="px-2 py-0.5 rounded-full bg-teal-100 text-teal-800 font-semibold text-[10px]">
                      {cred.status}
                    </span>
                  </div>
                  <p className="font-semibold text-slate-800 text-sm">{cred.issuer_name}</p>
                  <p className="text-slate-500">Method: {cred.verification_method}</p>
                  {cred.metadata && (
                    <div className="p-2 rounded bg-white border border-slate-100 font-mono text-[11px] text-slate-600">
                      {JSON.stringify(cred.metadata)}
                    </div>
                  )}
                  <p className="text-[10px] text-slate-400">Issued: {new Date(cred.issued_at).toLocaleDateString()}</p>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* TAB CONTENT 6: Privacy & Rights */}
      {activeTab === 'privacy' && (
        <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-6 text-xs text-slate-600">
          <div>
            <h3 className="font-bold text-slate-900 text-base">Privacy & Data Architecture</h3>
            <p className="text-xs text-slate-500 mt-1">
              Your rights and architectural guarantees under EKA ID.
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 space-y-2">
              <h4 className="font-bold text-slate-900 text-sm">Data Minimization</h4>
              <p className="leading-relaxed">
                EKA ID enforces strict data minimization. Verifiers only receive what you have explicitly authorized. No mass scanning or demographic profiling is permitted.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 space-y-2">
              <h4 className="font-bold text-slate-900 text-sm">Zero-Knowledge ID Separation</h4>
              <p className="leading-relaxed">
                Your public identifier (<code>{identity?.eka_id}</code>) has zero correlation with your internal database primary key or personal identification numbers.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 space-y-2">
              <h4 className="font-bold text-slate-900 text-sm">Ephemeral QR Tokens</h4>
              <p className="leading-relaxed">
                QR codes generated on this device do not encode your name, address, or phone number. They point exclusively to short-lived signed tokens that you can revoke at any time.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 space-y-2">
              <h4 className="font-bold text-slate-900 text-sm">Auditability & Right of Access</h4>
              <p className="leading-relaxed">
                Every external request for your identity is permanently recorded in your audit trail. You have the right to review all actors who interacted with your credential.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}