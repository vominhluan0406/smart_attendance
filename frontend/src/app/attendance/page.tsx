"use client";

import { useState, useEffect, useCallback } from "react";
import { Camera, MapPin, CheckCircle, AlertCircle } from "lucide-react";

export default function AttendancePage() {
  const [totpCode, setTotpCode] = useState("");
  const [lat, setLat] = useState<number | null>(null);
  const [lng, setLng] = useState<number | null>(null);
  const [accuracy, setAccuracy] = useState<number | null>(null);
  const [gpsStatus, setGpsStatus] = useState("Dang xac dinh vi tri...");
  const [gpsOk, setGpsOk] = useState(false);
  const [deviceFingerprint, setDeviceFingerprint] = useState("");
  const [result, setResult] = useState<{
    success: boolean;
    message: string;
  } | null>(null);
  const [loading, setLoading] = useState(false);

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
      setGpsStatus("Trinh duyet khong ho tro GPS");
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
        setGpsStatus(`Loi GPS: ${err.message}`);
        setGpsOk(false);
      },
      { enableHighAccuracy: true, timeout: 20000 }
    );

    return () => navigator.geolocation.clearWatch(watchId);
  }, []);

  const handleSubmit = useCallback(
    async (code?: string) => {
      setLoading(true);
      setResult(null);

      const submitCode = code || totpCode;
      if (!submitCode) {
        setResult({ success: false, message: "Vui long nhap ma TOTP" });
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
            totp_code: submitCode,
            lat: lat,
            lng: lng,
            accuracy: accuracy,
            device_fingerprint: deviceFingerprint,
          }),
        });

        const body = await res.json();
        if (body.success) {
          setResult({ success: true, message: "Cham cong thanh cong!" });
          setTotpCode("");
        } else {
          setResult({
            success: false,
            message: body.error?.message || "Cham cong that bai",
          });
        }
      } catch {
        setResult({
          success: false,
          message: "Khong the ket noi den server",
        });
      } finally {
        setLoading(false);
      }
    },
    [totpCode, lat, lng, accuracy, deviceFingerprint]
  );

  return (
    <div className="min-h-full bg-gray-50">
      <main className="mx-auto max-w-md px-4 py-6">
        <div className="flex items-center justify-center gap-3 mb-8">
          <div className="p-2 bg-primary-100 rounded-xl">
            <Camera className="w-6 h-6 text-primary-600" />
          </div>
          <h1 className="text-2xl font-bold text-gray-900">Cham cong</h1>
        </div>

        {/* Result */}
        {result && (
          <div
            className={`mb-6 p-4 rounded-2xl flex items-center gap-3 ${
              result.success
                ? "bg-green-50 border border-green-100"
                : "bg-red-50 border border-red-100"
            }`}
          >
            {result.success ? (
              <CheckCircle className="w-5 h-5 text-green-600 flex-shrink-0" />
            ) : (
              <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0" />
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

        {/* QR Scanner Placeholder */}
        <div className="bg-white shadow-xl rounded-2xl border border-gray-100 overflow-hidden mb-6">
          <div className="p-6">
            <div className="rounded-xl overflow-hidden border-2 border-dashed border-gray-200 aspect-square flex items-center justify-center bg-gray-900">
              <div className="text-center p-6">
                <Camera className="w-12 h-12 text-gray-600 mx-auto mb-4 animate-pulse" />
                <p className="text-sm text-gray-400">
                  Camera QR scanner
                </p>
                <p className="text-xs text-gray-500 mt-2">
                  Tich hop thu vien html5-qrcode de quet ma QR
                </p>
              </div>
            </div>

            {/* GPS Status */}
            <div className="mt-6">
              <div
                className={`flex items-center justify-center text-xs font-bold mb-4 px-3 py-1 rounded-full w-max mx-auto ${
                  gpsOk
                    ? "text-green-600 bg-green-50"
                    : "text-gray-400 bg-gray-50"
                }`}
              >
                <span className="relative flex h-2 w-2 mr-2">
                  {!gpsOk && (
                    <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-yellow-400 opacity-75" />
                  )}
                  <span
                    className={`relative inline-flex rounded-full h-2 w-2 ${
                      gpsOk ? "bg-green-500" : "bg-yellow-500"
                    }`}
                  />
                </span>
                {gpsStatus}
              </div>
            </div>
          </div>
        </div>

        {/* Manual TOTP Input */}
        <div className="bg-white shadow-sm rounded-2xl border border-gray-100 p-6 mb-6">
          <h3 className="text-sm font-bold text-gray-500 uppercase mb-4">
            Nhap ma TOTP thu cong
          </h3>
          <div className="flex gap-3">
            <input
              type="text"
              value={totpCode}
              onChange={(e) => setTotpCode(e.target.value)}
              placeholder="Nhap ma 6 so..."
              maxLength={6}
              className="flex-1 rounded-2xl border-gray-200 bg-gray-50 py-3 px-4 text-gray-900 focus:border-primary-500 focus:ring-primary-500 text-sm outline-none border focus:bg-white text-center font-mono text-lg tracking-widest"
            />
            <button
              onClick={() => handleSubmit()}
              disabled={loading || !totpCode}
              className="rounded-2xl bg-primary-600 px-6 py-3 text-sm font-bold text-white shadow-lg hover:bg-primary-700 active:scale-95 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? <span className="spinner" /> : "Gui"}
            </button>
          </div>
        </div>

        {/* GPS Info */}
        {lat !== null && lng !== null && (
          <div className="bg-white shadow-sm rounded-2xl border border-gray-100 p-4">
            <div className="flex items-center gap-2 text-sm text-gray-600">
              <MapPin className="w-4 h-4 text-primary-600" />
              <span className="font-mono text-xs">
                {lat.toFixed(6)}, {lng.toFixed(6)} (+-
                {Math.round(accuracy || 0)}m)
              </span>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
