"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Save, X, MapPin, Clock, ShieldCheck, CheckSquare, Square } from "lucide-react";
import type { Branch } from "@/lib/types";
import { apiPut } from "@/lib/api";

const ALL_METHODS = [
  { id: "qr_totp", label: "QR Code" },
  { id: "ip", label: "IP Whitelist" },
  { id: "location", label: "GPS Location" },
  { id: "face", label: "Face ID" },
  { id: "password", label: "Password" },
  { id: "wifi_gps", label: "WiFi/GPS" },
  { id: "nfc", label: "NFC" },
  { id: "ble", label: "BLE" },
];

interface BranchFormProps {
  branch: Branch;
}

export default function BranchForm({ branch }: BranchFormProps) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [formData, setFormData] = useState({
    name: branch.name,
    address: branch.address,
    lat: branch.lat || 10.762622,
    lng: branch.lng || 106.660172,
    radius_m: branch.radius_m,
    allowed_methods: branch.allowed_methods ? branch.allowed_methods.split(",").map(m => m.trim()) : [],
    work_start_time: branch.work_start_time,
    work_end_time: branch.work_end_time,
    is_active: branch.is_active,
    require_biometric: branch.require_biometric,
  });

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
      const res = await apiPut(`/api/branches/${branch.id}`, {
        ...formData,
        allowed_methods: formData.allowed_methods.join(","),
      });

      if (res.success) {
        router.push("/branches");
        router.refresh();
      } else {
        setError(res.error?.message || "Lỗi cập nhật chi nhánh");
      }
    } catch (err) {
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
        {/* Basic Info */}
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

        {/* Location Settings */}
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

        {/* Work Hours */}
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

        {/* Status Settings */}
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

      {/* Methods */}
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
          {loading ? "Đang xử lý..." : "Lưu thay đổi"}
        </button>
      </div>
    </form>
  );
}
