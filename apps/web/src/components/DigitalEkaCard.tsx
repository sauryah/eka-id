'use client';

import React, { useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { ShieldCheck, Printer, RotateCw, CheckCircle2, Shield, Calendar, MapPin } from 'lucide-react';
import { Identity, Profile } from '@/lib/api';

interface DigitalEkaCardProps {
  identity: Identity;
  profile: Profile;
  qrUrl?: string;
}

export default function DigitalEkaCard({ identity, profile, qrUrl }: DigitalEkaCardProps) {
  const [isFlipped, setIsFlipped] = useState(false);

  const verificationUrl = qrUrl || (typeof window !== 'undefined' ? `${window.location.origin}/verify?token=demo_token` : 'https://id.eka.dev/verify');

  const handlePrint = () => {
    window.print();
  };

  return (
    <div className="flex flex-col items-center">
      {/* Action Bar */}
      <div className="flex items-center space-x-3 mb-4 no-print">
        <button
          onClick={() => setIsFlipped(!isFlipped)}
          className="flex items-center space-x-1.5 px-3 py-1.5 bg-white border border-slate-200 text-slate-700 text-xs font-medium rounded-lg shadow-sm hover:bg-slate-50 transition"
        >
          <RotateCw className="w-3.5 h-3.5" />
          <span>Flip to {isFlipped ? 'Front' : 'Back'}</span>
        </button>
        <button
          onClick={handlePrint}
          className="flex items-center space-x-1.5 px-3 py-1.5 bg-teal-700 text-white text-xs font-medium rounded-lg shadow-sm hover:bg-teal-800 transition"
        >
          <Printer className="w-3.5 h-3.5" />
          <span>Print / Save as PDF</span>
        </button>
      </div>

      {/* Card Container */}
      <div
        id="printable-eka-card"
        className="w-[360px] sm:w-[380px] h-[520px] rounded-2xl bg-gradient-to-br from-slate-900 via-slate-800 to-teal-950 text-white shadow-2xl p-6 relative overflow-hidden border border-slate-700 flex flex-col justify-between"
      >
        {/* Background decorative patterns */}
        <div className="absolute top-0 right-0 -mr-16 -mt-16 w-64 h-64 rounded-full bg-teal-500/10 blur-3xl pointer-events-none" />
        <div className="absolute bottom-0 left-0 -ml-16 -mb-16 w-64 h-64 rounded-full bg-cyan-500/10 blur-3xl pointer-events-none" />

        {!isFlipped ? (
          // --- FRONT OF CARD ---
          <>
            {/* Header */}
            <div className="flex items-center justify-between border-b border-slate-700/60 pb-3">
              <div className="flex items-center space-x-2">
                <div className="w-7 h-7 rounded-md bg-teal-600 flex items-center justify-center text-white">
                  <Shield className="w-4 h-4" />
                </div>
                <div>
                  <h3 className="font-extrabold text-sm tracking-wider">EKA ID</h3>
                  <p className="text-[10px] text-teal-400 uppercase tracking-widest font-semibold">Universal Identity</p>
                </div>
              </div>
              <div className="flex items-center space-x-1 bg-emerald-500/20 text-emerald-300 text-[11px] font-semibold px-2 py-0.5 rounded-full border border-emerald-500/30">
                <CheckCircle2 className="w-3 h-3" />
                <span>ACTIVE</span>
              </div>
            </div>

            {/* Photo & Name */}
            <div className="flex flex-col items-center text-center my-auto py-2">
              <div className="relative mb-3">
                <div className="w-24 h-24 rounded-full p-1 bg-gradient-to-tr from-teal-400 to-cyan-300 shadow-md">
                  {profile?.profile_photo_url ? (
                    <img
                      src={profile.profile_photo_url}
                      alt={profile.legal_name}
                      className="w-full h-full rounded-full object-cover bg-slate-800"
                    />
                  ) : (
                    <div className="w-full h-full rounded-full bg-slate-800 flex items-center justify-center text-2xl font-bold text-teal-400">
                      {profile?.legal_name?.charAt(0) || 'E'}
                    </div>
                  )}
                </div>
                <div className="absolute bottom-0 right-1 w-6 h-6 rounded-full bg-teal-600 border-2 border-slate-900 flex items-center justify-center text-white shadow">
                  <ShieldCheck className="w-3.5 h-3.5" />
                </div>
              </div>

              <h2 className="text-xl font-bold text-slate-50 tracking-tight">{profile?.legal_name}</h2>
              <p className="text-xs text-slate-400 mt-0.5">{profile?.city ? `${profile.city}, ${profile.state || profile.country}` : 'Verified Digital Persona'}</p>

              {/* Public EKA ID - Prominent and Distinct */}
              <div className="mt-3 px-4 py-1.5 rounded-lg bg-slate-950/80 border border-slate-700/80 shadow-inner">
                <span className="text-xs font-medium text-slate-400 mr-2 uppercase tracking-wider">EKA ID</span>
                <span className="font-mono text-base font-bold text-teal-300 tracking-wider">
                  {identity?.eka_id}
                </span>
              </div>
            </div>

            {/* QR Code & Status */}
            <div className="flex items-center justify-between bg-slate-950/50 p-3 rounded-xl border border-slate-800">
              <div className="text-left space-y-1">
                <p className="text-[10px] text-slate-400 uppercase font-semibold tracking-wider">Verification Level</p>
                <div className="flex items-center space-x-1 text-xs font-semibold text-emerald-400">
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  <span>{identity?.verification_level?.replace(/_/g, ' ') || 'Basic Verified'}</span>
                </div>
                <p className="text-[10px] text-slate-500">Zero-PII Dynamic QR</p>
              </div>

              <div className="p-1.5 bg-white rounded-lg shadow-md">
                <QRCodeSVG value={verificationUrl} size={64} level="M" />
              </div>
            </div>

            {/* Card Footer Microcopy */}
            <div className="text-[9px] text-slate-500 text-center pt-2 border-t border-slate-800/80">
              Platform-Level Universal Digital Identity • Not a Government ID
            </div>
          </>
        ) : (
          // --- BACK OF CARD ---
          <>
            <div className="flex items-center justify-between border-b border-slate-700/60 pb-3">
              <span className="text-xs font-bold uppercase tracking-wider text-teal-400">Card Specifications</span>
              <span className="text-[10px] text-slate-400 font-mono">SEC-VER-1.0</span>
            </div>

            <div className="space-y-4 my-auto text-xs text-slate-300">
              <div className="p-3 bg-slate-950/60 rounded-xl border border-slate-800 space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-slate-400 text-[11px] flex items-center space-x-1.5">
                    <Calendar className="w-3 h-3 text-teal-400" />
                    <span>Verified Date</span>
                  </span>
                  <span className="font-semibold text-slate-200">
                    {identity?.verified_at ? new Date(identity.verified_at).toLocaleDateString() : 'Active'}
                  </span>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-slate-400 text-[11px] flex items-center space-x-1.5">
                    <MapPin className="w-3 h-3 text-teal-400" />
                    <span>Jurisdiction</span>
                  </span>
                  <span className="font-semibold text-slate-200">{profile?.country || 'India'}</span>
                </div>
              </div>

              <div className="p-3 bg-slate-950/60 rounded-xl border border-slate-800 space-y-1.5 text-[11px] leading-relaxed text-slate-400">
                <p className="font-semibold text-slate-200">Privacy & Security Guarantees:</p>
                <ul className="list-disc list-inside space-y-1">
                  <li>Public EKA ID contains no encoded personal information.</li>
                  <li>QR code contains cryptographic signed short-lived tokens only.</li>
                  <li>All verification checks are strictly logged in an immutable audit ledger.</li>
                </ul>
              </div>

              <div className="p-2.5 bg-amber-950/40 border border-amber-800/40 rounded-lg text-[10px] text-amber-200/90 leading-tight">
                <strong>Legal Disclaimer:</strong> EKA ID is an independent private digital identity. It is not an Aadhaar card, not government-issued, and is valid solely within participating platform ecosystems.
              </div>
            </div>

            <div className="text-[9px] text-slate-500 text-center pt-2 border-t border-slate-800/80">
              Report lost / compromised credentials immediately at id.eka.dev/security
            </div>
          </>
        )}
      </div>
    </div>
  );
}