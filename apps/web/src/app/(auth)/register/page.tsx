'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { Shield, ArrowRight, ArrowLeft, CheckCircle2, KeyRound, Mail, Phone, User, Calendar, Lock } from 'lucide-react';
import { requestOTP, registerUser } from '@/lib/api';

export default function RegisterPage() {
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form Fields
  const [legalName, setLegalName] = useState('');
  const [dateOfBirth, setDateOfBirth] = useState('');
  const [gender, setGender] = useState('MALE');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [password, setPassword] = useState('');
  const [otpCode, setOtpCode] = useState('');
  const [devOtpHint, setDevOtpHint] = useState<string | null>(null);

  // Registration Result
  const [createdEkaId, setCreatedEkaId] = useState<string | null>(null);

  const handleRequestOtp = async () => {
    if (!email || !phone) {
      setError('Please provide both email and phone number.');
      return;
    }
    setError(null);
    setLoading(true);
    try {
      const res = await requestOTP(email);
      setDevOtpHint(res.dev_otp || '123456');
      setOtpCode(res.dev_otp || '123456');
      setStep(3);
    } catch (err: any) {
      setError(err.message || 'Failed to dispatch OTP');
    } finally {
      setLoading(false);
    }
  };

  const handleFinalSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!password) {
      setError('Password is required.');
      return;
    }
    setError(null);
    setLoading(true);

    try {
      const payload = {
        legal_name: legalName,
        date_of_birth: dateOfBirth,
        gender,
        email,
        phone,
        password,
        otp_code: otpCode,
        country: 'India',
      };

      const res = await registerUser(payload);
      setCreatedEkaId(res.identity.eka_id);
      localStorage.setItem('eka_token', res.token);
      localStorage.setItem('eka_user', JSON.stringify(res.user));
      setStep(4); // Success step
    } catch (err: any) {
      setError(err.message || 'Registration failed.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-[85vh] flex flex-col justify-center py-12 sm:px-6 lg:px-8 bg-slate-50">
      <div className="sm:mx-auto sm:w-full sm:max-w-md">
        <div className="flex justify-center">
          <div className="w-12 h-12 rounded-xl bg-teal-700 flex items-center justify-center text-white shadow-md">
            <Shield className="w-7 h-7" />
          </div>
        </div>
        <h2 className="mt-4 text-center text-2xl font-extrabold text-slate-900 tracking-tight">
          Create your EKA ID
        </h2>
        <p className="mt-1 text-center text-xs text-slate-500">
          One person • One EKA ID • Cryptographically protected
        </p>
      </div>

      <div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
        <div className="bg-white py-8 px-6 shadow-sm rounded-2xl sm:px-10 border border-slate-200">
          {/* Progress Indicator */}
          {step < 4 && (
            <div className="flex items-center justify-between mb-8 pb-4 border-b border-slate-100">
              {[1, 2, 3].map((s) => (
                <div key={s} className="flex items-center space-x-2">
                  <div
                    className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold transition ${
                      step >= s
                        ? 'bg-teal-700 text-white'
                        : 'bg-slate-100 text-slate-400'
                    }`}
                  >
                    {s}
                  </div>
                  <span className="text-xs font-medium text-slate-600 hidden sm:inline">
                    {s === 1 ? 'Personal' : s === 2 ? 'Contact' : 'Verify'}
                  </span>
                </div>
              ))}
            </div>
          )}

          {error && (
            <div className="mb-6 p-3 rounded-lg bg-rose-50 border border-rose-200 text-rose-800 text-xs leading-relaxed">
              {error}
            </div>
          )}

          {/* STEP 1: Basic Person Info */}
          {step === 1 && (
            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (!legalName || !dateOfBirth) {
                  setError('Legal Name and Date of Birth are required.');
                  return;
                }
                setError(null);
                setStep(2);
              }}
              className="space-y-4"
            >
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Full Legal Name
                </label>
                <div className="relative">
                  <User className="w-4 h-4 text-slate-400 absolute left-3 top-3" />
                  <input
                    type="text"
                    required
                    placeholder="e.g. John Mathew"
                    value={legalName}
                    onChange={(e) => setLegalName(e.target.value)}
                    className="w-full pl-9 pr-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-600 focus:border-transparent"
                  />
                </div>
                <p className="text-[11px] text-slate-400 mt-1">Must match your official documents.</p>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Date of Birth
                </label>
                <div className="relative">
                  <Calendar className="w-4 h-4 text-slate-400 absolute left-3 top-3" />
                  <input
                    type="date"
                    required
                    value={dateOfBirth}
                    onChange={(e) => setDateOfBirth(e.target.value)}
                    className="w-full pl-9 pr-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-600 focus:border-transparent"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Gender (Optional)
                </label>
                <select
                  value={gender}
                  onChange={(e) => setGender(e.target.value)}
                  className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-600"
                >
                  <option value="MALE">Male</option>
                  <option value="FEMALE">Female</option>
                  <option value="NON_BINARY">Non-Binary / Other</option>
                  <option value="PREFER_NOT_TO_SAY">Prefer not to say</option>
                </select>
              </div>

              <button
                type="submit"
                className="w-full mt-4 flex items-center justify-center space-x-2 py-2.5 px-4 bg-teal-700 hover:bg-teal-800 text-white text-sm font-semibold rounded-lg shadow-sm transition"
              >
                <span>Continue to Contact</span>
                <ArrowRight className="w-4 h-4" />
              </button>
            </form>
          )}

          {/* STEP 2: Contact Information */}
          {step === 2 && (
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Email Address
                </label>
                <div className="relative">
                  <Mail className="w-4 h-4 text-slate-400 absolute left-3 top-3" />
                  <input
                    type="email"
                    required
                    placeholder="john.mathew@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    className="w-full pl-9 pr-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-600"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Mobile Phone Number
                </label>
                <div className="relative">
                  <Phone className="w-4 h-4 text-slate-400 absolute left-3 top-3" />
                  <input
                    type="tel"
                    required
                    placeholder="+91 98765 43210"
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    className="w-full pl-9 pr-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-600"
                  />
                </div>
              </div>

              <div className="flex items-center space-x-3 pt-2">
                <button
                  type="button"
                  onClick={() => setStep(1)}
                  className="w-1/3 py-2.5 px-3 border border-slate-300 text-slate-700 text-sm font-medium rounded-lg hover:bg-slate-50 transition"
                >
                  Back
                </button>
                <button
                  type="button"
                  disabled={loading}
                  onClick={handleRequestOtp}
                  className="w-2/3 flex items-center justify-center space-x-2 py-2.5 px-4 bg-teal-700 hover:bg-teal-800 text-white text-sm font-semibold rounded-lg shadow-sm transition disabled:opacity-50"
                >
                  <span>{loading ? 'Sending OTP...' : 'Send Verification OTP'}</span>
                  <ArrowRight className="w-4 h-4" />
                </button>
              </div>
            </div>
          )}

          {/* STEP 3: OTP & Password Completion */}
          {step === 3 && (
            <form onSubmit={handleFinalSubmit} className="space-y-4">
              <div className="p-3 bg-teal-50 border border-teal-200 rounded-lg text-teal-800 text-xs">
                <span>Verification code sent to <strong>{email}</strong>.</span>
                {devOtpHint && (
                  <div className="mt-1 font-mono text-[11px] text-teal-900 bg-teal-100/70 p-1 rounded">
                    Demo Code: <strong>{devOtpHint}</strong>
                  </div>
                )}
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  6-Digit OTP Code
                </label>
                <div className="relative">
                  <KeyRound className="w-4 h-4 text-slate-400 absolute left-3 top-3" />
                  <input
                    type="text"
                    required
                    maxLength={6}
                    placeholder="123456"
                    value={otpCode}
                    onChange={(e) => setOtpCode(e.target.value)}
                    className="w-full pl-9 pr-3 py-2 text-sm font-mono tracking-widest border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-600"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Create Master Password
                </label>
                <div className="relative">
                  <Lock className="w-4 h-4 text-slate-400 absolute left-3 top-3" />
                  <input
                    type="password"
                    required
                    placeholder="••••••••••••"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    className="w-full pl-9 pr-3 py-2 text-sm border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-teal-600"
                  />
                </div>
                <p className="text-[11px] text-slate-400 mt-1">Minimum 8 characters with letters & numbers.</p>
              </div>

              <div className="flex items-center space-x-3 pt-2">
                <button
                  type="button"
                  onClick={() => setStep(2)}
                  className="w-1/3 py-2.5 px-3 border border-slate-300 text-slate-700 text-sm font-medium rounded-lg hover:bg-slate-50 transition"
                >
                  Back
                </button>
                <button
                  type="submit"
                  disabled={loading}
                  className="w-2/3 flex items-center justify-center space-x-2 py-2.5 px-4 bg-teal-700 hover:bg-teal-800 text-white text-sm font-semibold rounded-lg shadow-sm transition disabled:opacity-50"
                >
                  <span>{loading ? 'Creating Identity...' : 'Finalize & Issue EKA ID'}</span>
                  <CheckCircle2 className="w-4 h-4" />
                </button>
              </div>
            </form>
          )}

          {/* STEP 4: Success & EKA ID Display */}
          {step === 4 && (
            <div className="text-center space-y-5 py-3">
              <div className="w-16 h-16 rounded-full bg-emerald-100 text-emerald-600 flex items-center justify-center mx-auto shadow-sm">
                <CheckCircle2 className="w-9 h-9" />
              </div>

              <div>
                <h3 className="text-lg font-bold text-slate-900">Your EKA ID is Active!</h3>
                <p className="text-xs text-slate-500 mt-1">
                  Cryptographically registered and ready for secure verification.
                </p>
              </div>

              <div className="p-4 rounded-xl bg-slate-900 text-white border border-slate-800 shadow-inner">
                <p className="text-[10px] uppercase tracking-widest text-teal-400 font-semibold mb-1">
                  Your Public Identifier
                </p>
                <p className="font-mono text-2xl font-extrabold text-white tracking-widest">
                  {createdEkaId}
                </p>
              </div>

              <div className="pt-2">
                <Link
                  href="/dashboard"
                  className="w-full inline-flex items-center justify-center space-x-2 py-3 px-4 bg-teal-700 hover:bg-teal-800 text-white text-sm font-semibold rounded-xl shadow transition"
                >
                  <span>Enter My Identity Dashboard</span>
                  <ArrowRight className="w-4 h-4" />
                </Link>
              </div>
            </div>
          )}

          <div className="mt-6 text-center text-xs text-slate-500">
            Already have an account?{' '}
            <Link href="/login" className="text-teal-700 hover:underline font-semibold">
              Sign in
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}