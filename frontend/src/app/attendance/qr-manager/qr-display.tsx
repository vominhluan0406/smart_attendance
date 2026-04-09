"use client";

import { useState, useEffect } from "react";
import { QrCode, Building, Maximize, Minimize, RefreshCw, AlertTriangle } from "lucide-react";
import type { Branch } from "@/lib/types";
import { apiGet } from "@/lib/api";

interface QRDisplayProps {
  branches: Branch[];
  defaultBranchId?: string;
}

export default function QRDisplay({ branches, defaultBranchId }: QRDisplayProps) {
  const [selectedBranch, setSelectedBranch] = useState<string>(
    defaultBranchId || (branches.length > 0 ? branches[0].id : "")
  );
  const [isFullScreen, setIsFullScreen] = useState(false);
  
  const [totpCode, setTotpCode] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const [refreshKey, setRefreshKey] = useState(Date.now());
  const [timeLeft, setTimeLeft] = useState(30);

  // Sync with standard TOTP 30-second windows and refetch
  useEffect(() => {
    if (!selectedBranch) return;

    const fetchCode = async () => {
      setLoading(true);
      setError(null);
      try {
        const res = await apiGet<{ code: string }>(`/api/attendance/qr/${selectedBranch}/code`);
        if (res.success && res.data) {
          setTotpCode(res.data.code);
          // Update refresh key to re-mount the image and force a fresh fetch from the browser
          setRefreshKey(Date.now());
        } else {
          setError(res.error?.message || "Không thể lấy mã QR");
          setTotpCode(null);
        }
      } catch (err) {
        setError("Lỗi kết nối đến máy chủ");
        setTotpCode(null);
      } finally {
        setLoading(false);
      }
    };

    fetchCode();

    const interval = setInterval(() => {
      const currentSeconds = new Date().getSeconds();
      const remaining = 30 - (currentSeconds % 30);
      setTimeLeft(remaining);

      // Refresh exactly at the window flip
      if (remaining === 30) {
        fetchCode();
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [selectedBranch]);

  const toggleFullScreen = () => {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen().catch(err => {
        console.error("Error attempting to enable full-screen mode:", err.message);
      });
      setIsFullScreen(true);
    } else {
      if (document.exitFullscreen) {
        document.exitFullscreen();
        setIsFullScreen(false);
      }
    }
  };

  if (branches.length === 0) {
    return (
      <div className="bg-white rounded-3xl shadow p-12 text-center max-w-lg mx-auto">
        <AlertTriangle className="w-16 h-16 text-yellow-500 mx-auto mb-4" />
        <h2 className="text-xl font-bold text-gray-900 mb-2">Không có chi nhánh</h2>
        <p className="text-gray-500">
          Tài khoản của bạn không được phân quyền truy cập vào bất kỳ chi nhánh nào hỗ trợ quét QR.
        </p>
      </div>
    );
  }

  const selectedBranchName = branches.find(b => b.id === selectedBranch)?.name || "";

  return (
    <div className={`transition-all duration-300 ${isFullScreen ? 'min-h-screen fixed inset-0 z-50 bg-gray-900 flex flex-col p-8' : ''}`}>
      
      {/* Header Controls */}
      <div className={`flex flex-col sm:flex-row items-center justify-between gap-4 mb-8 ${isFullScreen ? 'w-full max-w-4xl mx-auto' : ''}`}>
        <div className="flex-1 w-full relative">
          <Building className={`absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 ${isFullScreen ? 'text-gray-400' : 'text-gray-500'}`} />
          <select
            value={selectedBranch}
            onChange={(e) => setSelectedBranch(e.target.value)}
            className={`w-full ${isFullScreen ? 'bg-gray-800 border-gray-700 text-white focus:ring-blue-500' : 'bg-white border-gray-200 text-gray-900 focus:ring-blue-100'} appearance-none border rounded-2xl pl-12 pr-10 py-3 font-bold focus:outline-none focus:ring-4 transition-all`}
            disabled={branches.length <= 1}
          >
            {branches.map(b => (
              <option key={b.id} value={b.id}>{b.name}</option>
            ))}
          </select>
        </div>

        <button
          onClick={toggleFullScreen}
          className={`flex-shrink-0 flex items-center justify-center gap-2 px-6 py-3 rounded-2xl font-bold transition-all ${
            isFullScreen 
              ? 'bg-gray-800 text-white hover:bg-gray-700' 
              : 'bg-white text-gray-700 shadow-sm border border-gray-100 hover:bg-gray-50'
          }`}
        >
          {isFullScreen ? (
            <><Minimize className="w-5 h-5" /> Thu nhỏ</>
          ) : (
            <><Maximize className="w-5 h-5" /> Toàn màn hình</>
          )}
        </button>
      </div>

      {/* Main Display Matrix */}
      <div className={`flex-1 flex flex-col items-center justify-center ${isFullScreen ? '' : ''}`}>
        <div className={`relative ${isFullScreen ? 'bg-gray-800 border-gray-700 shadow-2xl' : 'bg-white border-gray-100 shadow-xl'} border rounded-[2.5rem] p-8 md:p-12 w-full max-w-2xl text-center overflow-hidden transition-colors duration-300`}>
          
          {/* Header Title */}
          <div className="mb-8">
            <h1 className={`text-2xl md:text-3xl font-extrabold mb-2 tracking-tight ${isFullScreen ? 'text-white' : 'text-gray-900'}`}>
              Check-in Chấm Công
            </h1>
            <p className={`text-sm md:text-base font-medium ${isFullScreen ? 'text-gray-400' : 'text-gray-500'}`}>
              {selectedBranchName}
            </p>
          </div>

          {/* QR Code Container */}
          <div className="relative inline-block mb-10 group">
            {/* Pulsing rings for visual flair */}
            <div className={`absolute inset-0 rounded-3xl ${isFullScreen ? 'bg-blue-500/10' : 'bg-blue-50/50'} scale-110 -z-10`} />
            <div className={`absolute inset-0 rounded-3xl ${isFullScreen ? 'bg-blue-500/20' : 'bg-blue-100/50'} scale-105 -z-10 animate-pulse`} />
            
            <div className="bg-white p-4 rounded-3xl shadow-lg relative overflow-hidden">
              {error ? (
                <div className="w-64 h-64 flex flex-col items-center justify-center bg-gray-50 rounded-2xl">
                  <AlertTriangle className="w-12 h-12 text-red-400 mb-3" />
                  <p className="text-red-600 font-bold text-center px-4">{error}</p>
                </div>
              ) : (
                <>
                  {loading && !totpCode && (
                    <div className="absolute inset-0 bg-white/80 backdrop-blur-sm z-10 flex items-center justify-center">
                      <RefreshCw className="w-8 h-8 text-blue-500 animate-spin" />
                    </div>
                  )}
                  <img 
                    src={`${process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"}/api/attendance/qr/${selectedBranch}/image?t=${refreshKey}`} 
                    crossOrigin="use-credentials"
                    alt="Attendance QR Code"
                    className="w-64 h-64 md:w-80 md:h-80 object-contain rounded-xl"
                  />
                </>
              )}
            </div>
          </div>

          {/* Code & Timer UI */}
          <div className={`max-w-xs mx-auto p-6 rounded-2xl ${isFullScreen ? 'bg-gray-900 border-gray-700' : 'bg-gray-50'} border`}>
            <div className="flex items-center justify-between mb-4">
              <span className={`text-xs font-bold uppercase tracking-wider ${isFullScreen ? 'text-gray-400' : 'text-gray-500'}`}>
                Mã 6 Số Tạm Thời
              </span>
              <span className={`flex items-center gap-1 text-xs font-bold ${timeLeft <= 5 ? 'text-red-500 animate-pulse' : (isFullScreen ? 'text-blue-400' : 'text-blue-600')}`}>
                <Clock className="w-3.5 h-3.5" />
                {timeLeft}s
              </span>
            </div>
            
            <div className={`text-4xl font-black tracking-[0.25em] font-mono tabular-nums ${isFullScreen ? 'text-white' : 'text-gray-900'}`}>
              {totpCode ? (
                <span className="flex justify-center">
                  {totpCode.slice(0, 3)}
                  <span className={isFullScreen ? 'text-gray-600' : 'text-gray-300'}>-</span>
                  {totpCode.slice(3, 6)}
                </span>
              ) : (
                <span className={isFullScreen ? 'text-gray-700' : 'text-gray-300'}>------</span>
              )}
            </div>
          </div>
          
          {/* Progress Bar Layer */}
          <div className="absolute bottom-0 left-0 right-0 h-1.5 bg-gray-100">
            <div 
              className={`h-full transition-all ease-linear ${timeLeft <= 5 ? 'bg-red-500' : 'bg-blue-500'}`} 
              style={{ width: `${(timeLeft / 30) * 100}%`, transitionDuration: '1000ms' }}
            />
          </div>
        </div>
      </div>
      
    </div>
  );
}

// Custom Clock icon replacing the internal one to ensure we don't duplicate imports if imported above
function Clock({ className }: { className?: string }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className={className}>
      <circle cx="12" cy="12" r="10"/>
      <polyline points="12 6 12 12 16 14"/>
    </svg>
  );
}
