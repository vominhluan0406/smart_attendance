"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Save, X, User as UserIcon, Shield, Briefcase, Phone, Mail, Building, Key, CheckSquare, Square } from "lucide-react";
import type { User, Branch } from "@/lib/types";
import { apiPut } from "@/lib/api";

interface EditUserFormProps {
  user: User;
  branches: Branch[];
}

export default function EditUserForm({ user, branches }: EditUserFormProps) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [formData, setFormData] = useState({
    full_name: user.full_name,
    email: user.email,
    phone: user.phone || "",
    employee_code: user.employee_code || "",
    role: user.role,
    branch_id: user.branch_id || "",
    position: user.position || "",
    is_active: user.is_active,
    password: "", // empty = don't change
  });

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const payload: Record<string, unknown> = {};

    if (formData.full_name !== user.full_name) payload.full_name = formData.full_name;
    if (formData.email !== user.email) payload.email = formData.email;
    if (formData.phone !== (user.phone || "")) payload.phone = formData.phone;
    if (formData.employee_code !== (user.employee_code || "")) payload.employee_code = formData.employee_code;
    if (formData.role !== user.role) payload.role = formData.role;
    if (formData.branch_id !== (user.branch_id || "")) payload.branch_id = formData.branch_id || null;
    if (formData.position !== (user.position || "")) payload.position = formData.position;
    if (formData.is_active !== user.is_active) payload.is_active = formData.is_active;
    if (formData.password) payload.password = formData.password;

    try {
      const res = await apiPut(`/api/users/${user.id}`, payload);

      if (res.success) {
        router.push("/users");
        router.refresh();
      } else {
        if (res.error?.code === "DUPLICATE_EMAIL") {
          setError("Email này đã được sử dụng. Vui lòng chọn email khác.");
        } else {
          setError(res.error?.message || "Lỗi cập nhật nhân viên");
        }
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
        {/* Thông tin cá nhân */}
        <section className="space-y-6">
          <div className="flex items-center gap-2 mb-4">
            <div className="p-2 bg-blue-100 rounded-xl">
              <UserIcon className="w-5 h-5 text-blue-600" />
            </div>
            <h3 className="text-lg font-bold text-gray-900">Thông tin cá nhân</h3>
          </div>

          <div className="space-y-1">
            <label className="text-xs font-bold text-gray-500 uppercase px-1">Họ và tên</label>
            <input
              type="text"
              required
              value={formData.full_name}
              onChange={e => setFormData({ ...formData, full_name: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
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
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs font-bold text-gray-500 uppercase px-1">Mã nhân viên</label>
            <input
              type="text"
              value={formData.employee_code}
              onChange={e => setFormData({ ...formData, employee_code: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              placeholder="VD: NV001"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs font-bold text-gray-500 uppercase px-1">Chức vụ</label>
            <input
              type="text"
              value={formData.position}
              onChange={e => setFormData({ ...formData, position: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              placeholder="VD: Trưởng phòng"
            />
          </div>
        </section>

        {/* Tài khoản */}
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
              Email đăng nhập
            </label>
            <input
              type="email"
              required
              value={formData.email}
              onChange={e => setFormData({ ...formData, email: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
            />
          </div>

          <div className="space-y-1">
            <label className="flex items-center gap-2 text-xs font-bold text-gray-500 uppercase px-1">
              <Key className="w-4 h-4 text-gray-400" />
              Đổi mật khẩu (để trống = không đổi)
            </label>
            <input
              type="password"
              minLength={6}
              value={formData.password}
              onChange={e => setFormData({ ...formData, password: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none"
              placeholder="Nhập mật khẩu mới..."
            />
          </div>

          <div className="space-y-1">
            <label className="flex items-center gap-2 text-xs font-bold text-gray-500 uppercase px-1">
              <Briefcase className="w-4 h-4 text-gray-400" />
              Vai trò
            </label>
            <select
              value={formData.role}
              onChange={e => setFormData({ ...formData, role: e.target.value as any })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none appearance-none"
            >
              <option value="employee">Nhân viên</option>
              <option value="manager">Quản lý chi nhánh</option>
              <option value="manager_device">Manager Máy (Kiosk)</option>
              <option value="admin">Quản trị viên</option>
            </select>
          </div>

          <div className="space-y-1">
            <label className="flex items-center gap-2 text-xs font-bold text-gray-500 uppercase px-1">
              <Building className="w-4 h-4 text-gray-400" />
              Chi nhánh
            </label>
            <select
              value={formData.branch_id}
              onChange={e => setFormData({ ...formData, branch_id: e.target.value })}
              className="w-full bg-gray-50 border border-gray-100 rounded-2xl px-4 py-3 text-sm focus:bg-white focus:ring-4 focus:ring-blue-100 focus:border-blue-500 transition-all outline-none appearance-none"
            >
              <option value="">Không gán chi nhánh</option>
              {branches.map(b => (
                <option key={b.id} value={b.id}>{b.name}</option>
              ))}
            </select>
          </div>

          <div>
            <button
              type="button"
              onClick={() => setFormData({ ...formData, is_active: !formData.is_active })}
              className={`flex items-center gap-2 p-4 rounded-2xl border-2 transition-all w-full justify-center ${
                formData.is_active
                  ? "bg-green-50 border-green-500 text-green-700 font-bold"
                  : "bg-red-50 border-red-300 text-red-600 font-bold"
              }`}
            >
              {formData.is_active ? <CheckSquare className="w-5 h-5" /> : <Square className="w-5 h-5" />}
              {formData.is_active ? "Đang hoạt động" : "Đã vô hiệu hóa"}
            </button>
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
          <Save className="w-4 h-4" />
          {loading ? "Đang lưu..." : "Lưu thay đổi"}
        </button>
      </div>
    </form>
  );
}
