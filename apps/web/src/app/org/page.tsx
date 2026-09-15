'use client';

import React, { useState, useEffect } from 'react';
import { Building2, Send, CheckCircle2, AlertCircle, Shield, Key, Search, FileText, Radio, Zap, Clock, UserCheck, XCircle } from 'lucide-react';
import { API_BASE, createVerificationRequest, loginUser } from '@/lib/api';

export default function OrgPage() {
  const [ekaId, setEkaId] = useState('EKA-7K4M-92PX');
  const [purpose, setPurpose] = useState('Employee Background Verification & Identity Check');
  const [scopes, setScopes] = useState<string[]>(['identity_valid', 'name_match', 'phone']);
  const [loading, setLoading] = useState(false);
  const [submittedRequest, setSubmittedRequest] = useState<any | null>(null);
  const [liveResponse, setLiveResponse] = useState<any | null>(null);
  const [error, setError] = useState<string | null>(null);

  const toggleScope = (scope: string) => {
    if (scopes.includes(scope)) {
      setScopes(scopes.filter((s) => s !== scope));
    } else {
      setScopes([...scopes, scope]);
    }
  };

  // Real-Time SSE Listener for Request Consent Response
  useEffect(() => {
    if (!submittedRequest?.id) return;

    const token = typeof window !== 'undefined' ? localStorage.getItem('eka_token') : null;
    if (!token) return;

    let eventSource: EventSource | null = null;
    try {
      const sseUrl = `${API_BASE}/api/v1/events/stream?token=${encodeURIComponent(token)}&topic=${encodeURIComponent(submittedRequest.id)}`;
      eventSource = new EventSource(sseUrl);

      eventSource.addEventListener('CONSENT_REQUEST_RESPONDED', (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data);
          if (data.payload?.request_id === submittedRequest.id || !data.payload?.request_id) {
            setLiveResponse(data.payload);
          }
        } catch (err) {
          console.error('Failed to parse SSE payload:', err);
        }
      });
    } catch (err) {
      console.warn('SSE stream error:', err);
    }

    return () => {
      if (eventSource) {
        eventSource.close();
      }
    };
  }, [submittedRequest]);

  const handleCreateRequest = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmittedRequest(null);
    setLiveResponse(null);
    setLoading(true);

    try {
      let token = localStorage.getItem('eka_token');
      if (!token) {
        // Auto-authenticate as Acme recruiter for seamless testing
        try {
          const auth = await loginUser('sarah.recruiter@acme.example.com', 'Password123!');
          token = auth.token;
          localStorage.setItem('eka_token', auth.token);
          localStorage.setItem('eka_user', JSON.stringify(auth.user));
        } catch (authErr) {
          // Fallback to standard admin/user credentials
          const auth = await loginUser('admin@eka.dev', 'Password123!');
          token = auth.token;
          localStorage.setItem('eka_token', auth.token);
          localStorage.setItem('eka_user', JSON.stringify(auth.user));
        }
      }

      const payload = {
        eka_id: ekaId.trim(),
        purpose: purpose.trim(),
        requested_scopes: scopes,
        duration_days: 7,
      };

      const res = await createVerificationRequest(token || '', payload);
      setSubmittedRequest(res);
    } catch (err: any) {
      setError(err.message || 'Failed to submit verification request.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-8">
        <div>
          <div className="inline-flex items-center space-x-2 px-3 py-1 rounded-full bg-teal-50 border border-teal-200 text-teal-800 text-xs font-semibold uppercase tracking-wider mb-2">
            <Building2 className="w-3.5 h-3.5" />
            <span>Organization Portal</span>
          </div>
          <h1 className="text-2xl sm:text-3xl font-extrabold text-slate-900 tracking-tight">
            Acme Technologies Ltd.
          </h1>
          <p className="text-xs text-slate-500 mt-1">
            Initiate consent-driven verification requests & query authorized identity claims.
          </p>
        </div>

        <div className="p-3 bg-white rounded-xl border border-slate-200 shadow-sm flex items-center space-x-3 text-xs">
          <Key className="w-4 h-4 text-teal-700" />
          <div>
            <span className="text-slate-400 uppercase text-[10px] font-semibold">API Status</span>
            <p className="font-bold text-slate-900">OAuth 2.1 Ready • Active</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Verification Request Form */}
        <div className="lg:col-span-2 bg-white p-6 sm:p-8 rounded-2xl border border-slate-200 shadow-sm space-y-6">
          <div>
            <h3 className="font-bold text-slate-900 text-base">Create Identity Verification Request</h3>
            <p className="text-xs text-slate-500 mt-1">
              The identity holder will be notified in their dashboard and must explicitly grant consent.
            </p>
          </div>

          {error && (
            <div className="p-3 rounded-lg bg-rose-50 border border-rose-200 text-rose-800 text-xs">
              {error}
            </div>
          )}

          <form onSubmit={handleCreateRequest} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                Target Public EKA ID
              </label>
              <div className="relative">
                <Search className="w-4 h-4 text-slate-400 absolute left-3 top-3" />
                <input
                  type="text"
                  required
                  placeholder="EKA-7K4M-92PX"
                  value={ekaId}
                  onChange={(e) => setEkaId(e.target.value)}
                  className="w-full pl-9 pr-3 py-2 text-sm font-mono border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-600"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                Verification Purpose
              </label>
              <input
                type="text"
                required
                placeholder="e.g. KYC Onboarding, Employment Screening"
                value={purpose}
                onChange={(e) => setPurpose(e.target.value)}
                className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-600"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-2">
                Requested Identity Claims
              </label>
              <div className="space-y-2">
                {[
                  { id: 'identity_valid', label: 'Identity is Active & Valid' },
                  { id: 'name_match', label: 'Full Legal Name Match' },
                  { id: 'dob', label: 'Date of Birth' },
                  { id: 'phone', label: 'Primary Contact Phone' },
                  { id: 'address', label: 'Residential Address Verification' },
                ].map((item) => (
                  <label
                    key={item.id}
                    className="flex items-center space-x-3 p-2.5 rounded-lg border border-slate-200 hover:bg-slate-50 cursor-pointer text-xs"
                  >
                    <input
                      type="checkbox"
                      checked={scopes.includes(item.id)}
                      onChange={() => toggleScope(item.id)}
                      className="rounded text-teal-700 focus:ring-teal-600 w-4 h-4"
                    />
                    <span className="font-medium text-slate-800">{item.label}</span>
                  </label>
                ))}
              </div>
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 bg-teal-700 hover:bg-teal-800 text-white text-sm font-semibold rounded-lg shadow-sm transition disabled:opacity-50 flex items-center justify-center space-x-2"
            >
              <Send className="w-4 h-4" />
              <span>{loading ? 'Submitting...' : 'Dispatch Verification Request'}</span>
            </button>
          </form>

          {/* Submission Result */}
          {submittedRequest && !liveResponse && (
            <div className="p-5 rounded-xl bg-amber-50 border border-amber-300 text-xs space-y-3 shadow-sm">
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2 text-amber-900 font-bold text-sm">
                  <Clock className="w-4 h-4 text-amber-700 animate-spin" />
                  <span>Verification Request Dispatched & Awaiting Consent</span>
                </div>
                <span className="inline-flex items-center space-x-1 px-2.5 py-0.5 rounded-full bg-amber-200 text-amber-900 text-[10px] font-bold">
                  <Radio className="w-3 h-3 text-amber-800 animate-pulse" />
                  <span>Real-Time SSE Loop Active</span>
                </span>
              </div>
              <div className="p-3 bg-white rounded-lg border border-amber-200 font-mono text-[11px] space-y-1">
                <p><span className="text-slate-500">Request UUID:</span> <strong className="text-slate-900">{submittedRequest.id}</strong></p>
                <p><span className="text-slate-500">Target Identity:</span> <strong className="text-slate-900">{submittedRequest.eka_id}</strong></p>
                <p><span className="text-slate-500">Status:</span> <span className="text-amber-700 font-bold uppercase">PENDING USER CONSENT</span></p>
              </div>
              <p className="text-amber-800">
                💡 <strong>Live Test Loop:</strong> Open the user dashboard in another tab (or approve in John Mathew's dashboard). The consent event will stream directly here in real-time!
              </p>
            </div>
          )}

          {/* Live Consent Response Streamed via SSE */}
          {liveResponse && (
            <div className={`p-5 rounded-xl border text-xs space-y-3 shadow-md ${liveResponse.approved ? 'bg-emerald-50 border-emerald-300' : 'bg-rose-50 border-rose-300'}`}>
              <div className="flex items-center justify-between">
                <div className="flex items-center space-x-2 font-bold text-sm">
                  {liveResponse.approved ? (
                    <>
                      <CheckCircle2 className="w-5 h-5 text-emerald-600" />
                      <span className="text-emerald-900">LIVE CONSENT GRANTED: Verified Claims Disclosed</span>
                    </>
                  ) : (
                    <>
                      <XCircle className="w-5 h-5 text-rose-600" />
                      <span className="text-rose-900">CONSENT REJECTED: Request Denied by Identity Holder</span>
                    </>
                  )}
                </div>
                <span className={`px-2.5 py-0.5 rounded-full text-[10px] font-bold ${liveResponse.approved ? 'bg-emerald-200 text-emerald-900' : 'bg-rose-200 text-rose-900'}`}>
                  Instant SSE Event Received
                </span>
              </div>

              {liveResponse.approved && liveResponse.result?.disclosed_claims && (
                <div className="p-4 bg-white rounded-xl border border-emerald-200 space-y-3 shadow-inner">
                  <h4 className="font-bold text-slate-900 text-xs uppercase tracking-wider">Authorized Zero-Knowledge Claims</h4>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-xs">
                    {Object.entries(liveResponse.result.disclosed_claims).map(([k, v]) => (
                      <div key={k} className="p-2 bg-slate-50 rounded border border-slate-200">
                        <span className="text-[10px] text-slate-500 uppercase font-semibold block">{k.replace(/_/g, ' ')}</span>
                        <span className="font-bold text-slate-900 font-mono">{String(v)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Info & API Specifications */}
        <div className="space-y-6 text-xs text-slate-600">
          <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-3">
            <h4 className="font-bold text-slate-900 text-sm flex items-center space-x-2">
              <Shield className="w-4 h-4 text-teal-700" />
              <span>Zero-Harvester Guarantee</span>
            </h4>
            <p className="leading-relaxed">
              Organizations cannot bulk-scrape the EKA ID database. Every query requires either a user-presented ephemeral QR token or an explicitly approved consent request.
            </p>
          </div>

          <div className="bg-slate-900 text-white p-6 rounded-2xl border border-slate-800 space-y-3">
            <h4 className="font-bold text-teal-400 text-sm">API Integration Snippet</h4>
            <pre className="p-3 bg-slate-950 rounded-lg text-slate-300 font-mono text-[11px] overflow-x-auto leading-relaxed">
{`POST /api/v1/verification-requests
Authorization: Bearer <ORG_API_KEY>
Content-Type: application/json

{
  "eka_id": "EKA-7K4M-92PX",
  "purpose": "KYC Onboarding",
  "requested_scopes": [
    "identity_valid",
    "name_match"
  ]
}`}
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
}