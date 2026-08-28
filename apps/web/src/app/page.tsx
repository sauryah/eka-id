import React from 'react';
import Link from 'next/link';
import { Shield, QrCode, CheckCircle, ArrowRight, Lock, EyeOff, FileText, Building2, Users } from 'lucide-react';

export default function LandingPage() {
  return (
    <div className="space-y-20 pb-20">
      {/* Hero Section */}
      <section className="relative overflow-hidden pt-16 pb-20 lg:pt-24 lg:pb-28 bg-gradient-to-b from-teal-50/50 via-white to-slate-50 border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center relative z-10">
          <div className="inline-flex items-center space-x-2 px-3 py-1 rounded-full bg-teal-100/80 border border-teal-200 text-teal-800 text-xs font-semibold uppercase tracking-wider mb-6">
            <Shield className="w-3.5 h-3.5" />
            <span>Universal Digital Identity Platform</span>
          </div>

          <h1 className="text-4xl sm:text-6xl font-extrabold text-slate-900 tracking-tight max-w-4xl mx-auto leading-tight sm:leading-none">
            One Identity.<br />
            <span className="text-teal-700">Verified Everywhere.</span>
          </h1>

          <p className="mt-6 text-lg sm:text-xl text-slate-600 max-w-2xl mx-auto leading-relaxed">
            EKA ID provides an independent, cryptographic digital identity platform designed for seamless verification, zero-PII sharing, and explicit user consent.
          </p>

          <div className="mt-10 flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link
              href="/register"
              className="w-full sm:w-auto inline-flex items-center justify-center space-x-2 px-7 py-3.5 rounded-xl bg-teal-700 hover:bg-teal-800 text-white font-semibold shadow-lg shadow-teal-700/20 transition-all text-base"
            >
              <span>Create EKA ID</span>
              <ArrowRight className="w-4 h-4" />
            </Link>
            <Link
              href="/verify"
              className="w-full sm:w-auto inline-flex items-center justify-center space-x-2 px-7 py-3.5 rounded-xl bg-white hover:bg-slate-50 text-slate-800 font-semibold border border-slate-300 shadow-sm transition text-base"
            >
              <QrCode className="w-4 h-4 text-slate-500" />
              <span>Verify an EKA ID</span>
            </Link>
          </div>

          {/* Non-Government Disclaimer Banner */}
          <div className="mt-12 max-w-2xl mx-auto p-4 rounded-xl bg-amber-50/90 border border-amber-200 text-amber-900 text-xs leading-relaxed text-left flex items-start space-x-3">
            <Shield className="w-5 h-5 text-amber-700 flex-shrink-0 mt-0.5" />
            <div>
              <span className="font-bold">Independent Architecture:</span> EKA ID is a private, platform-level identity ecosystem. It is not Aadhaar, not a replacement for Aadhaar, and not an official government identity.
            </div>
          </div>
        </div>
      </section>

      {/* Core Principles Section */}
      <section className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="text-center max-w-3xl mx-auto mb-16">
          <h2 className="text-xs font-bold text-teal-700 uppercase tracking-widest">Built On Integrity</h2>
          <p className="mt-2 text-3xl font-extrabold text-slate-900 sm:text-4xl tracking-tight">
            Architected for Privacy & Scale
          </p>
          <p className="mt-4 text-slate-600 text-base">
            Every feature is engineered to uphold data minimization and cryptographic separation.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          <div className="p-7 rounded-2xl bg-white border border-slate-200 shadow-sm hover:shadow-md transition">
            <div className="w-12 h-12 rounded-xl bg-teal-50 border border-teal-100 flex items-center justify-center text-teal-700 mb-5">
              <EyeOff className="w-6 h-6" />
            </div>
            <h3 className="text-lg font-bold text-slate-900 mb-2">Zero-PII in Public Identifiers</h3>
            <p className="text-slate-600 text-sm leading-relaxed">
              Your public EKA ID (e.g. <code>EKA-7K4M-92PX</code>) is generated with cryptographically secure Crockford Base32 entropy. It never encodes name, birthdate, or phone number.
            </p>
          </div>

          <div className="p-7 rounded-2xl bg-white border border-slate-200 shadow-sm hover:shadow-md transition">
            <div className="w-12 h-12 rounded-xl bg-teal-50 border border-teal-100 flex items-center justify-center text-teal-700 mb-5">
              <QrCode className="w-6 h-6" />
            </div>
            <h3 className="text-lg font-bold text-slate-900 mb-2">Short-Lived Signed QR Codes</h3>
            <p className="text-slate-600 text-sm leading-relaxed">
              QR codes never contain static personal data. Scanners receive a signed, revocable token with pre-authorized disclosure scopes that expire automatically.
            </p>
          </div>

          <div className="p-7 rounded-2xl bg-white border border-slate-200 shadow-sm hover:shadow-md transition">
            <div className="w-12 h-12 rounded-xl bg-teal-50 border border-teal-100 flex items-center justify-center text-teal-700 mb-5">
              <Lock className="w-6 h-6" />
            </div>
            <h3 className="text-lg font-bold text-slate-900 mb-2">Immutable Audit Ledger</h3>
            <p className="text-slate-600 text-sm leading-relaxed">
              Every verification event, status update, and consent authorization is permanently logged into an audit trail without storing raw secrets or passwords.
            </p>
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section className="bg-slate-100/70 border-y border-slate-200 py-16">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center max-w-3xl mx-auto mb-14">
            <h2 className="text-xs font-bold text-teal-700 uppercase tracking-widest">Workflow</h2>
            <p className="mt-2 text-3xl font-extrabold text-slate-900 sm:text-4xl tracking-tight">
              How EKA ID Works
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm text-center">
              <div className="w-8 h-8 rounded-full bg-teal-700 text-white font-bold flex items-center justify-center mx-auto mb-3 text-sm">1</div>
              <h4 className="font-bold text-slate-900 mb-1 text-sm">Register</h4>
              <p className="text-xs text-slate-500">Submit minimal personal details and verify contact via OTP.</p>
            </div>
            <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm text-center">
              <div className="w-8 h-8 rounded-full bg-teal-700 text-white font-bold flex items-center justify-center mx-auto mb-3 text-sm">2</div>
              <h4 className="font-bold text-slate-900 mb-1 text-sm">Receive EKA ID</h4>
              <p className="text-xs text-slate-500">System generates an unambiguous, collision-resistant identifier.</p>
            </div>
            <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm text-center">
              <div className="w-8 h-8 rounded-full bg-teal-700 text-white font-bold flex items-center justify-center mx-auto mb-3 text-sm">3</div>
              <h4 className="font-bold text-slate-900 mb-1 text-sm">Generate QR</h4>
              <p className="text-xs text-slate-500">Create a dynamic verification token with custom validity and scope.</p>
            </div>
            <div className="bg-white p-6 rounded-xl border border-slate-200 shadow-sm text-center">
              <div className="w-8 h-8 rounded-full bg-teal-700 text-white font-bold flex items-center justify-center mx-auto mb-3 text-sm">4</div>
              <h4 className="font-bold text-slate-900 mb-1 text-sm">Verify with Consent</h4>
              <p className="text-xs text-slate-500">Verifiers authenticate valid claims while maintaining privacy.</p>
            </div>
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="rounded-3xl bg-gradient-to-r from-slate-900 to-teal-900 p-8 sm:p-14 text-white shadow-xl flex flex-col md:flex-row items-center justify-between gap-8">
          <div className="space-y-3 text-center md:text-left">
            <h3 className="text-2xl sm:text-3xl font-extrabold tracking-tight">Ready to activate your EKA ID?</h3>
            <p className="text-slate-300 text-sm max-w-xl">
              Experience the modern standard in verifiable digital identity. Setup takes less than two minutes.
            </p>
          </div>
          <div className="flex flex-col sm:flex-row gap-3">
            <Link
              href="/register"
              className="px-6 py-3 rounded-xl bg-teal-500 hover:bg-teal-400 text-slate-950 font-bold transition shadow"
            >
              Get Started Now
            </Link>
            <Link
              href="/login"
              className="px-6 py-3 rounded-xl bg-white/10 hover:bg-white/20 text-white font-medium border border-white/20 transition text-center"
            >
              Demo Sign In
            </Link>
          </div>
        </div>
      </section>
    </div>
  );
}