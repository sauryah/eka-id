'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { Shield, User, QrCode, Building, Lock, LogOut } from 'lucide-react';

export default function Navbar() {
  const [token, setToken] = useState<string | null>(null);

  useEffect(() => {
    const saved = localStorage.getItem('eka_token');
    setToken(saved);
  }, []);

  const handleLogout = () => {
    localStorage.removeItem('eka_token');
    localStorage.removeItem('eka_user');
    setToken(null);
    window.location.href = '/';
  };

  return (
    <header className="sticky top-0 z-50 border-b border-slate-200 bg-white/95 backdrop-blur-sm">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        <Link href="/" className="flex items-center space-x-2.5">
          <div className="w-9 h-9 rounded-lg bg-teal-700 flex items-center justify-center text-white shadow-sm font-bold tracking-wider">
            <Shield className="w-5 h-5" />
          </div>
          <div>
            <span className="font-extrabold text-xl tracking-tight text-slate-900">EKA<span className="text-teal-700"> ID</span></span>
            <span className="hidden sm:inline-block ml-2 text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600 border border-slate-200">Platform</span>
          </div>
        </Link>

        <nav className="flex items-center space-x-1 sm:space-x-4">
          <Link href="/verify" className="flex items-center space-x-1.5 px-3 py-2 text-sm font-medium text-slate-700 hover:text-teal-700 hover:bg-slate-50 rounded-md transition">
            <QrCode className="w-4 h-4 text-slate-500" />
            <span>Verify ID</span>
          </Link>
          <Link href="/org" className="flex items-center space-x-1.5 px-3 py-2 text-sm font-medium text-slate-700 hover:text-teal-700 hover:bg-slate-50 rounded-md transition">
            <Building className="w-4 h-4 text-slate-500" />
            <span>Organizations</span>
          </Link>
          <Link href="/admin" className="hidden md:flex items-center space-x-1.5 px-3 py-2 text-sm font-medium text-slate-700 hover:text-teal-700 hover:bg-slate-50 rounded-md transition">
            <Lock className="w-4 h-4 text-slate-500" />
            <span>Admin</span>
          </Link>

          {token ? (
            <div className="flex items-center space-x-2 border-l border-slate-200 pl-3">
              <Link href="/dashboard" className="flex items-center space-x-1.5 bg-teal-700 hover:bg-teal-800 text-white text-sm font-medium px-3.5 py-1.5 rounded-lg shadow-sm transition">
                <User className="w-4 h-4" />
                <span>Dashboard</span>
              </Link>
              <button onClick={handleLogout} title="Sign Out" className="p-2 text-slate-500 hover:text-rose-600 hover:bg-rose-50 rounded-md transition">
                <LogOut className="w-4 h-4" />
              </button>
            </div>
          ) : (
            <div className="flex items-center space-x-2 border-l border-slate-200 pl-3">
              <Link href="/login" className="text-sm font-medium text-slate-700 hover:text-slate-900 px-3 py-1.5 rounded-md hover:bg-slate-50 transition">
                Sign In
              </Link>
              <Link href="/register" className="bg-teal-700 hover:bg-teal-800 text-white text-sm font-medium px-3.5 py-1.5 rounded-lg shadow-sm transition">
                Create EKA ID
              </Link>
            </div>
          )}
        </nav>
      </div>
    </header>
  );
}