"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Camera, MapPin, CheckCircle, AlertCircle, Send, Target, RefreshCw, XCircle } from "lucide-react";
import { Html5QrcodeScanner } from "html5-qrcode";

export default function AttendanceClient() {
  const [totpCode, setTotpCode] = useState("");
  const [lat, setLat] = useState<number | null>(null);
  const [lng, setLng] = useState<number | null>(null);
  const [accuracy, setAccuracy] = useState<number | null>(null);
  const [gpsStatus, setGpsStatus] = useState("Đang xác định vị trí...");
  const [gpsOk, setGpsOk] = useState(false);
  const [deviceFingerprint, setDeviceFingerprint] = useState("");
  const [result, setResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);
  const [loading, setLoading] = useState(false);
  const scannerRef = useRef<Html5QrcodeScanner | null>(null);
  const [isCapturing, setIsCapturing] = useState(false);
  const [proximityStatus, setProximityStatus] = useState<{
    ip_verified: boolean;
    loc_verified: boolean;
    wifi_gps_verified: boolean;
    is_any_passed: boolean;
    validation_errors?: string[];
  } | null>(null);
  const [isProximityChecking, setIsProximityChecking] = useState(false);

  // Generate device fingerprint
  useEffect(() => {
    async function generateFP() {
      const parts = [
        navigator.userAgent,
        `${screen.width}x${screen.height}`,
        String(screen.colorDepth),
        Intl.DateTimeFormat().resolvedOptions().timeZone,
        navigator.language,
        String(navigator.hardwareConcurrency || 0),
        navigator.platform,
      ].join("|");

      if (crypto.subtle) {
        const buf = await crypto.subtle.digest(
          "SHA-256",
          new TextEncoder().encode(parts)
        );
        const hash = Array.from(new Uint8Array(buf))
          .map((b) => b.toString(16).padStart(2, "0"))
          .join("");
        setDeviceFingerprint(hash);
      }
    }
    generateFP();
  }, []);

  // Watch GPS position
  useEffect(() => {
    if (!navigator.geolocation) {
      setGpsStatus("Trình duyệt không hỗ trợ GPS");
      return;
    }

    const watchId = navigator.geolocation.watchPosition(
      (pos) => {
        setLat(pos.coords.latitude);
        setLng(pos.coords.longitude);
        setAccuracy(pos.coords.accuracy);
        setGpsOk(true);
        setGpsStatus(
          `GPS: ${pos.coords.latitude.toFixed(6)}, ${pos.coords.longitude.toFixed(
            6
          )} (+-${Math.round(pos.coords.accuracy)}m)`
        );
      },
      (err) => {
        setGpsStatus(`Lỗi GPS: ${err.message}`);
        setGpsOk(false);
      },
      { enableHighAccuracy: true, timeout: 20000 }
    );

    return () => navigator.geolocation.clearWatch(watchId);
  }, []);

  // Auto-validate proximity
  const checkProximity = useCallback(async () => {
    if (!gpsOk || lat === null || lng === null) return;
    setIsProximityChecking(true);
    try {
      const baseUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
      const res = await fetch(`${baseUrl}/api/attendance/check-proximity`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ 
          lat, 
          lng, 
          accuracy_m: accuracy,
          device_fingerprint: deviceFingerprint
        }),
      });
      const data = await res.json();
      setProximityStatus(data);
    } catch (err) {
      console.error("Proximity check failed", err);
    } finally {
      setIsProximityChecking(false);
    }
  }, [lat, lng, accuracy, gpsOk, deviceFingerprint]);

  useEffect(() => {
    const timer = setTimeout(checkProximity, 2000); // 2s debounce
    return () => clearTimeout(timer);
  }, [checkProximity]);

  const handleSubmit = useCallback(
    async (code?: string, methodOverride?: string) => {
      setLoading(true);
      setResult(null);

      const submitCode = code || totpCode;
      
      // If no code and not a GPS-only override, error out
      if (!submitCode && methodOverride !== "location") {
        setResult({ success: false, message: "Vui lòng quét mã QR hoặc bật GPS" });
        setLoading(false);
        return;
      }

      try {
        const baseUrl =
          process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
        const res = await fetch(`${baseUrl}/api/attendance/log`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({
            totp_code: submitCode || undefined,
            lat: lat,
            lng: lng,
            accuracy: accuracy,
            device_fingerprint: deviceFingerprint,
          }),
        });

        const body = await res.json();
        if (body.success) {
          setResult({ success: true, message: "Chấm công thành công!" });
          setTotpCode("");
          // Clear notification after 5s
          setTimeout(() => setResult(null), 5000);
        } else {
          setResult({
            success: false,
            message: body.error?.message || "Chấm công thất bại",
          });
        }
      } catch {
        setResult({
          success: false,
          message: "Không thể kết nối đến server",
        });
      } finally {
        setLoading(false);
      }
    },
    [totpCode, lat, lng, accuracy, deviceFingerprint]
  );

  // Initialize QR Scanner
  useEffect(() => {
    if (isCapturing && !scannerRef.current) {
      const scanner = new Html5QrcodeScanner(
        "qr-reader",
        { 
          fps: 10, 
          qrbox: { width: 250, height: 250 },
          aspectRatio: 1.0,
          showTorchButtonIfSupported: true,
        },
        /* verbose= */ false
      );

      scanner.render(
        (decodedText) => {
          // Success
          console.log(`Scan result: ${decodedText}`);
          handleSubmit(decodedText);
          setIsCapturing(false);
          scanner.clear();
          scannerRef.current = null;
        },
        (error) => {
          // Silently ignore errors during continuous scanning
        }
      );

      scannerRef.current = scanner;
    }

    return () => {
      if (scannerRef.current) {
        scannerRef.current.clear().catch(err => console.error("Error clearing scanner", err));
        scannerRef.current = null;
      }
    };
  }, [isCapturing, handleSubmit]);

  return (
    <div className="bg-gray-50 pb-20">
      <main className="mx-auto max-w-md px-4 py-6">
        <div className="flex items-center justify-center gap-3 mb-8">
          <div className="p-2 bg-primary-100 rounded-xl">
            <Camera className="w-6 h-6 text-primary-600" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900">Chấm công</h1>
        </div>

        {/* Result Tracking */}
        {result && (
          <div
            className={`mb-6 p-4 rounded-2xl flex items-center gap-3 animate-in slide-in-from-top duration-300 ${
              result.success
                ? "bg-green-50 border border-green-100 shadow-sm"
                : "bg-red-50 border border-red-100 shadow-sm"
            }`}
          >
            {result.success ? (
              <CheckCircle className="w-6 h-6 text-green-600 flex-shrink-0" />
            ) : (
              <AlertCircle className="w-6 h-6 text-red-600 flex-shrink-0" />
            )}
            <p
              className={`text-sm font-bold ${
                result.success ? "text-green-800" : "text-red-800"
              }`}
            >
              {result.message}
            </p>
          </div>
        )}

        {/* Main Action Area */}
        <div className="bg-white shadow-xl rounded-2xl border border-gray-100 overflow-hidden mb-6">
          <div className="p-6">
            {!isCapturing ? (
              <div 
                onClick={() => setIsCapturing(true)}
                className="rounded-xl cursor-pointer overflow-hidden border-2 border-dashed border-primary-200 aspect-square flex flex-col items-center justify-center bg-primary-50 hover:bg-primary-100 transition-colors group"
              >
                <div className="text-center p-6">
                  <Camera className="w-16 h-16 text-primary-400 mx-auto mb-4 group-hover:scale-110 transition-transform" />
                  <p className="text-lg font-bold text-primary-700">Mở Camera Quét QR</p>
                  <p className="text-sm text-primary-500 mt-2">Đưa mã QR chi nhánh vào khung quét</p>
                </div>
              </div>
            ) : (
              <div className="relative">
                <div id="qr-reader" className="rounded-xl overflow-hidden border-2 border-primary-500"></div>
                <button 
                  onClick={() => setIsCapturing(false)}
                  className="mt-4 w-full py-2 text-sm font-bold text-gray-500 hover:text-gray-700"
                >
                  Đóng Camera
                </button>
              </div>
            )}

            {/* Proximity Status Dashboard */}
            <div className="mt-6 space-y-3">
              <div className="p-4 bg-gray-50 rounded-2xl border border-gray-100">
                <div className="flex items-center justify-between mb-3">
                  <span className="text-[10px] font-bold text-gray-400 uppercase tracking-widest">Kiểm tra vùng chấm công</span>
                  <button 
                    onClick={() => checkProximity()}
                    disabled={isProximityChecking || !gpsOk}
                    className="flex items-center gap-1 group outline-none"
                  >
                    <div className={`w-1.5 h-1.5 rounded-full ${isProximityChecking ? "bg-primary-500 animate-pulse" : "bg-gray-300 group-hover:bg-primary-400"}`}></div>
                    <span className="text-[9px] text-gray-400 group-hover:text-primary-600 transition-colors font-bold uppercase tracking-tighter">Cập nhật</span>
                  </button>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  {/* GPS Indicator */}
                  <div className={`p-3 rounded-xl border flex flex-col items-center gap-2 transition-all duration-500 ${
                    proximityStatus?.loc_verified 
                    ? "bg-green-50 border-green-100 text-green-700 shadow-sm shadow-green-100/50" 
                    : proximityStatus 
                      ? "bg-red-50 border-red-100 text-red-700"
                      : "bg-white border-gray-100 text-gray-400"
                  }`}>
                    <Target className={`w-5 h-5 transition-colors ${
                      proximityStatus?.loc_verified ? "text-green-600" : proximityStatus ? "text-red-600" : "text-gray-300"
                    }`} />
                    <span className="text-[10px] font-bold">Vị trí (GPS)</span>
                    {proximityStatus?.loc_verified ? (
                      <CheckCircle className="w-3 h-3 text-green-500 animate-in zoom-in duration-300" />
                    ) : proximityStatus ? (
                      <XCircle className="w-3 h-3 text-red-500 animate-in zoom-in duration-300" />
                    ) : (
                      <div className={`w-3 h-3 rounded-full border-2 ${isProximityChecking ? "border-primary-300 animate-spin border-t-transparent" : "border-gray-200"}`} />
                    )}
                  </div>

                  {/* WiFi/IP Indicator */}
                  <div className={`p-3 rounded-xl border flex flex-col items-center gap-2 transition-all duration-500 ${
                    proximityStatus?.ip_verified 
                    ? "bg-blue-50 border-blue-100 text-blue-700 shadow-sm shadow-blue-100/50" 
                    : proximityStatus 
                      ? "bg-red-50 border-red-100 text-red-700"
                      : "bg-white border-gray-100 text-gray-400"
                  }`}>
                    <RefreshCw className={`w-5 h-5 transition-all ${
                      proximityStatus?.ip_verified ? "text-blue-600" : proximityStatus ? "text-red-600" : "text-gray-300"
                    } ${isProximityChecking ? "animate-spin" : ""}`} />
                    <span className="text-[10px] font-bold">Mạng WiFi</span>
                    {proximityStatus?.ip_verified ? (
                      <CheckCircle className="w-3 h-3 text-blue-500 animate-in zoom-in duration-300" />
                    ) : proximityStatus ? (
                      <XCircle className="w-3 h-3 text-red-500 animate-in zoom-in duration-300" />
                    ) : (
                      <div className={`w-3 h-3 rounded-full border-2 ${isProximityChecking ? "border-primary-300 animate-spin border-t-transparent" : "border-gray-200"}`} />
                    )}
                  </div>
                </div>

                {proximityStatus && !proximityStatus.is_any_passed && (
                  <div className="mt-4 p-3 bg-red-50/50 rounded-xl border border-red-100/50 animate-in fade-in slide-in-from-top-1 duration-300">
                    <p className="text-[10px] font-bold text-red-800 mb-1 flex items-center gap-1">
                      <AlertCircle className="w-3 h-3" />
                      Lý do chưa thể chấm công:
                    </p>
                    <ul className="list-disc list-inside space-y-0.5 ml-1">
                      {(!proximityStatus.validation_errors || proximityStatus.validation_errors.length === 0) && (
                        <li className="text-[9px] text-red-600 font-medium">Bạn chưa ở trong vùng cho phép hoặc sai mạng WiFi chi nhánh.</li>
                      )}
                      {proximityStatus.validation_errors?.map((err, i) => (
                        <li key={i} className="text-[9px] text-red-600 font-medium leading-tight">
                          {err.replace(/:\d+$/, '') /* Strip port */}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}
                
                <p className="mt-3 text-[10px] text-center text-gray-400 font-medium leading-relaxed">
                  {gpsStatus}
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Final Check-in Button */}
        <div className="mb-6">
          <button
            onClick={() => handleSubmit(undefined, "location")}
            disabled={loading || !proximityStatus?.is_any_passed}
            className={`w-full flex flex-col items-center justify-center gap-1 py-4 px-6 rounded-2xl shadow-xl transition-all active:scale-[0.98] ${
              proximityStatus?.is_any_passed
              ? "bg-primary-600 text-white shadow-primary-200 hover:bg-primary-700"
              : "bg-gray-100 text-gray-400 shadow-none grayscale cursor-not-allowed"
            }`}
          >
            {loading ? (
              <RefreshCw className="w-6 h-6 animate-spin" />
            ) : (
              <>
                <span className="text-lg font-bold">Chấm công ngay</span>
                <span className="text-[10px] opacity-80 uppercase tracking-tighter">
                  {proximityStatus?.is_any_passed ? "Đã xác thực vùng an toàn" : "Chờ xác thực vị trí..."}
                </span>
              </>
            )}
          </button>
          <p className="text-center text-[10px] text-gray-400 mt-4 px-6 italic">
            * Hệ thống tự động nhận diện chi nhánh khi bạn ở trong phạm vi cho phép.
          </p>
        </div>
      </main>

      {/* Styles for html5-qrcode to match our theme */}
      <style dangerouslySetInnerHTML={{__html: `
        #qr-reader { border: none !important; }
        #qr-reader-results { display: none; }
        #qr-reader__dashboard { padding: 10px !important; }
        #qr-reader__camera_selection { 
          width: 100%;
          padding: 8px;
          border-radius: 8px;
          border: 1px solid #e2e8f0;
          margin-bottom: 10px;
        }
        #qr-reader__dashboard_section_csr button {
          padding: 8px 16px;
          border-radius: 8px;
          background: #2563eb;
          color: white;
          border: none;
          font-weight: 600;
          font-size: 14px;
        }
      `}} />
    </div>
  );
}
