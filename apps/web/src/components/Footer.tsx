import React from 'react';
import Link from 'next/link';
import { ShieldCheck, Lock, EyeOff } from 'lucide-react';

export default function Footer() {
  return (
    <footer className="border-t border-slate-200 bg-white text-slate-600 text-sm">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-8 mb-8">
          <div className="md:col-span-2 space-y-3">
            <div className="flex items-center space-x-2">
              <div className="w-6 h-6 rounded bg-teal-700 flex items-center justify-center text-white font-bold text-xs">
                EKA
              </div>
              <span className="font-bold text-slate-900">EKA ID Platform</span>
            </div>
            <p className="text-xs text-slate-500 leading-relaxed max-w-md">
              A modern, privacy-first universal digital identity framework built on cryptographic randomness,
              zero-PII verification, and strict user consent disclosure.
            </p>
            <div className="p-3 bg-amber-50 border border-amber-200 rounded-lg text-amber-900 text-xs leading-relaxed max-w-lg">
              <strong>Notice:</strong> EKA ID is an independent private digital identity platform.
              It is not Aadhaar, not a replacement for Aadhaar, and does not represent an official government-issued identification.
            </div>
          </div>

          <div>
            <h4 className="font-semibold text-slate-900 mb-3 text-xs uppercase tracking-wider">Platform</h4>
            <ul className="space-y-2 text-xs">
              <li><Link href="/register" className="hover:text-teal-700 transition">Create EKA ID</Link></li>
              <li><Link href="/verify" className="hover:text-teal-700 transition">QR Verification</Link></li>
              <li><Link href="/org" className="hover:text-teal-700 transition">Organization API</Link></li>
              <li><Link href="/admin" className="hover:text-teal-700 transition">System Admin</Link></li>
            </ul>
          </div>

          <div>
            <h4 className="font-semibold text-slate-900 mb-3 text-xs uppercase tracking-wider">Security & Trust</h4>
            <ul className="space-y-2 text-xs text-slate-500">
              <li className="flex items-center space-x-1.5"><ShieldCheck className="w-3.5 h-3.5 text-teal-600" /><span>Crockford Base32 Entropy</span></li>
              <li className="flex items-center space-x-1.5"><EyeOff className="w-3.5 h-3.5 text-teal-600" /><span>Zero-PII in QR Codes</span></li>
              <li className="flex items-center space-x-1.5"><Lock className="w-3.5 h-3.5 text-teal-600" /><span>Immutable Audit Logging</span></li>
            </ul>
          </div>
        </div>

        <div className="border-t border-slate-100 pt-6 flex flex-col sm:flex-row items-center justify-between text-xs text-slate-400">
          <p>© 2026 EKA ID Universal Identity Platform. All rights reserved.</p>
          <p>Built for production resilience, privacy, and integrity.</p>
        </div>
      </div>
    </footer>
  );
}