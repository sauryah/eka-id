'use client';

import React, { useState } from 'react';
import { Building2, Send, CheckCircle2, AlertCircle, Shield, Key, Search, FileText } from 'lucide-react';
import { createVerificationRequest } from '@/lib/api';

export default function OrgPage() {
  const [ekaId, setEkaId] = useState('EKA-7K4M-92PX');
  const [purpose, setPurpose] = useState('Employee Background Verification & Identity Check');
  const [scopes, setScopes] = useState<string[]>(['identity_valid', 'name_match', 'phone']);
  const [loading, setLoading] = useState(false);
  const [submittedRequest, setSubmittedRequest] = useState<any | null>(null);
  const [error, setError] = useState<string | null>(null);

  const toggleScope = (scope: string) => {
    if (scopes.includes(scope)) {
      setScopes(scopes.filter((s) => s !== scope));
    } else {
      setScopes([...scopes, scope]);
    }
  };

  const handleCreateRequest = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSubmittedRequest(null);
    setLoading(true);

    try {
      const token = localStorage.getItem('eka_token') || 'dev_org_token';
      const payload = {
        eka_id: ekaId.trim(),
        purpose: purpose.trim(),
        requested_scopes: scopes,
        duration_days: 7,
      };

      const res = await createVerificationRequest(token, payload);
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
          {submittedRequest && (
            <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-xs space-y-2">
              <div className="flex items-center space-x-2 text-emerald-800 font-bold text-sm">
                <CheckCircle2 className="w-4 h-4" />
                <span>Verification Request Registered!</span>
              </div>
              <p className="text-emerald-700">Request ID: <code className="font-mono">{submittedRequest.id}</code></p>
              <p className="text-emerald-700">Status: <strong>PENDING IDENTITY OWNER CONSENT</strong></p>
              <p className="text-slate-600 mt-2">
                Tip: If you are testing as John Mathew, switch to your <strong>Dashboard &gt; Consent Requests</strong> tab to approve this request.
              </p>
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