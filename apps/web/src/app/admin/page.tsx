'use client';

import React, { useEffect, useState } from 'react';
import {
  ShieldAlert, Users, Search, AlertTriangle, FileText, CheckCircle2,
  XCircle, RefreshCw, Lock, Shield, UserX
} from 'lucide-react';
import {
  adminListIdentities, adminUpdateStatus, adminListDuplicates,
  adminResolveDuplicate, adminListAudit, Identity, AuditEvent
} from '@/lib/api';

export default function AdminPage() {
  const [activeTab, setActiveTab] = useState<'identities' | 'duplicates' | 'audit'>('identities');
  const [loading, setLoading] = useState(true);
  const [identities, setIdentities] = useState<Identity[]>([]);
  const [duplicates, setDuplicates] = useState<any[]>([]);
  const [auditEvents, setAuditEvents] = useState<AuditEvent[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAdminData();
  }, []);

  const loadAdminData = async () => {
    const token = localStorage.getItem('eka_token') || 'dev_admin_token';
    setLoading(true);
    setError(null);
    try {
      const [idRes, dupRes, audRes] = await Promise.allSettled([
        adminListIdentities(token),
        adminListDuplicates(token),
        adminListAudit(token),
      ]);

      if (idRes.status === 'fulfilled') setIdentities(idRes.value.identities || []);
      if (dupRes.status === 'fulfilled') setDuplicates(dupRes.value || []);
      if (audRes.status === 'fulfilled') setAuditEvents(audRes.value.events || []);
    } catch (err: any) {
      setError(err.message || 'Failed to load administrative console data.');
    } finally {
      setLoading(false);
    }
  };

  const handleStatusChange = async (identityId: string, newStatus: string) => {
    const token = localStorage.getItem('eka_token') || 'dev_admin_token';
    try {
      await adminUpdateStatus(token, identityId, newStatus);
      setActionSuccess(`Identity status updated to ${newStatus}`);
      loadAdminData();
      setTimeout(() => setActionSuccess(null), 3000);
    } catch (err: any) {
      setError(err.message || 'Failed to change status');
    }
  };

  const handleResolveDuplicate = async (flagId: string, status: string) => {
    const token = localStorage.getItem('eka_token') || 'dev_admin_token';
    try {
      await adminResolveDuplicate(token, flagId, status);
      setActionSuccess(`Duplicate conflict resolved: ${status}`);
      loadAdminData();
      setTimeout(() => setActionSuccess(null), 3000);
    } catch (err: any) {
      setError(err.message || 'Failed to resolve duplicate flag');
    }
  };

  const handleSuspendAndResolve = async (flagId: string, suspectedId: string) => {
    const token = localStorage.getItem('eka_token') || 'dev_admin_token';
    try {
      if (suspectedId) {
        await adminUpdateStatus(token, suspectedId, 'SUSPENDED');
      }
      await adminResolveDuplicate(token, flagId, 'RESOLVED_DUPLICATE');
      setActionSuccess('Suspected duplicate suspended and conflict confirmed.');
      loadAdminData();
      setTimeout(() => setActionSuccess(null), 4000);
    } catch (err: any) {
      setError(err.message || 'Failed to suspend duplicate identity');
    }
  };

  const filteredIdentities = identities.filter((id) =>
    id.eka_id?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-8">
        <div>
          <div className="inline-flex items-center space-x-2 px-3 py-1 rounded-full bg-slate-900 text-teal-400 text-xs font-semibold uppercase tracking-wider mb-2">
            <Lock className="w-3.5 h-3.5" />
            <span>Elevated Administration</span>
          </div>
          <h1 className="text-2xl sm:text-3xl font-extrabold text-slate-900 tracking-tight">
            System Admin Console
          </h1>
          <p className="text-xs text-slate-500 mt-1">
            Global identity governance, duplicate heuristics review, and immutable audit logs.
          </p>
        </div>

        <button
          onClick={loadAdminData}
          className="flex items-center space-x-2 px-3.5 py-2 bg-white border border-slate-300 hover:bg-slate-50 rounded-lg text-xs font-semibold text-slate-700 shadow-sm transition"
        >
          <RefreshCw className="w-3.5 h-3.5" />
          <span>Refresh Records</span>
        </button>
      </div>

      {actionSuccess && (
        <div className="mb-6 p-3.5 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-xs font-semibold flex items-center space-x-2">
          <CheckCircle2 className="w-4 h-4" />
          <span>{actionSuccess}</span>
        </div>
      )}

      {error && (
        <div className="mb-6 p-3.5 rounded-xl bg-rose-50 border border-rose-200 text-rose-800 text-xs">
          {error}
        </div>
      )}

      {/* Metrics Row */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mb-8">
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <p className="text-[10px] text-slate-400 uppercase font-bold tracking-wider">Total Identities</p>
          <p className="text-2xl font-extrabold text-slate-900 mt-1">{identities.length}</p>
        </div>
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <p className="text-[10px] text-slate-400 uppercase font-bold tracking-wider">Duplicate Suspects</p>
          <p className="text-2xl font-extrabold text-amber-600 mt-1">{duplicates.length}</p>
        </div>
        <div className="bg-white p-5 rounded-2xl border border-slate-200 shadow-sm">
          <p className="text-[10px] text-slate-400 uppercase font-bold tracking-wider">Audit Log Records</p>
          <p className="text-2xl font-extrabold text-teal-700 mt-1">{auditEvents.length}</p>
        </div>
      </div>

      {/* Navigation Tabs */}
      <div className="flex space-x-2 border-b border-slate-200 pb-2 mb-6">
        {[
          { id: 'identities', label: `Identities (${identities.length})`, icon: Users },
          { id: 'duplicates', label: `Duplicate Queue (${duplicates.length})`, icon: AlertTriangle },
          { id: 'audit', label: `Audit Log (${auditEvents.length})`, icon: FileText },
        ].map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as any)}
              className={`flex items-center space-x-2 px-4 py-2 rounded-lg text-xs font-semibold transition ${
                isActive
                  ? 'bg-slate-900 text-white shadow-sm'
                  : 'text-slate-600 hover:text-slate-900 hover:bg-slate-100'
              }`}
            >
              <Icon className="w-4 h-4" />
              <span>{tab.label}</span>
            </button>
          );
        })}
      </div>

      {/* TAB 1: Identities List */}
      {activeTab === 'identities' && (
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          <div className="p-4 border-b border-slate-100 flex items-center justify-between">
            <div className="relative w-72">
              <Search className="w-4 h-4 text-slate-400 absolute left-3 top-2.5" />
              <input
                type="text"
                placeholder="Search EKA ID..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="w-full pl-9 pr-3 py-1.5 text-xs border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-slate-900"
              />
            </div>
            <span className="text-xs text-slate-400">Showing {filteredIdentities.length} records</span>
          </div>

          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs text-slate-600">
              <thead className="bg-slate-50 text-[10px] text-slate-400 uppercase tracking-wider border-b border-slate-100">
                <tr>
                  <th className="py-3 px-4 font-bold">Public EKA ID</th>
                  <th className="py-3 px-4 font-bold">Status</th>
                  <th className="py-3 px-4 font-bold">Verification Tier</th>
                  <th className="py-3 px-4 font-bold">Created At</th>
                  <th className="py-3 px-4 font-bold text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {filteredIdentities.map((id) => (
                  <tr key={id.id} className="hover:bg-slate-50/80 transition">
                    <td className="py-3.5 px-4 font-mono font-bold text-slate-900">{id.eka_id}</td>
                    <td className="py-3.5 px-4">
                      <span
                        className={`px-2 py-0.5 rounded-full font-semibold text-[10px] ${
                          id.status === 'ACTIVE'
                            ? 'bg-emerald-100 text-emerald-800'
                            : 'bg-rose-100 text-rose-800'
                        }`}
                      >
                        {id.status}
                      </span>
                    </td>
                    <td className="py-3.5 px-4">{id.verification_level}</td>
                    <td className="py-3.5 px-4">{new Date(id.created_at).toLocaleDateString()}</td>
                    <td className="py-3.5 px-4 text-right space-x-2">
                      {id.status === 'ACTIVE' ? (
                        <button
                          onClick={() => handleStatusChange(id.id, 'SUSPENDED')}
                          className="px-2.5 py-1 bg-rose-50 hover:bg-rose-100 text-rose-700 font-semibold rounded text-[11px] transition"
                        >
                          Suspend
                        </button>
                      ) : (
                        <button
                          onClick={() => handleStatusChange(id.id, 'ACTIVE')}
                          className="px-2.5 py-1 bg-emerald-50 hover:bg-emerald-100 text-emerald-700 font-semibold rounded text-[11px] transition"
                        >
                          Restore
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* TAB 2: Duplicate Queue */}
      {activeTab === 'duplicates' && (
        <div className="space-y-6">
          {duplicates.length === 0 ? (
            <div className="bg-white p-12 rounded-2xl border border-slate-200 text-center text-slate-400 text-sm">
              <CheckCircle2 className="w-8 h-8 text-emerald-500 mx-auto mb-2" />
              No duplicate identity conflicts flagged.
            </div>
          ) : (
            duplicates.map((dup) => {
              const isBiometric = dup.match_reasons?.some((r: string) => r.includes('Biometric') || r.includes('Face'));
              return (
                <div
                  key={dup.id}
                  className={`bg-white p-6 rounded-2xl border shadow-sm space-y-5 ${
                    isBiometric ? 'border-purple-300 bg-gradient-to-br from-white to-purple-50/30' : 'border-amber-200'
                  }`}
                >
                  <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 pb-3">
                    <div className="flex items-center space-x-2">
                      {isBiometric ? (
                        <span className="flex items-center space-x-1.5 px-3 py-1 rounded-full bg-purple-100 border border-purple-200 text-purple-800 text-xs font-bold uppercase tracking-wider">
                          <span>🧬 Biometric Facial Match Flagged</span>
                        </span>
                      ) : (
                        <span className="flex items-center space-x-1.5 px-3 py-1 rounded-full bg-amber-100 border border-amber-200 text-amber-800 text-xs font-bold uppercase tracking-wider">
                          <AlertTriangle className="w-3.5 h-3.5" />
                          <span>Demographic Duplicate Flagged</span>
                        </span>
                      )}
                    </div>
                    <span
                      className={`px-3 py-1 rounded-lg text-xs font-black tracking-wide ${
                        dup.confidence_score >= 85
                          ? 'bg-rose-100 text-rose-800'
                          : 'bg-amber-100 text-amber-800'
                      }`}
                    >
                      Match Confidence: {dup.confidence_score}%
                    </span>
                  </div>

                  {/* Side-by-side identity comparison */}
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {/* Primary Identity */}
                    <div className="p-4 rounded-xl bg-slate-50 border border-slate-200 flex items-center space-x-4">
                      {dup.primary_photo ? (
                        <img
                          src={dup.primary_photo}
                          alt="Primary"
                          className="w-16 h-16 rounded-xl object-cover border-2 border-teal-600 shadow-sm flex-shrink-0"
                        />
                      ) : (
                        <div className="w-16 h-16 rounded-xl bg-teal-800 text-white font-bold text-xl flex items-center justify-center flex-shrink-0">
                          {dup.primary_name?.charAt(0) || 'U'}
                        </div>
                      )}
                      <div className="min-w-0">
                        <span className="text-[10px] uppercase font-bold text-teal-700 tracking-wider">
                          Original Identity
                        </span>
                        <p className="font-bold text-slate-900 text-sm truncate">
                          {dup.primary_name || 'Primary User'}
                        </p>
                        <p className="font-mono text-xs text-slate-500 truncate">
                          {dup.primary_eka_id || dup.identity_id}
                        </p>
                      </div>
                    </div>

                    {/* Suspected Duplicate */}
                    <div className="p-4 rounded-xl bg-rose-50/60 border border-rose-200 flex items-center space-x-4">
                      {dup.suspected_photo ? (
                        <img
                          src={dup.suspected_photo}
                          alt="Suspect"
                          className="w-16 h-16 rounded-xl object-cover border-2 border-rose-500 shadow-sm flex-shrink-0"
                        />
                      ) : (
                        <div className="w-16 h-16 rounded-xl bg-rose-800 text-white font-bold text-xl flex items-center justify-center flex-shrink-0">
                          {dup.suspected_name?.charAt(0) || 'S'}
                        </div>
                      )}
                      <div className="min-w-0">
                        <span className="text-[10px] uppercase font-bold text-rose-700 tracking-wider">
                          Suspected Duplicate / Clone
                        </span>
                        <p className="font-bold text-slate-900 text-sm truncate">
                          {dup.suspected_name || 'Suspect User'}
                        </p>
                        <p className="font-mono text-xs text-slate-500 truncate">
                          {dup.suspected_eka_id || dup.suspected_duplicate_id}
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Match Reasons */}
                  <div className="space-y-1.5 pt-1">
                    <p className="text-xs font-semibold text-slate-700">Detection Match Signals:</p>
                    <div className="flex flex-wrap gap-1.5">
                      {dup.match_reasons?.map((reason: string, idx: number) => (
                        <span
                          key={idx}
                          className={`text-xs px-2.5 py-1 rounded-md font-medium ${
                            reason.includes('Biometric')
                              ? 'bg-purple-100 text-purple-900 font-semibold'
                              : 'bg-slate-100 text-slate-700'
                          }`}
                        >
                          {reason}
                        </span>
                      ))}
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center space-x-3 pt-3 border-t border-slate-100">
                    <button
                      onClick={() => handleResolveDuplicate(dup.id, 'RESOLVED_FALSE_POSITIVE')}
                      className="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-700 font-semibold text-xs rounded-lg transition"
                    >
                      Mark False Positive
                    </button>
                    <button
                      onClick={() => handleSuspendAndResolve(dup.id, dup.suspected_duplicate_id)}
                      className="px-4 py-2 bg-rose-600 hover:bg-rose-700 text-white font-semibold text-xs rounded-lg shadow-sm transition flex items-center space-x-1.5"
                    >
                      <UserX className="w-3.5 h-3.5" />
                      <span>Confirm & Suspend Clone</span>
                    </button>
                  </div>
                </div>
              );
            })
          )}
        </div>
      )}

      {/* TAB 3: Audit Log */}
      {activeTab === 'audit' && (
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs text-slate-600">
              <thead className="bg-slate-50 text-[10px] text-slate-400 uppercase tracking-wider border-b border-slate-100">
                <tr>
                  <th className="py-3 px-4 font-bold">Timestamp</th>
                  <th className="py-3 px-4 font-bold">Actor Type</th>
                  <th className="py-3 px-4 font-bold">Action</th>
                  <th className="py-3 px-4 font-bold">Resource</th>
                  <th className="py-3 px-4 font-bold">Result</th>
                  <th className="py-3 px-4 font-bold">Client IP</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 font-mono">
                {auditEvents.map((ev) => (
                  <tr key={ev.event_id} className="hover:bg-slate-50/80 transition text-[11px]">
                    <td className="py-3 px-4 text-slate-500">{new Date(ev.created_at).toLocaleString()}</td>
                    <td className="py-3 px-4 font-bold text-slate-800">{ev.actor_type}</td>
                    <td className="py-3 px-4 text-teal-800 font-bold">{ev.action}</td>
                    <td className="py-3 px-4 text-slate-600">{ev.resource_type}: {ev.resource_id}</td>
                    <td className="py-3 px-4">
                      <span className={`px-2 py-0.5 rounded font-semibold text-[10px] ${
                        ev.result === 'SUCCESS' ? 'bg-emerald-100 text-emerald-800' : 'bg-rose-100 text-rose-800'
                      }`}>
                        {ev.result}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-slate-400">{ev.ip_address || '127.0.0.1'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}