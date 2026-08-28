'use client';

import React, { useEffect, useState, Suspense } from 'react';
import { useSearchParams } from 'next/navigation';
import { ShieldCheck, XCircle, Clock, CheckCircle2, QrCode, Search, AlertCircle, ArrowRight } from 'lucide-react';
import { verifyQRToken, QRVerificationResult } from '@/lib/api';

function VerifyContent() {
  const searchParams = useSearchParams();
  const tokenParam = searchParams.get('token');

  const [tokenInput, setTokenInput] = useState(tokenParam || '');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<QRVerificationResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (tokenParam) {
      handleVerify(tokenParam);
    }
  }, [tokenParam]);

  const handleVerify = async (tokenToVerify: string) => {
    if (!tokenToVerify.trim()) {
      setError('Please provide a verification token.');
      return;
    }
    setError(null);
    setResult(null);
    setLoading(true);

    try {
      const res = await verifyQRToken(tokenToVerify.trim());
      setResult(res);
    } catch (err: any) {
      setError(err.message || 'Verification unavailable: This verification request is no longer valid.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-[80vh] py-12 px-4 sm:px-6 lg:px-8 max-w-3xl mx-auto">
      <div className="text-center mb-8">
        <div className="w-12 h-12 rounded-xl bg-teal-700 flex items-center justify-center text-white mx-auto mb-3 shadow-md">
          <QrCode className="w-6 h-6" />
        </div>
        <h1 className="text-3xl font-extrabold text-slate-900 tracking-tight">
          EKA ID Verification
        </h1>
        <p className="text-xs text-slate-500 mt-1">
          Cryptographic token resolution with selective claim disclosure
        </p>
      </div>

      {/* Input / Scanner Section */}
      <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm mb-8">
        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleVerify(tokenInput);
          }}
          className="space-y-3"
        >
          <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider">
            Paste Verification Token or URL
          </label>
          <div className="flex flex-col sm:flex-row gap-2">
            <div className="relative flex-grow">
              <Search className="w-4 h-4 text-slate-400 absolute left-3 top-3" />
              <input
                type="text"
                placeholder="e.g. 4a9f3b1c8e..."
                value={tokenInput}
                onChange={(e) => setTokenInput(e.target.value)}
                className="w-full pl-9 pr-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-600 font-mono"
              />
            </div>
            <button
              type="submit"
              disabled={loading}
              className="px-6 py-2 bg-teal-700 hover:bg-teal-800 text-white text-sm font-semibold rounded-lg shadow-sm transition disabled:opacity-50 flex items-center justify-center space-x-2"
            >
              <span>{loading ? 'Verifying...' : 'Verify Now'}</span>
              <ArrowRight className="w-4 h-4" />
            </button>
          </div>
        </form>

        <p className="text-[11px] text-slate-400 mt-2">
          Tokens are issued with short lifetimes (typically 5–15 minutes) and reveal only owner-approved scopes.
        </p>
      </div>

      {/* Error / Invalidation Result */}
      {error && (
        <div className="p-6 rounded-2xl bg-rose-50 border border-rose-200 text-center space-y-3">
          <XCircle className="w-12 h-12 text-rose-600 mx-auto" />
          <h3 className="text-lg font-bold text-rose-900">Verification Unavailable</h3>
          <p className="text-sm text-rose-700 max-w-md mx-auto">{error}</p>
        </div>
      )}

      {/* Success Result View */}
      {result && (
        <div className="bg-white rounded-2xl border border-slate-200 shadow-lg overflow-hidden">
          {/* Header Banner */}
          <div className="bg-emerald-600 text-white p-6 text-center space-y-1">
            <div className="w-12 h-12 rounded-full bg-white/20 flex items-center justify-center mx-auto mb-2">
              <CheckCircle2 className="w-7 h-7 text-white" />
            </div>
            <h2 className="text-2xl font-extrabold tracking-tight">Identity Verified</h2>
            <p className="text-xs text-emerald-100 font-medium">
              EKA Digital Attestation Valid • Disclosed with User Consent
            </p>
          </div>

          <div className="p-6 sm:p-8 space-y-6">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
              <div className="p-4 rounded-xl bg-slate-50 border border-slate-100 space-y-1">
                <span className="text-[10px] text-slate-400 uppercase font-semibold tracking-wider">Public EKA ID</span>
                <p className="font-mono text-base font-bold text-teal-800">{result.eka_id}</p>
              </div>

              <div className="p-4 rounded-xl bg-slate-50 border border-slate-100 space-y-1">
                <span className="text-[10px] text-slate-400 uppercase font-semibold tracking-wider">Verification Level</span>
                <p className="text-sm font-bold text-slate-900">{result.verification_level?.replace(/_/g, ' ')}</p>
              </div>

              {result.legal_name && (
                <div className="p-4 rounded-xl bg-slate-50 border border-slate-100 space-y-1">
                  <span className="text-[10px] text-slate-400 uppercase font-semibold tracking-wider">Legal Name</span>
                  <p className="text-sm font-bold text-slate-900">{result.legal_name}</p>
                </div>
              )}

              <div className="p-4 rounded-xl bg-slate-50 border border-slate-100 space-y-1">
                <span className="text-[10px] text-slate-400 uppercase font-semibold tracking-wider">Verified Timestamp</span>
                <p className="text-sm font-bold text-slate-900">
                  {result.verified_at ? new Date(result.verified_at).toLocaleDateString() : 'Active'}
                </p>
              </div>
            </div>

            {/* Selectively Disclosed Claims */}
            <div className="space-y-3 pt-2">
              <h4 className="font-bold text-xs uppercase tracking-wider text-slate-700">
                Authorized Disclosed Claims
              </h4>
              <div className="p-4 rounded-xl bg-slate-900 text-teal-300 font-mono text-xs overflow-x-auto">
                <pre>{JSON.stringify(result.disclosed_claims, null, 2)}</pre>
              </div>
            </div>

            <div className="p-3 bg-slate-50 rounded-lg text-[11px] text-slate-500 text-center border border-slate-100">
              Verified on {new Date(result.verification_date).toLocaleString()} • Audit Event Logged
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default function VerifyPage() {
  return (
    <Suspense fallback={<div className="p-12 text-center text-sm text-slate-500">Loading verification module...</div>}>
      <VerifyContent />
    </Suspense>
  );
}