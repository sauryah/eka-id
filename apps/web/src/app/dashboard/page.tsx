'use client';

import React, { useEffect, useState } from 'react';
import {
  Shield, QrCode, CheckCircle2, Award, Clock, Eye, Sliders,
  Check, X, AlertCircle, Lock, Download, Printer, UserCheck, RefreshCw, Key,
  FileCode, ExternalLink, Copy, Radio, Zap, FileText, Upload, Plus, FileCheck, XCircle
} from 'lucide-react';
import DigitalEkaCard from '@/components/DigitalEkaCard';
import {
  API_BASE,
  getMyIdentity, generateQRToken, getPendingVerificationRequests,
  respondVerificationRequest, getCredentialW3C, getDIDDocument,
  uploadIdentityDocument, listMyDocuments, createAmendmentRequest, listMyAmendments,
  Identity, Profile, Credential, VerificationRequest, W3CVerifiableCredential, DIDDocument,
  IdentityDocument, AmendmentRequest
} from '@/lib/api';

export default function DashboardPage() {
  const [activeTab, setActiveTab] = useState<'identity' | 'card' | 'qr' | 'requests' | 'credentials' | 'amendments' | 'privacy'>('identity');
  const [loading, setLoading] = useState(true);
  const [identity, setIdentity] = useState<Identity | null>(null);
  const [profile, setProfile] = useState<Profile | null>(null);
  const [credentials, setCredentials] = useState<Credential[]>([]);
  const [requests, setRequests] = useState<VerificationRequest[]>([]);
  const [documents, setDocuments] = useState<IdentityDocument[]>([]);
  const [amendments, setAmendments] = useState<AmendmentRequest[]>([]);
  const [error, setError] = useState<string | null>(null);

  // Real-Time SSE Stream State
  const [sseConnected, setSseConnected] = useState(false);
  const [liveEventMessage, setLiveEventMessage] = useState<string | null>(null);

  // Dynamic QR Studio State
  const [qrDuration, setQrDuration] = useState<number>(15);
  const [qrScopes, setQrScopes] = useState<string[]>(['identity_valid', 'legal_name']);
  const [generatedQR, setGeneratedQR] = useState<{ token: string; verify_url: string; expires_at: string } | null>(null);
  const [qrGenerating, setQrGenerating] = useState(false);

  // Request response feedback
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);

  // W3C VC Modal & DID Modal State
  const [selectedVC, setSelectedVC] = useState<W3CVerifiableCredential | null>(null);
  const [loadingVC, setLoadingVC] = useState(false);
  const [didDoc, setDidDoc] = useState<DIDDocument | null>(null);
  const [loadingDID, setLoadingDID] = useState(false);
  const [copiedText, setCopiedText] = useState(false);

  // Amendment Request Modal State
  const [showAmendmentModal, setShowAmendmentModal] = useState(false);
  const [amendForm, setAmendForm] = useState({
    legal_name: '',
    date_of_birth: '',
    gender: '',
    address_line1: '',
    city: '',
    state: '',
    postal_code: '',
    justification: '',
  });
  const [submittingAmendment, setSubmittingAmendment] = useState(false);
  const [uploadingDoc, setUploadingDoc] = useState(false);
  const [docType, setDocType] = useState('PASSPORT');
  const [selectedDocIDs, setSelectedDocIDs] = useState<string[]>([]);

  useEffect(() => {
    loadDashboardData();

    const token = typeof window !== 'undefined' ? localStorage.getItem('eka_token') : null;
    if (!token) return;

    let eventSource: EventSource | null = null;
    try {
      eventSource = new EventSource(`${API_BASE}/api/v1/events/stream?token=${encodeURIComponent(token)}`);

      eventSource.onopen = () => {
        setSseConnected(true);
      };

      eventSource.addEventListener('CONNECTED', () => {
        setSseConnected(true);
      });

      eventSource.addEventListener('CONSENT_REQUEST_CREATED', async (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data);
          const org = data.payload?.org_name || 'An organization';
          setLiveEventMessage(`🔔 Real-Time Alert: ${org} has sent an identity verification request!`);
          const updated = await getPendingVerificationRequests(token);
          setRequests(updated || []);
        } catch (err) {
          console.error('SSE message parse error:', err);
        }
      });

      eventSource.addEventListener('AMENDMENT_STATUS_CHANGED', async (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data);
          if (data.payload?.status === 'APPROVED') {
            setActionSuccess(`🎉 Profile amendment APPROVED! Your verified identity profile is updated.`);
          } else {
            setError(`Amendment request was ${data.payload?.status}: ${data.payload?.rejection_reason || 'See details'}`);
          }
          await loadDashboardData();
        } catch (err) {
          console.error('SSE amendment event error:', err);
        }
      });

      eventSource.onerror = () => {
        setSseConnected(false);
      };
    } catch (err) {
      console.warn('SSE subscription failed:', err);
    }

    return () => {
      if (eventSource) {
        eventSource.close();
      }
    };
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

      if (data.profile) {
        setAmendForm({
          legal_name: data.profile.legal_name || '',
          date_of_birth: data.profile.date_of_birth || '',
          gender: data.profile.gender || 'MALE',
          address_line1: data.profile.address_line1 || '',
          city: data.profile.city || '',
          state: data.profile.state || '',
          postal_code: data.profile.postal_code || '',
          justification: '',
        });
      }

      // Load pending verification requests
      try {
        const reqs = await getPendingVerificationRequests(token);
        setRequests(reqs || []);
      } catch (e) {
        console.warn('Could not load pending requests:', e);
      }

      // Load amendment history & documents
      try {
        const [amendList, docList] = await Promise.all([
          listMyAmendments(token),
          listMyDocuments(token),
        ]);
        setAmendments(amendList || []);
        setDocuments(docList || []);
      } catch (e) {
        console.warn('Could not load amendments/docs:', e);
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

  const handleToggleDocSelection = (docId: string) => {
    if (selectedDocIDs.includes(docId)) {
      setSelectedDocIDs(selectedDocIDs.filter((id) => id !== docId));
    } else {
      setSelectedDocIDs([...selectedDocIDs, docId]);
    }
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const token = localStorage.getItem('eka_token');
    if (!token) return;

    setUploadingDoc(true);
    try {
      const reader = new FileReader();
      reader.onload = async () => {
        const base64Content = reader.result as string;
        try {
          const uploaded = await uploadIdentityDocument(token, {
            document_type: docType,
            document_name: file.name,
            mime_type: file.type || 'application/octet-stream',
            file_content: base64Content,
          });
          setDocuments((prev) => [uploaded, ...prev]);
          setSelectedDocIDs((prev) => [...prev, uploaded.id]);
          setActionSuccess(`Document "${file.name}" uploaded (SHA-256: ${uploaded.sha256_hash.slice(0, 10)}...)`);
          setTimeout(() => setActionSuccess(null), 4000);
        } catch (uploadErr: any) {
          setError(uploadErr.message || 'Failed to upload document');
        } finally {
          setUploadingDoc(false);
        }
      };
      reader.readAsDataURL(file);
    } catch (err: any) {
      setError(err.message || 'Failed to process file');
      setUploadingDoc(false);
    }
  };

  const handleSubmitAmendment = async (e: React.FormEvent) => {
    e.preventDefault();
    const token = localStorage.getItem('eka_token');
    if (!token) return;

    setSubmittingAmendment(true);
    setError(null);
    try {
      const requestedChanges: Record<string, any> = {};
      if (amendForm.legal_name && amendForm.legal_name !== profile?.legal_name) {
        requestedChanges.legal_name = amendForm.legal_name.trim();
      }
      if (amendForm.date_of_birth && amendForm.date_of_birth !== profile?.date_of_birth) {
        requestedChanges.date_of_birth = amendForm.date_of_birth.trim();
      }
      if (amendForm.gender && amendForm.gender !== profile?.gender) {
        requestedChanges.gender = amendForm.gender.trim();
      }
      if (amendForm.address_line1 && amendForm.address_line1 !== profile?.address_line1) {
        requestedChanges.address_line1 = amendForm.address_line1.trim();
      }
      if (amendForm.city && amendForm.city !== profile?.city) {
        requestedChanges.city = amendForm.city.trim();
      }
      if (amendForm.state && amendForm.state !== profile?.state) {
        requestedChanges.state = amendForm.state.trim();
      }
      if (amendForm.postal_code && amendForm.postal_code !== profile?.postal_code) {
        requestedChanges.postal_code = amendForm.postal_code.trim();
      }

      if (Object.keys(requestedChanges).length === 0) {
        setError('Please modify at least one field to submit an amendment request.');
        setSubmittingAmendment(false);
        return;
      }

      const newAmend = await createAmendmentRequest(token, {
        requested_changes: requestedChanges,
        justification: amendForm.justification.trim() || 'Profile attribute amendment with supporting proof documents',
        document_ids: selectedDocIDs,
      });

      setAmendments((prev) => [newAmend, ...prev]);
      setShowAmendmentModal(false);
      setActionSuccess('Identity Amendment Request submitted for admin verification!');
      setActiveTab('amendments');
      setTimeout(() => setActionSuccess(null), 5000);
    } catch (err: any) {
      setError(err.message || 'Failed to submit amendment request');
    } finally {
      setSubmittingAmendment(false);
    }
  };

  const handleConsent = async (requestId: string, approved: boolean) => {
    const token = localStorage.getItem('eka_token');
    if (!token) return;

    try {
      await respondVerificationRequest(token, requestId, approved);
      setActionSuccess(approved ? 'Verification request approved with selective claims.' : 'Verification request denied.');
      const updated = await getPendingVerificationRequests(token);
      setRequests(updated || []);
      setTimeout(() => setActionSuccess(null), 4000);
    } catch (err: any) {
      setError(err.message || 'Failed to process request');
    }
  };

  const handleInspectW3C = async (credId: string) => {
    setLoadingVC(true);
    try {
      const vcData = await getCredentialW3C(credId);
      setSelectedVC(vcData);
    } catch (err: any) {
      setError(err.message || 'Failed to load W3C credential');
    } finally {
      setLoadingVC(false);
    }
  };

  const handleViewDID = async () => {
    if (!identity?.eka_id) return;
    setLoadingDID(true);
    try {
      const doc = await getDIDDocument(`did:eka:${identity.eka_id}`);
      setDidDoc(doc);
    } catch (err: any) {
      setError(err.message || 'Failed to resolve DID document');
    } finally {
      setLoadingDID(false);
    }
  };

  const handleCopyJSON = (jsonObj: any) => {
    navigator.clipboard.writeText(JSON.stringify(jsonObj, null, 2));
    setCopiedText(true);
    setTimeout(() => setCopiedText(false), 2000);
  };

  const handleDownloadJSON = (jsonObj: any, filename: string) => {
    const blob = new Blob([JSON.stringify(jsonObj, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
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
                {sseConnected && (
                  <span className="flex items-center space-x-1 px-2 py-0.5 bg-teal-500/20 border border-teal-400/40 text-teal-300 rounded-full text-[11px] font-semibold">
                    <Radio className="w-3 h-3 text-emerald-400 animate-pulse" />
                    <span>Live SSE Stream</span>
                  </span>
                )}
              </div>
              <p className="text-slate-300 text-xs mt-0.5">{profile?.email} • {profile?.phone}</p>
              <div className="mt-1.5 flex items-center space-x-2">
                <span className="text-[11px] font-mono text-teal-300 bg-slate-950/60 px-2 py-0.5 rounded border border-teal-500/30">
                  did:eka:{identity?.eka_id}
                </span>
                <button
                  onClick={handleViewDID}
                  className="text-[11px] text-teal-300 hover:text-white underline font-medium flex items-center space-x-1"
                >
                  <FileCode className="w-3 h-3" />
                  <span>W3C DID Doc</span>
                </button>
              </div>
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

      {liveEventMessage && (
        <div className="mb-6 p-4 rounded-xl bg-amber-50 border border-amber-300 text-amber-900 text-sm flex items-center justify-between shadow-sm animate-bounce-once">
          <div className="flex items-center space-x-2">
            <Zap className="w-5 h-5 text-amber-600 flex-shrink-0 animate-pulse" />
            <span className="font-semibold">{liveEventMessage}</span>
          </div>
          <button
            onClick={() => {
              setLiveEventMessage(null);
              setActiveTab('requests');
            }}
            className="px-3 py-1 bg-amber-600 hover:bg-amber-700 text-white rounded-lg text-xs font-bold transition"
          >
            Review Now
          </button>
        </div>
      )}

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
          { id: 'amendments', label: `Amendments & KYC (${amendments.length})`, icon: FileText },
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
              <div className="flex items-center justify-between">
                <h3 className="font-bold text-slate-900 text-base flex items-center space-x-2">
                  <Shield className="w-5 h-5 text-teal-700" />
                  <span>Identity Attributes</span>
                </h3>
                <button
                  onClick={() => setShowAmendmentModal(true)}
                  className="inline-flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-teal-50 border border-teal-200 text-teal-800 hover:bg-teal-100 text-xs font-semibold transition"
                >
                  <Plus className="w-3.5 h-3.5" />
                  <span>Request Profile Amendment</span>
                </button>
              </div>

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
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <h3 className="font-bold text-slate-900 text-base">Verifiable Credentials (W3C Standard)</h3>
              <p className="text-xs text-slate-500 mt-0.5">
                Cryptographically signed claims compliant with the W3C Verifiable Credentials Data Model v1.1/v2.0.
              </p>
            </div>
            <button
              onClick={handleViewDID}
              className="inline-flex items-center space-x-1.5 px-3 py-1.5 rounded-lg border border-slate-200 bg-slate-50 text-slate-700 text-xs font-semibold hover:bg-slate-100 transition shadow-sm"
            >
              <FileCode className="w-3.5 h-3.5 text-teal-700" />
              <span>Resolve Subject DID Document</span>
            </button>
          </div>

          {credentials.length === 0 ? (
            <div className="p-12 text-center text-slate-400 text-sm">
              No credentials registered yet.
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {credentials.map((cred) => (
                <div key={cred.id} className="p-5 rounded-xl border border-slate-200 bg-slate-50 space-y-3 text-xs flex flex-col justify-between">
                  <div className="space-y-2">
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

                  <div className="pt-2 border-t border-slate-200 flex items-center justify-between gap-2">
                    <button
                      onClick={() => handleInspectW3C(cred.id)}
                      className="inline-flex items-center space-x-1 px-2.5 py-1 rounded bg-teal-700 text-white font-semibold text-[11px] hover:bg-teal-800 transition"
                    >
                      <FileCode className="w-3 h-3" />
                      <span>Inspect W3C VC</span>
                    </button>
                    <span className="text-[10px] text-slate-400 font-mono">W3C VC JSON-LD</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* TAB CONTENT 6: Identity Amendments & KYC Proofs */}
      {activeTab === 'amendments' && (
        <div className="space-y-8">
          {/* Header Banner */}
          <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div>
              <h3 className="font-bold text-slate-900 text-base flex items-center space-x-2">
                <FileText className="w-5 h-5 text-teal-700" />
                <span>Identity Amendment & KYC Vault</span>
              </h3>
              <p className="text-xs text-slate-500 mt-0.5">
                Submit verified profile updates with cryptographic proof documents (SHA-256 non-repudiation).
              </p>
            </div>
            <button
              onClick={() => setShowAmendmentModal(true)}
              className="inline-flex items-center space-x-1.5 px-4 py-2 bg-teal-700 hover:bg-teal-800 text-white rounded-lg text-xs font-semibold shadow-sm transition"
            >
              <Plus className="w-4 h-4" />
              <span>New Amendment Request</span>
            </button>
          </div>

          {/* Section 1: Amendment Requests History */}
          <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-6">
            <div className="flex items-center justify-between">
              <h4 className="font-bold text-slate-900 text-sm">Amendment Requests History</h4>
              <span className="text-xs text-slate-500">{amendments.length} Total Requests</span>
            </div>

            {amendments.length === 0 ? (
              <div className="p-12 text-center text-slate-400 text-xs border border-dashed border-slate-200 rounded-xl">
                <FileCheck className="w-8 h-8 mx-auto mb-2 text-slate-300" />
                No amendment requests submitted. Click &quot;New Amendment Request&quot; above to request profile modifications.
              </div>
            ) : (
              <div className="space-y-4">
                {amendments.map((amend) => (
                  <div key={amend.id} className="p-5 rounded-xl border border-slate-200 bg-slate-50 space-y-4">
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-slate-200/80 pb-3">
                      <div className="flex items-center space-x-2">
                        <span className="font-mono text-xs font-bold text-slate-700">REQ-{amend.id.slice(0, 8)}</span>
                        <span
                          className={`px-2.5 py-0.5 rounded-full text-[11px] font-bold uppercase tracking-wider ${
                            amend.status === 'APPROVED'
                              ? 'bg-emerald-100 text-emerald-800 border border-emerald-300'
                              : amend.status === 'REJECTED'
                              ? 'bg-rose-100 text-rose-800 border border-rose-300'
                              : 'bg-amber-100 text-amber-800 border border-amber-300'
                          }`}
                        >
                          {amend.status}
                        </span>
                      </div>
                      <span className="text-[11px] text-slate-400">
                        Submitted: {new Date(amend.created_at).toLocaleString()}
                      </span>
                    </div>

                    {/* Justification */}
                    <div className="text-xs text-slate-700">
                      <span className="font-semibold text-slate-900">Reason / Justification: </span>
                      <span>{amend.justification}</span>
                    </div>

                    {/* Diff Table of Changes */}
                    <div className="border border-slate-200 rounded-lg overflow-hidden bg-white">
                      <table className="w-full text-left text-xs">
                        <thead className="bg-slate-100 text-slate-600 font-semibold uppercase text-[10px] tracking-wider border-b border-slate-200">
                          <tr>
                            <th className="py-2 px-3">Field</th>
                            <th className="py-2 px-3">Previous Value (Snapshot)</th>
                            <th className="py-2 px-3">Requested Value</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100 font-mono text-[11px]">
                          {Object.entries(amend.requested_changes || {}).map(([key, newVal]) => {
                            const prevVal = amend.snapshot_prev_values?.[key] ?? '(none)';
                            return (
                              <tr key={key} className="hover:bg-slate-50/60">
                                <td className="py-2 px-3 font-semibold text-slate-800 capitalize font-sans">
                                  {key.replace(/_/g, ' ')}
                                </td>
                                <td className="py-2 px-3 text-slate-400 line-through">
                                  {String(prevVal)}
                                </td>
                                <td className="py-2 px-3 text-teal-700 font-bold">
                                  {String(newVal)}
                                </td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>

                    {/* Attached Documents */}
                    {amend.documents && amend.documents.length > 0 && (
                      <div className="space-y-1.5 pt-1">
                        <span className="text-[11px] font-semibold text-slate-700">Attached Proof Documents ({amend.documents.length}):</span>
                        <div className="flex flex-wrap gap-2">
                          {amend.documents.map((doc) => (
                            <div
                              key={doc.id}
                              className="p-2 rounded-lg bg-white border border-slate-200 text-xs flex items-center space-x-2 shadow-xs"
                            >
                              <FileCheck className="w-3.5 h-3.5 text-teal-700 flex-shrink-0" />
                              <div>
                                <span className="font-semibold text-slate-800">{doc.document_name}</span>
                                <span className="text-[10px] text-slate-500 ml-1.5 font-mono">[{doc.document_type}]</span>
                                <p className="text-[10px] text-slate-400 font-mono">SHA-256: {doc.sha256_hash.slice(0, 16)}...</p>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Review Feedback */}
                    {amend.status === 'APPROVED' && (
                      <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-lg text-xs text-emerald-800 flex items-center space-x-2">
                        <CheckCircle2 className="w-4 h-4 text-emerald-600 flex-shrink-0" />
                        <span>
                          Approved by <strong>{amend.reviewed_by || 'System Admin'}</strong> on{' '}
                          {amend.reviewed_at ? new Date(amend.reviewed_at).toLocaleString() : 'N/A'}. Profile updated and W3C credential re-minted.
                        </span>
                      </div>
                    )}

                    {amend.status === 'REJECTED' && (
                      <div className="p-3 bg-rose-50 border border-rose-200 rounded-lg text-xs text-rose-800 flex items-start space-x-2">
                        <XCircle className="w-4 h-4 text-rose-600 flex-shrink-0 mt-0.5" />
                        <div>
                          <p className="font-bold">Amendment Rejected:</p>
                          <p className="mt-0.5">{amend.rejection_reason || 'Supporting document verification could not be authenticated.'}</p>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Section 2: KYC & Identity Proof Vault */}
          <div className="bg-white p-6 rounded-2xl border border-slate-200 shadow-sm space-y-6">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div>
                <h4 className="font-bold text-slate-900 text-sm">Cryptographic KYC Document Vault</h4>
                <p className="text-xs text-slate-500 mt-0.5">
                  Uploaded documents are cryptographically hashed (SHA-256) for non-repudiation and immutable verification.
                </p>
              </div>
            </div>

            {/* Inline Quick Upload */}
            <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 space-y-3">
              <span className="text-xs font-bold text-slate-800">Upload New Supporting Document</span>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <div>
                  <label className="block text-[11px] font-semibold text-slate-600 mb-1">Document Category</label>
                  <select
                    value={docType}
                    onChange={(e) => setDocType(e.target.value)}
                    className="w-full text-xs p-2 rounded-lg border border-slate-300 bg-white text-slate-800 focus:ring-1 focus:ring-teal-700"
                  >
                    <option value="PASSPORT">Passport</option>
                    <option value="NATIONAL_ID">National ID / Aadhaar</option>
                    <option value="BIRTH_CERTIFICATE">Birth Certificate</option>
                    <option value="DRIVING_LICENSE">Driving License</option>
                    <option value="UTILITY_BILL">Utility Bill (Address Proof)</option>
                    <option value="MARRIAGE_CERTIFICATE">Marriage Certificate</option>
                    <option value="GAZETTE_NOTIFICATION">Gazette Notification</option>
                    <option value="OTHER">Other Proof Document</option>
                  </select>
                </div>

                <div className="sm:col-span-2">
                  <label className="block text-[11px] font-semibold text-slate-600 mb-1">Select File (PDF, PNG, JPG)</label>
                  <div className="flex items-center space-x-2">
                    <input
                      type="file"
                      accept=".pdf,image/png,image/jpeg,image/webp"
                      onChange={handleFileUpload}
                      disabled={uploadingDoc}
                      className="text-xs text-slate-600 file:mr-2 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:text-xs file:font-semibold file:bg-teal-700 file:text-white hover:file:bg-teal-800 cursor-pointer"
                    />
                    {uploadingDoc && (
                      <span className="text-xs text-teal-700 font-semibold flex items-center space-x-1">
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        <span>Hashing & Storing...</span>
                      </span>
                    )}
                  </div>
                </div>
              </div>
            </div>

            {/* Document Vault List */}
            {documents.length === 0 ? (
              <div className="p-8 text-center text-slate-400 text-xs border border-dashed border-slate-200 rounded-xl">
                No documents uploaded yet in your KYC vault.
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {documents.map((doc) => (
                  <div
                    key={doc.id}
                    className="p-3.5 rounded-xl border border-slate-200 bg-white hover:border-teal-200 transition space-y-2 shadow-xs"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center space-x-2 truncate">
                        <FileText className="w-4 h-4 text-teal-700 flex-shrink-0" />
                        <span className="font-semibold text-xs text-slate-900 truncate">{doc.document_name}</span>
                      </div>
                      <span className="px-2 py-0.5 rounded bg-teal-50 text-teal-800 border border-teal-200 text-[10px] font-bold">
                        {doc.document_type}
                      </span>
                    </div>

                    <div className="space-y-1 text-[11px] text-slate-500 font-mono">
                      <div className="flex items-center justify-between">
                        <span>MIME: {doc.mime_type}</span>
                        <span>{new Date(doc.created_at).toLocaleDateString()}</span>
                      </div>
                      <div className="p-1.5 bg-slate-50 rounded border border-slate-100 flex items-center justify-between">
                        <span className="truncate text-[10px] text-slate-600 font-mono">
                          SHA: {doc.sha256_hash}
                        </span>
                        <Lock className="w-3 h-3 text-emerald-600 flex-shrink-0 ml-1" />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* TAB CONTENT 7: Privacy & Rights */}
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
              <h4 className="font-bold text-slate-900 text-sm">Decentralized Identifiers (DID)</h4>
              <p className="leading-relaxed">
                Your public identifier (<code>did:eka:{identity?.eka_id}</code>) conforms to W3C DID Core 1.0. It resolves cryptographically without exposing backend database keys.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 space-y-2">
              <h4 className="font-bold text-slate-900 text-sm">Ephemeral QR Tokens</h4>
              <p className="leading-relaxed">
                QR codes generated on this device do not encode your name, address, or phone number. They point exclusively to short-lived signed tokens that you can revoke at any time.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 space-y-2">
              <h4 className="font-bold text-slate-900 text-sm">W3C Verifiable Credentials</h4>
              <p className="leading-relaxed">
                All claims carry cryptographic tamper-evident signatures (HMAC-SHA256 / JWS proofs), allowing verifiers worldwide to validate credential integrity.
              </p>
            </div>
          </div>
        </div>
      )}

      {/* W3C VC Inspector Modal */}
      {selectedVC && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm">
          <div className="bg-white rounded-2xl max-w-2xl w-full p-6 shadow-2xl border border-slate-200 max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div className="flex items-center space-x-2">
                <FileCode className="w-5 h-5 text-teal-700" />
                <h3 className="font-bold text-slate-900 text-base">W3C Verifiable Credential (JSON-LD)</h3>
              </div>
              <button
                onClick={() => setSelectedVC(null)}
                className="p-1 rounded-lg hover:bg-slate-100 text-slate-400 hover:text-slate-600 transition"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="my-4 p-4 rounded-xl bg-slate-950 text-teal-300 font-mono text-xs overflow-y-auto flex-grow shadow-inner">
              <pre>{JSON.stringify(selectedVC, null, 2)}</pre>
            </div>

            <div className="pt-3 border-t border-slate-100 flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <button
                  onClick={() => handleCopyJSON(selectedVC)}
                  className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg border border-slate-200 bg-slate-50 hover:bg-slate-100 text-slate-700 text-xs font-semibold transition"
                >
                  <Copy className="w-3.5 h-3.5" />
                  <span>{copiedText ? 'Copied!' : 'Copy JSON'}</span>
                </button>
                <button
                  onClick={() => handleDownloadJSON(selectedVC, `w3c-credential-${selectedVC.id.split(':').pop()}.json`)}
                  className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-teal-700 hover:bg-teal-800 text-white text-xs font-semibold shadow-sm transition"
                >
                  <Download className="w-3.5 h-3.5" />
                  <span>Download JSON-LD</span>
                </button>
              </div>

              <button
                onClick={() => setSelectedVC(null)}
                className="px-4 py-1.5 rounded-lg text-slate-600 hover:bg-slate-100 text-xs font-semibold transition"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* DID Document Modal */}
      {didDoc && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm">
          <div className="bg-white rounded-2xl max-w-2xl w-full p-6 shadow-2xl border border-slate-200 max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div className="flex items-center space-x-2">
                <FileCode className="w-5 h-5 text-teal-700" />
                <h3 className="font-bold text-slate-900 text-base">W3C Decentralized Identifier (DID) Document</h3>
              </div>
              <button
                onClick={() => setDidDoc(null)}
                className="p-1 rounded-lg hover:bg-slate-100 text-slate-400 hover:text-slate-600 transition"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="my-4 p-4 rounded-xl bg-slate-950 text-cyan-300 font-mono text-xs overflow-y-auto flex-grow shadow-inner">
              <pre>{JSON.stringify(didDoc, null, 2)}</pre>
            </div>

            <div className="pt-3 border-t border-slate-100 flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <button
                  onClick={() => handleCopyJSON(didDoc)}
                  className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg border border-slate-200 bg-slate-50 hover:bg-slate-100 text-slate-700 text-xs font-semibold transition"
                >
                  <Copy className="w-3.5 h-3.5" />
                  <span>{copiedText ? 'Copied!' : 'Copy DID Doc'}</span>
                </button>
                <button
                  onClick={() => handleDownloadJSON(didDoc, `did-document.json`)}
                  className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-teal-700 hover:bg-teal-800 text-white text-xs font-semibold shadow-sm transition"
                >
                  <Download className="w-3.5 h-3.5" />
                  <span>Download DID JSON</span>
                </button>
              </div>

              <button
                onClick={() => setDidDoc(null)}
                className="px-4 py-1.5 rounded-lg text-slate-600 hover:bg-slate-100 text-xs font-semibold transition"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Profile Amendment Request Modal */}
      {showAmendmentModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-sm">
          <div className="bg-white rounded-2xl max-w-2xl w-full p-6 shadow-2xl border border-slate-200 max-h-[90vh] flex flex-col">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div className="flex items-center space-x-2">
                <FileText className="w-5 h-5 text-teal-700" />
                <div>
                  <h3 className="font-bold text-slate-900 text-base">Request Profile Attribute Amendment</h3>
                  <p className="text-[11px] text-slate-500">Submit official updates with cryptographic proof documents</p>
                </div>
              </div>
              <button
                onClick={() => setShowAmendmentModal(false)}
                className="p-1 rounded-lg hover:bg-slate-100 text-slate-400 hover:text-slate-600 transition"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSubmitAmendment} className="space-y-4 my-4 overflow-y-auto flex-grow pr-1">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">Legal Name</label>
                  <input
                    type="text"
                    value={amendForm.legal_name}
                    onChange={(e) => setAmendForm({ ...amendForm, legal_name: e.target.value })}
                    placeholder="Full legal name"
                    className="w-full text-xs p-2.5 rounded-lg border border-slate-300 focus:ring-1 focus:ring-teal-700 focus:border-teal-700"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">Date of Birth</label>
                  <input
                    type="date"
                    value={amendForm.date_of_birth}
                    onChange={(e) => setAmendForm({ ...amendForm, date_of_birth: e.target.value })}
                    className="w-full text-xs p-2.5 rounded-lg border border-slate-300 focus:ring-1 focus:ring-teal-700 focus:border-teal-700"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">Gender</label>
                  <select
                    value={amendForm.gender}
                    onChange={(e) => setAmendForm({ ...amendForm, gender: e.target.value })}
                    className="w-full text-xs p-2.5 rounded-lg border border-slate-300 bg-white focus:ring-1 focus:ring-teal-700"
                  >
                    <option value="MALE">Male</option>
                    <option value="FEMALE">Female</option>
                    <option value="OTHER">Other</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">Residential Street Address</label>
                  <input
                    type="text"
                    value={amendForm.address_line1}
                    onChange={(e) => setAmendForm({ ...amendForm, address_line1: e.target.value })}
                    placeholder="Street address line"
                    className="w-full text-xs p-2.5 rounded-lg border border-slate-300 focus:ring-1 focus:ring-teal-700 focus:border-teal-700"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">City</label>
                  <input
                    type="text"
                    value={amendForm.city}
                    onChange={(e) => setAmendForm({ ...amendForm, city: e.target.value })}
                    placeholder="City"
                    className="w-full text-xs p-2.5 rounded-lg border border-slate-300 focus:ring-1 focus:ring-teal-700"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">State / Province</label>
                  <input
                    type="text"
                    value={amendForm.state}
                    onChange={(e) => setAmendForm({ ...amendForm, state: e.target.value })}
                    placeholder="State"
                    className="w-full text-xs p-2.5 rounded-lg border border-slate-300 focus:ring-1 focus:ring-teal-700"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">Postal Code</label>
                  <input
                    type="text"
                    value={amendForm.postal_code}
                    onChange={(e) => setAmendForm({ ...amendForm, postal_code: e.target.value })}
                    placeholder="Postal / ZIP Code"
                    className="w-full text-xs p-2.5 rounded-lg border border-slate-300 focus:ring-1 focus:ring-teal-700"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Justification / Reason for Amendment</label>
                <textarea
                  rows={2}
                  required
                  value={amendForm.justification}
                  onChange={(e) => setAmendForm({ ...amendForm, justification: e.target.value })}
                  placeholder="e.g., Legal name update as per Gazette Notification #2026-B or address relocation."
                  className="w-full text-xs p-2.5 rounded-lg border border-slate-300 focus:ring-1 focus:ring-teal-700"
                />
              </div>

              {/* Supporting Proof Documents Attachment */}
              <div className="space-y-2 border-t border-slate-100 pt-3">
                <div className="flex items-center justify-between">
                  <label className="block text-xs font-semibold text-slate-700">Attach Verified KYC Proof Documents</label>
                  <span className="text-[11px] text-teal-700 font-semibold">{selectedDocIDs.length} Selected</span>
                </div>

                {documents.length === 0 ? (
                  <div className="p-3 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-500">
                    No documents currently in your KYC vault. Please upload a supporting document below first.
                  </div>
                ) : (
                  <div className="space-y-1.5 max-h-36 overflow-y-auto p-2 bg-slate-50 rounded-lg border border-slate-200">
                    {documents.map((doc) => (
                      <label
                        key={doc.id}
                        className="flex items-center space-x-2.5 p-2 rounded bg-white border border-slate-200 hover:bg-teal-50/50 cursor-pointer text-xs"
                      >
                        <input
                          type="checkbox"
                          checked={selectedDocIDs.includes(doc.id)}
                          onChange={() => handleToggleDocSelection(doc.id)}
                          className="rounded text-teal-700 focus:ring-teal-600 w-4 h-4"
                        />
                        <div className="flex-1 truncate">
                          <span className="font-semibold text-slate-800">{doc.document_name}</span>
                          <span className="text-[10px] text-slate-500 ml-2 font-mono">[{doc.document_type}]</span>
                          <p className="text-[10px] text-slate-400 font-mono truncate">SHA-256: {doc.sha256_hash}</p>
                        </div>
                      </label>
                    ))}
                  </div>
                )}

                {/* Inline Upload within Modal */}
                <div className="p-2.5 bg-teal-50/60 border border-teal-200/80 rounded-lg space-y-2">
                  <span className="text-[11px] font-bold text-teal-900 block">Or upload a new proof file right now:</span>
                  <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
                    <select
                      value={docType}
                      onChange={(e) => setDocType(e.target.value)}
                      className="text-xs p-1.5 rounded border border-teal-300 bg-white text-slate-800"
                    >
                      <option value="PASSPORT">Passport</option>
                      <option value="NATIONAL_ID">National ID / Aadhaar</option>
                      <option value="BIRTH_CERTIFICATE">Birth Certificate</option>
                      <option value="DRIVING_LICENSE">Driving License</option>
                      <option value="UTILITY_BILL">Utility Bill (Address Proof)</option>
                      <option value="MARRIAGE_CERTIFICATE">Marriage Certificate</option>
                      <option value="GAZETTE_NOTIFICATION">Gazette Notification</option>
                      <option value="OTHER">Other Proof Document</option>
                    </select>
                    <input
                      type="file"
                      accept=".pdf,image/png,image/jpeg,image/webp"
                      onChange={handleFileUpload}
                      disabled={uploadingDoc}
                      className="text-xs text-slate-600 file:mr-2 file:py-1 file:px-2.5 file:rounded file:border-0 file:text-[11px] file:font-semibold file:bg-teal-700 file:text-white hover:file:bg-teal-800 cursor-pointer"
                    />
                    {uploadingDoc && (
                      <span className="text-xs text-teal-700 font-semibold flex items-center space-x-1">
                        <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        <span>Hashing...</span>
                      </span>
                    )}
                  </div>
                </div>
              </div>

              <div className="pt-3 border-t border-slate-100 flex items-center justify-end space-x-3">
                <button
                  type="button"
                  onClick={() => setShowAmendmentModal(false)}
                  className="px-4 py-2 rounded-lg text-slate-600 hover:bg-slate-100 text-xs font-semibold transition"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={submittingAmendment}
                  className="px-5 py-2 bg-teal-700 hover:bg-teal-800 disabled:opacity-50 text-white text-xs font-bold rounded-lg shadow-sm transition flex items-center space-x-1.5"
                >
                  {submittingAmendment && <RefreshCw className="w-3.5 h-3.5 animate-spin" />}
                  <span>Submit for Verification</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}