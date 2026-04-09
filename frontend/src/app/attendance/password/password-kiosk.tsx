"use client";

import { useState, useRef, useEffect } from "react";
import { KeyRound, Maximize, Minimize, CheckCircle, AlertCircle, Play, Fingerprint, ShieldCheck } from "lucide-react";

export default function PasswordKiosk({ branchName }: { branchName: string }) {
  const [isFullscreen, setIsFullscreen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);

  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };
    document.addEventListener("fullscreenchange", handleFullscreenChange);
    return () => document.removeEventListener("fullscreenchange", handleFullscreenChange);
  }, []);

  const toggleFullscreen = () => {
    if (!document.fullscreenElement) {
      containerRef.current?.requestFullscreen().catch(err => {
        console.error("Error attempting to enable fullscreen:", err);
      });
    } else {
      document.exitFullscreen();
    }
  };

  const handleKioskLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setResult(null);

    const baseUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

    try {
      // 1. Authenticate Employee (fetch ephemeral token without writing to document.cookie)
      const authRes = await fetch(`${baseUrl}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });

      const authBody = await authRes.json();
      if (!authRes.ok || !authBody.success) {
        setResult({
          success: false,
          message: authBody.error?.message || "Sai email hoặc mật khẩu"
        });
        setLoading(false);
        return;
      }

      const ephemeralToken = authBody.data?.access_token;
      if (!ephemeralToken) {
        setResult({ success: false, message: "Lỗi kết nối: Không lấy được token" });
        setLoading(false);
        return;
      }

      // 2. Submit Attendance Log as the Employee using the ephemeral token
      const logRes = await fetch(`${baseUrl}/api/attendance/log`, {
        method: "POST",
        headers: { 
          "Content-Type": "application/json",
          "Authorization": `Bearer ${ephemeralToken}`
        },
        body: JSON.stringify({
          password_verified: true // Backend validates if branch allows MethodPassword
        }),
      });

      const logBody = await logRes.json();
      if (logBody.success) {
        setResult({ success: true, message: "Chấm công thành công!" });
        setEmail("");
        setPassword("");
        
        // Auto clear success message to keep queue moving
        setTimeout(() => setResult(null), 3000);
      } else {
        setResult({
          success: false,
          message: logBody.error?.message || "Từ chối chấm công"
        });
      }

    } catch (error) {
      console.error(error);
      setResult({ success: false, message: "Lỗi kết nối máy chủ" });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div 
      ref={containerRef} 
      className={`bg-white rounded-3xl shadow-2xl overflow-hidden flex flex-col md:flex-row transition-all ${
        isFullscreen ? "h-screen w-screen rounded-none" : "min-h-[600px]"
      }`}
    >
      {/* Left Identity Pane */}
      <div className="bg-primary-600 md:w-5/12 p-10 flex flex-col justify-between text-white relative overflow-hidden">
        <div className="relative z-10 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <ShieldCheck className="w-8 h-8" />
            <h1 className="text-xl font-bold font-mono tracking-wider">Smart Kiosk</h1>
          </div>
          <button 
            onClick={toggleFullscreen}
            className="p-3 bg-white/10 hover:bg-white/20 rounded-xl transition-all"
            title="Toàn màn hình"
          >
            {isFullscreen ? <Minimize className="w-6 h-6 text-white" /> : <Maximize className="w-6 h-6 text-white" />}
          </button>
        </div>

        <div className="relative z-10 mt-12 gap-8 flex flex-col items-start h-full justify-center">
            <div className="bg-white/10 p-4 rounded-2xl backdrop-blur-sm border border-white/20">
              <KeyRound className="w-12 h-12 text-blue-100" />
            </div>
            <div>
              <h2 className="text-4xl lg:text-5xl font-extrabold mb-4 leading-tight">
                Chấm công <br />
                Bảo mật
              </h2>
              <p className="text-primary-100 text-lg opacity-90 leading-relaxed font-medium">
                Vui lòng nhập Email và Mật khẩu nhân viên để thực hiện điểm danh tại quầy.
              </p>
            </div>

            <div className="mt-8 pt-8 border-t border-white/20 w-full flex items-center gap-4">
              <span className="w-3 h-3 bg-green-400 rounded-full animate-pulse"></span>
              <span className="text-white font-mono font-bold">{branchName}</span>
            </div>
        </div>

        {/* Decorative background shapes */}
        <div className="absolute top-0 right-0 -mr-20 -mt-20 w-64 h-64 bg-primary-500 rounded-full blur-3xl opacity-50 pointer-events-none"></div>
        <div className="absolute bottom-0 left-0 -ml-20 -mb-20 w-80 h-80 bg-primary-700 rounded-full blur-3xl opacity-50 pointer-events-none"></div>
      </div>

      {/* Right Login Pane */}
      <div className="flex-1 bg-white p-8 md:p-16 flex items-center justify-center relative">
        <div className="w-full max-w-md">
          {result && result.success ? (
            <div className="text-center animate-in fade-in zoom-in duration-300 py-16">
              <div className="w-24 h-24 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-6 shadow-inner">
                <CheckCircle className="w-12 h-12 text-green-600" />
              </div>
              <h2 className="text-3xl font-extrabold text-gray-900 mb-2">Thành công!</h2>
              <p className="text-gray-500 text-lg font-medium">{result.message}</p>
              <div className="w-full h-1 bg-gray-100 mt-12 rounded-full overflow-hidden">
                <div className="h-full bg-green-500 animate-[progress_3s_ease-in-out_forwards]"></div>
              </div>
              <p className="text-gray-400 text-sm mt-4">Sẵn sàng cho nhân viên tiếp theo...</p>
            </div>
          ) : (
            <form onSubmit={handleKioskLogin} className="space-y-8">
              
              {result && !result.success && (
                <div className="bg-red-50 border-l-4 border-red-500 p-4 rounded-r-xl">
                  <div className="flex">
                    <div className="flex-shrink-0">
                      <AlertCircle className="h-5 w-5 text-red-500" aria-hidden="true" />
                    </div>
                    <div className="ml-3">
                      <p className="text-sm text-red-700 font-medium">
                        {result.message}
                      </p>
                    </div>
                  </div>
                </div>
              )}

              <div>
                <label className="block text-sm font-bold text-gray-700 mb-2 uppercase tracking-wider">
                  Email nhân viên
                </label>
                <input
                  type="email"
                  required
                  autoFocus
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="block w-full text-lg rounded-xl border-gray-200 bg-gray-50 py-4 px-5 text-gray-900 shadow-sm focus:bg-white focus:ring-4 focus:ring-primary-500/20 focus:border-primary-500 transition-all font-medium"
                  placeholder="name@example.com"
                />
              </div>

              <div>
                <label className="block text-sm font-bold text-gray-700 mb-2 uppercase tracking-wider">
                  Mật khẩu
                </label>
                <input
                  type="password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="block w-full text-lg rounded-xl border-gray-200 bg-gray-50 py-4 px-5 text-gray-900 shadow-sm focus:bg-white focus:ring-4 focus:ring-primary-500/20 focus:border-primary-500 transition-all font-medium"
                  placeholder="••••••••"
                />
              </div>

              <button
                type="submit"
                disabled={loading || !email || !password}
                className="w-full flex items-center justify-center gap-2 py-5 px-6 border border-transparent rounded-xl shadow-lg text-lg font-bold text-white bg-primary-600 hover:bg-primary-700 hover:-translate-y-0.5 active:translate-y-0 active:shadow-md transition-all focus:outline-none focus:ring-4 focus:ring-primary-500/30 disabled:opacity-50 disabled:pointer-events-none"
              >
                {loading ? (
                  <span className="spinner mr-2" />
                ) : (
                  <>
                    <Fingerprint className="w-5 h-5" />
                    Xác nhận điểm danh
                  </>
                )}
              </button>
            </form>
          )}
        </div>
      </div>
      <style dangerouslySetInnerHTML={{__html: `
        @keyframes progress {
          from { width: 0%; }
          to { width: 100%; }
        }
      `}}/>
    </div>
  );
}
