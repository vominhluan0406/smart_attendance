"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Fingerprint, Check, XCircle, Trash2, ShieldCheck, ShieldOff } from "lucide-react";
import type { UserCredential } from "@/lib/types";
import { apiPost, apiDelete } from "@/lib/api";

interface Props {
  userId: string;
  credentials: UserCredential[];
}

export default function CredentialList({ userId, credentials }: Props) {
  const router = useRouter();
  const [loading, setLoading] = useState<string | null>(null);

  async function handleApprove(credId: string) {
    setLoading(credId);
    try {
      await apiPost(`/api/users/${userId}/credentials/${credId}/approve`, {});
      router.refresh();
    } finally {
      setLoading(null);
    }
  }

  async function handleRevoke(credId: string) {
    setLoading(credId);
    try {
      await apiPost(`/api/users/${userId}/credentials/${credId}/revoke`, {});
      router.refresh();
    } finally {
      setLoading(null);
    }
  }

  async function handleDelete(credId: string) {
    if (!confirm("Xoá credential này? Nhân viên sẽ không thể dùng thiết bị này để xác thực.")) return;
    setLoading(credId);
    try {
      await apiDelete(`/api/users/${userId}/credentials/${credId}`);
      router.refresh();
    } finally {
      setLoading(null);
    }
  }

  return (
    <div className="bg-white rounded-3xl shadow-sm border border-gray-100 overflow-hidden">
      <div className="p-6 border-b border-gray-50 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Fingerprint className="w-5 h-5 text-indigo-600" />
          <h2 className="text-lg font-bold text-gray-900">Sinh trắc học (WebAuthn)</h2>
        </div>
        <span className="px-3 py-1 bg-gray-100 text-xs font-bold text-gray-500 rounded-full">
          {credentials.length} credential
        </span>
      </div>

      {credentials.length === 0 ? (
        <div className="px-6 py-12 text-center text-gray-400 text-sm">
          Nhân viên chưa đăng ký thiết bị sinh trắc học nào.
        </div>
      ) : (
        <div className="divide-y divide-gray-50">
          {credentials.map((cred) => (
            <div key={cred.id} className="px-6 py-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className={`p-2 rounded-xl ${cred.is_approved ? "bg-green-50" : "bg-amber-50"}`}>
                  {cred.is_approved ? (
                    <ShieldCheck className="w-5 h-5 text-green-600" />
                  ) : (
                    <ShieldOff className="w-5 h-5 text-amber-600" />
                  )}
                </div>
                <div>
                  <div className="text-sm font-bold text-gray-900 flex items-center gap-2">
                    {cred.transport || "Platform"}
                    {cred.is_approved ? (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-bold bg-green-50 text-green-700">
                        <Check className="w-3 h-3" /> Đã duyệt
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-bold bg-amber-50 text-amber-700">
                        Chờ duyệt
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-gray-400 mt-0.5 flex items-center gap-3">
                    <span>Sign count: {cred.sign_count}</span>
                    <span>Loại: {cred.attestation_type || "none"}</span>
                    <span>Đăng ký: {new Date(cred.created_at).toLocaleDateString("vi-VN")}</span>
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-2">
                {!cred.is_approved ? (
                  <button
                    onClick={() => handleApprove(cred.id)}
                    disabled={loading === cred.id}
                    className="p-2 bg-green-50 text-green-600 rounded-xl hover:bg-green-600 hover:text-white transition-all"
                    title="Phê duyệt"
                  >
                    <Check className="w-4 h-4" />
                  </button>
                ) : (
                  <button
                    onClick={() => handleRevoke(cred.id)}
                    disabled={loading === cred.id}
                    className="p-2 bg-amber-50 text-amber-600 rounded-xl hover:bg-amber-600 hover:text-white transition-all"
                    title="Thu hồi"
                  >
                    <XCircle className="w-4 h-4" />
                  </button>
                )}
                <button
                  onClick={() => handleDelete(cred.id)}
                  disabled={loading === cred.id}
                  className="p-2 bg-red-50 text-red-600 rounded-xl hover:bg-red-600 hover:text-white transition-all"
                  title="Xoá"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
