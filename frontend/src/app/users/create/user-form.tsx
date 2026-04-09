"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Save, X, User, Shield, Briefcase, Phone, Mail, Building, Key } from "lucide-react";
import type { Branch } from "@/lib/types";
import { apiPost } from "@/lib/api";

interface UserFormProps {
  branches: Branch[];
}

export default function UserForm({ branches }: UserFormProps) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [formData, setFormData] = useState({
    email: "",
    password: "",
    full_name: "",
    phone: "",
    role: "employee",
    branch_id: "",
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);

    // Prepare payload, cleaning up empty branch_id
    const payload = { ...formData };
    if (!payload.branch_id) {
      delete (payload as any).branch_id;
    }

    try {
      const res = await apiPost(`/api/users`, payload);

      if (res.success) {
        router.push("/users");
        router.refresh();
      } else {
        if (res.error?.code === "DUPLICATE_EMAIL") {
          setError("Email này đã được sử dụng. Vui lòng chọn email khác.");
        } else {
          setError(res.error?.message || "Lỗi khi tạo nhân viên");
        }
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
        {/* Basic Information */}
        <section className="space-y-6">
          <div className="flex items-center gap-2 mb-4">
            <div className="p-2 bg-blue-100 rounded-xl">
              <User className="w-5 h-5 text-blue-600" />
            </div>
            <h3 className="text-lg font-bold text-gray-900">Thông tin cá nhân</h3>
          </div>

          <div className="space-y-1">
            <label className="flex items-center gap-2 text-xs font-bold text-gray-500 uppercase px-1">
              Họ và tên
            </label>
            <input
              type="text"
              required
              value={formData.full_name}
              onChange={e => setFormData({ ...formData, full_name: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              placeholder="VD: Nguyễn Văn A"
            />
          </div>

          <div className="space-y-1">
            <label className="flex items-center gap-2 text-xs font-bold text-gray-500 uppercase px-1">
              <Phone className="w-4 h-4 text-gray-400" />
              Số điện thoại
            </label>
            <input
              type="tel"
              value={formData.phone}
              onChange={e => setFormData({ ...formData, phone: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              placeholder="0912 345 678"
            />
          </div>
        </section>

        {/* Account Details */}
        <section className="space-y-6">
          <div className="flex items-center gap-2 mb-4">
            <div className="p-2 bg-indigo-100 rounded-xl">
              <Shield className="w-5 h-5 text-indigo-600" />
            </div>
            <h3 className="text-lg font-bold text-gray-900">Tài khoản & Phân quyền</h3>
          </div>

          <div className="space-y-1">
            <label className="flex items-center gap-2 text-xs font-bold text-gray-500 uppercase px-1">
              <Mail className="w-4 h-4 text-gray-400" />
              Email đăng nhập (bắt buộc)
            </label>
            <input
              type="email"
              required
              value={formData.email}
              onChange={e => setFormData({ ...formData, email: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              placeholder="email@congty.com"
            />
          </div>

          <div className="space-y-1">
            <label className="flex items-center gap-2 text-xs font-bold text-gray-500 uppercase px-1">
              <Key className="w-4 h-4 text-gray-400" />
              Mật khẩu khởi tạo (bắt buộc)
            </label>
            <input
              type="password"
              required
              minLength={6}
              value={formData.password}
              onChange={e => setFormData({ ...formData, password: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              placeholder="******"
            />
          </div>
        </section>

        {/* Work Placement */}
        <section className="space-y-6 md:col-span-2 pt-6 border-t border-gray-100">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
            <div className="space-y-1">
              <label className="flex items-center gap-2 text-xs font-bold text-gray-500 uppercase px-1">
                <Briefcase className="w-4 h-4 text-gray-400" />
                Vai trò hệ thống
              </label>
              <select
                value={formData.role}
                onChange={e => setFormData({ ...formData, role: e.target.value })}
                className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none appearance-none"
              >
                <option value="employee">Nhân viên vãng lai</option>
                <option value="manager">Quản lý chi nhánh</option>
                <option value="manager_device">Quản lý thiết bị (Manager Máy)</option>
                <option value="admin">Quản trị viên hệ thống (Admin)</option>
              </select>
            </div>

            <div className="space-y-1">
              <label className="flex items-center gap-2 text-xs font-bold text-gray-500 uppercase px-1">
                <Building className="w-4 h-4 text-gray-400" />
                Nơi làm việc (Chi nhánh)
              </label>
              <select
                value={formData.branch_id}
                onChange={e => setFormData({ ...formData, branch_id: e.target.value })}
                className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none appearance-none"
              >
                <option value="">Không gán chi nhánh (Tổng công ty)</option>
                {branches.map(b => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </section>
      </div>

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
          {loading ? "Đang xử lý..." : "Tạo nhân viên"}
        </button>
      </div>
    </form>
  );
}
