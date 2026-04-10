"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Save, X, MapPin, Clock, ShieldCheck, CheckSquare, Square, Plus, Trash2, Wifi, Navigation } from "lucide-react";
import { apiPost } from "@/lib/api";

interface IPEntry {
  ip_cidr: string;
  label: string;
}

interface LocationEntry {
  label: string;
  lat: number;
  lng: number;
  radius_m: number;
}

const ALL_METHODS = [
  { id: "qr_totp", label: "QR Code" },
  { id: "ip", label: "IP Whitelist" },
  { id: "location", label: "GPS Location" },
  { id: "face", label: "Face ID" },
  { id: "password", label: "Mật khẩu" },
  { id: "wifi_gps", label: "WiFi/GPS" },
  { id: "nfc", label: "NFC" },
];

export default function CreateBranchForm() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [formData, setFormData] = useState({
    name: "",
    address: "",
    lat: 10.762622,
    lng: 106.660172,
    radius_m: 200,
    allowed_methods: ["qr_totp", "password"],
    work_start_time: "08:00",
    work_end_time: "17:00",
    is_active: true,
    require_biometric: false,
  });

  const [ipWhitelist, setIpWhitelist] = useState<IPEntry[]>([
    { ip_cidr: "", label: "" },
  ]);

  const [locations, setLocations] = useState<LocationEntry[]>([
    { label: "", lat: 10.762622, lng: 106.660172, radius_m: 200 },
  ]);

  const toggleMethod = (methodId: string) => {
    setFormData(prev => ({
      ...prev,
      allowed_methods: prev.allowed_methods.includes(methodId)
        ? prev.allowed_methods.filter(m => m !== methodId)
        : [...prev.allowed_methods, methodId]
    }));
  };

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const validIPs = ipWhitelist.filter(ip => ip.ip_cidr.trim() !== "");
      const validLocations = locations.filter(loc => loc.label.trim() !== "" || loc.lat !== 0);

      const res = await apiPost("/api/branches", {
        ...formData,
        allowed_methods: formData.allowed_methods.join(","),
        ip_whitelist: validIPs,
        locations: validLocations,
      });

      if (res.success) {
        router.push("/branches");
        router.refresh();
      } else {
        setError(res.error?.message || "Lỗi tạo chi nhánh");
      }
    } catch {
      setError("Không thể kết nối đến server");
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-8">
      {error && (
        <div className="p-4 bg-red-50 border border-red-100 rounded-2xl text-red-700 text-sm font-bold">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
        {/* Thông tin cơ bản */}
        <section className="space-y-6">
          <div className="flex items-center gap-2 mb-4">
            <div className="p-2 bg-blue-100 rounded-xl">
              <ShieldCheck className="w-5 h-5 text-blue-600" />
            </div>
            <h3 className="text-lg font-bold text-gray-900">Thông tin cơ bản</h3>
          </div>

          <div className="space-y-1">
            <label className="text-xs font-bold text-gray-500 uppercase px-1">Tên chi nhánh</label>
            <input
              type="text"
              required
              value={formData.name}
              onChange={e => setFormData({ ...formData, name: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              placeholder="VD: Chi nhánh Quận 1"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs font-bold text-gray-500 uppercase px-1">Địa chỉ</label>
            <textarea
              required
              rows={2}
              value={formData.address}
              onChange={e => setFormData({ ...formData, address: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none resize-none"
              placeholder="Địa chỉ chi tiết..."
            />
          </div>
        </section>

        {/* Vị trí */}
        <section className="space-y-6">
          <div className="flex items-center gap-2 mb-4">
            <div className="p-2 bg-green-100 rounded-xl">
              <MapPin className="w-5 h-5 text-green-600" />
            </div>
            <h3 className="text-lg font-bold text-gray-900">Vị trí & Bán kính</h3>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1">
              <label className="text-xs font-bold text-gray-500 uppercase px-1">Latitude</label>
              <input
                type="number"
                step="any"
                required
                value={formData.lat}
                onChange={e => setFormData({ ...formData, lat: parseFloat(e.target.value) })}
                className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs font-bold text-gray-500 uppercase px-1">Longitude</label>
              <input
                type="number"
                step="any"
                required
                value={formData.lng}
                onChange={e => setFormData({ ...formData, lng: parseFloat(e.target.value) })}
                className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              />
            </div>
          </div>

          <div className="space-y-1">
            <label className="text-xs font-bold text-gray-500 uppercase px-1">Bán kính chấp nhận (m)</label>
            <input
              type="number"
              required
              value={formData.radius_m}
              onChange={e => setFormData({ ...formData, radius_m: parseInt(e.target.value) })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
            />
          </div>
        </section>

        {/* Giờ làm */}
        <section className="space-y-6">
          <div className="flex items-center gap-2 mb-4">
            <div className="p-2 bg-yellow-100 rounded-xl">
              <Clock className="w-5 h-5 text-yellow-600" />
            </div>
            <h3 className="text-lg font-bold text-gray-900">Giờ làm việc</h3>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1">
              <label className="text-xs font-bold text-gray-500 uppercase px-1">Bắt đầu</label>
              <input
                type="time"
                required
                value={formData.work_start_time}
                onChange={e => setFormData({ ...formData, work_start_time: e.target.value })}
                className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs font-bold text-gray-500 uppercase px-1">Kết thúc</label>
              <input
                type="time"
                required
                value={formData.work_end_time}
                onChange={e => setFormData({ ...formData, work_end_time: e.target.value })}
                className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              />
            </div>
          </div>
        </section>

        {/* Trạng thái */}
        <section className="space-y-6">
          <div className="flex items-center gap-2 mb-4">
            <div className="p-2 bg-indigo-100 rounded-xl">
              <ShieldCheck className="w-5 h-5 text-indigo-600" />
            </div>
            <h3 className="text-lg font-bold text-gray-900">Trạng thái & Bảo mật</h3>
          </div>

          <div className="flex gap-4">
            <button
              type="button"
              onClick={() => setFormData({ ...formData, is_active: !formData.is_active })}
              className={`flex-1 flex items-center justify-center gap-2 p-4 rounded-2xl border-2 transition-all ${
                formData.is_active
                  ? "bg-green-50 border-green-500 text-green-700 font-bold"
                  : "bg-gray-50 border-gray-200 text-gray-500"
              }`}
            >
              {formData.is_active ? <CheckSquare className="w-5 h-5" /> : <Square className="w-5 h-5" />}
              Hoạt động
            </button>
            <button
              type="button"
              onClick={() => setFormData({ ...formData, require_biometric: !formData.require_biometric })}
              className={`flex-1 flex items-center justify-center gap-2 p-4 rounded-2xl border-2 transition-all ${
                formData.require_biometric
                  ? "bg-blue-50 border-blue-500 text-blue-700 font-bold"
                  : "bg-gray-50 border-gray-200 text-gray-500"
              }`}
            >
              {formData.require_biometric ? <CheckSquare className="w-5 h-5" /> : <Square className="w-5 h-5" />}
              Bắt buộc vân tay
            </button>
          </div>
        </section>
      </div>

      {/* IP Whitelist */}
      <section className="space-y-4 pt-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="p-2 bg-orange-100 rounded-xl">
              <Wifi className="w-5 h-5 text-orange-600" />
            </div>
            <div>
              <h3 className="text-lg font-bold text-gray-900">IP Whitelist</h3>
              <p className="text-xs text-gray-400">Danh sách IP/CIDR được phép check-in</p>
            </div>
          </div>
          <button
            type="button"
            onClick={() => setIpWhitelist([...ipWhitelist, { ip_cidr: "", label: "" }])}
            className="flex items-center gap-1 px-3 py-1.5 rounded-xl bg-orange-50 text-orange-600 text-xs font-bold hover:bg-orange-100 transition-all"
          >
            <Plus className="w-3.5 h-3.5" />
            Thêm IP
          </button>
        </div>
        <div className="space-y-3">
          {ipWhitelist.map((ip, idx) => (
            <div key={idx} className="flex items-center gap-3">
              <input
                type="text"
                value={ip.ip_cidr}
                onChange={e => {
                  const updated = [...ipWhitelist];
                  updated[idx].ip_cidr = e.target.value;
                  setIpWhitelist(updated);
                }}
                placeholder="VD: 192.168.1.0/24"
                className="flex-1 bg-gray-50 border border-gray-100 rounded-xl px-4 py-2.5 text-sm focus:bg-white focus:ring-4 focus:ring-orange-100 focus:border-orange-500 transition-all outline-none font-mono"
              />
              <input
                type="text"
                value={ip.label}
                onChange={e => {
                  const updated = [...ipWhitelist];
                  updated[idx].label = e.target.value;
                  setIpWhitelist(updated);
                }}
                placeholder="Ghi chú (VD: WiFi văn phòng)"
                className="flex-1 bg-gray-50 border border-gray-100 rounded-xl px-4 py-2.5 text-sm focus:bg-white focus:ring-4 focus:ring-orange-100 focus:border-orange-500 transition-all outline-none"
              />
              {ipWhitelist.length > 1 && (
                <button
                  type="button"
                  onClick={() => setIpWhitelist(ipWhitelist.filter((_, i) => i !== idx))}
                  className="p-2 text-red-400 hover:text-red-600 hover:bg-red-50 rounded-xl transition-all"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              )}
            </div>
          ))}
        </div>
      </section>

      {/* Location Whitelist */}
      <section className="space-y-4 pt-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="p-2 bg-teal-100 rounded-xl">
              <Navigation className="w-5 h-5 text-teal-600" />
            </div>
            <div>
              <h3 className="text-lg font-bold text-gray-900">Vị trí cho phép (Location Whitelist)</h3>
              <p className="text-xs text-gray-400">Tọa độ GPS + bán kính cho phép check-in</p>
            </div>
          </div>
          <button
            type="button"
            onClick={() => setLocations([...locations, { label: "", lat: 0, lng: 0, radius_m: 200 }])}
            className="flex items-center gap-1 px-3 py-1.5 rounded-xl bg-teal-50 text-teal-600 text-xs font-bold hover:bg-teal-100 transition-all"
          >
            <Plus className="w-3.5 h-3.5" />
            Thêm vị trí
          </button>
        </div>
        <div className="space-y-3">
          {locations.map((loc, idx) => (
            <div key={idx} className="flex items-center gap-3 flex-wrap sm:flex-nowrap">
              <input
                type="text"
                value={loc.label}
                onChange={e => {
                  const updated = [...locations];
                  updated[idx].label = e.target.value;
                  setLocations(updated);
                }}
                placeholder="Tên vị trí (VD: Cổng chính)"
                className="flex-1 min-w-[140px] bg-gray-50 border border-gray-100 rounded-xl px-4 py-2.5 text-sm focus:bg-white focus:ring-4 focus:ring-teal-100 focus:border-teal-500 transition-all outline-none"
              />
              <input
                type="number"
                step="any"
                value={loc.lat}
                onChange={e => {
                  const updated = [...locations];
                  updated[idx].lat = parseFloat(e.target.value) || 0;
                  setLocations(updated);
                }}
                placeholder="Lat"
                className="w-28 bg-gray-50 border border-gray-100 rounded-xl px-3 py-2.5 text-sm font-mono focus:bg-white focus:ring-4 focus:ring-teal-100 focus:border-teal-500 transition-all outline-none"
              />
              <input
                type="number"
                step="any"
                value={loc.lng}
                onChange={e => {
                  const updated = [...locations];
                  updated[idx].lng = parseFloat(e.target.value) || 0;
                  setLocations(updated);
                }}
                placeholder="Lng"
                className="w-28 bg-gray-50 border border-gray-100 rounded-xl px-3 py-2.5 text-sm font-mono focus:bg-white focus:ring-4 focus:ring-teal-100 focus:border-teal-500 transition-all outline-none"
              />
              <input
                type="number"
                value={loc.radius_m}
                onChange={e => {
                  const updated = [...locations];
                  updated[idx].radius_m = parseInt(e.target.value) || 200;
                  setLocations(updated);
                }}
                placeholder="Bán kính (m)"
                className="w-24 bg-gray-50 border border-gray-100 rounded-xl px-3 py-2.5 text-sm focus:bg-white focus:ring-4 focus:ring-teal-100 focus:border-teal-500 transition-all outline-none"
              />
              {locations.length > 1 && (
                <button
                  type="button"
                  onClick={() => setLocations(locations.filter((_, i) => i !== idx))}
                  className="p-2 text-red-400 hover:text-red-600 hover:bg-red-50 rounded-xl transition-all"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              )}
            </div>
          ))}
        </div>
      </section>

      {/* Phương thức */}
      <section className="space-y-4 pt-4">
        <label className="text-xs font-bold text-gray-500 uppercase px-1">Phương thức chấm công được phép</label>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {ALL_METHODS.map(m => (
            <button
              key={m.id}
              type="button"
              onClick={() => toggleMethod(m.id)}
              className={`flex items-center gap-2 p-3 rounded-2xl border-2 transition-all text-xs font-bold ${
                formData.allowed_methods.includes(m.id)
                  ? "bg-blue-50 border-blue-500 text-blue-700 shadow-sm"
                  : "bg-white border-gray-100 text-gray-400 hover:border-gray-200"
              }`}
            >
              <div className={`w-4 h-4 rounded-full border-2 flex-shrink-0 flex items-center justify-center ${
                formData.allowed_methods.includes(m.id) ? "bg-blue-500 border-blue-500" : "border-gray-300"
              }`}>
                {formData.allowed_methods.includes(m.id) && <div className="w-1.5 h-1.5 bg-white rounded-full" />}
              </div>
              {m.label}
            </button>
          ))}
        </div>
      </section>

      {/* Actions */}
      <div className="pt-8 border-t border-gray-100 flex items-center justify-end gap-4">
        <button
          type="button"
          onClick={() => router.back()}
          className="px-6 py-3 rounded-2xl text-sm font-bold text-gray-500 hover:bg-gray-50 transition-all flex items-center gap-2"
        >
          <X className="w-4 h-4" />
          Hủy
        </button>
        <button
          type="submit"
          disabled={loading}
          className="px-10 py-3 rounded-2xl bg-blue-600 text-white text-sm font-bold shadow-xl shadow-blue-200 hover:bg-blue-700 active:scale-95 transition-all flex items-center gap-2 disabled:opacity-50"
        >
          <Save className="w-4 h-4" />
          {loading ? "Đang tạo..." : "Tạo chi nhánh"}
        </button>
      </div>
    </form>
  );
}
