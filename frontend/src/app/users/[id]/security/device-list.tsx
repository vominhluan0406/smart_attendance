"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Smartphone, ShieldBan, ShieldCheck, Trash2 } from "lucide-react";
import type { UserDevice } from "@/lib/types";
import { apiPost, apiDelete } from "@/lib/api";

interface Props {
  userId: string;
  devices: UserDevice[];
}

export default function DeviceList({ userId, devices }: Props) {
  const router = useRouter();
  const [loading, setLoading] = useState<string | null>(null);

  async function handleBlock(deviceId: string) {
    setLoading(deviceId);
    try {
      await apiPost(`/api/users/${userId}/devices/${deviceId}/block`, {});
      router.refresh();
    } finally {
      setLoading(null);
    }
  }

  async function handleUnblock(deviceId: string) {
    setLoading(deviceId);
    try {
      await apiPost(`/api/users/${userId}/devices/${deviceId}/unblock`, {});
      router.refresh();
    } finally {
      setLoading(null);
    }
  }

  async function handleDelete(deviceId: string) {
    if (!confirm("Xoá thiết bị này? Lần check-in tiếp theo từ thiết bị này sẽ được ghi nhận là thiết bị mới.")) return;
    setLoading(deviceId);
    try {
      await apiDelete(`/api/users/${userId}/devices/${deviceId}`);
      router.refresh();
    } finally {
      setLoading(null);
    }
  }

  return (
    <div className="bg-white rounded-3xl shadow-sm border border-gray-100 overflow-hidden">
      <div className="p-6 border-b border-gray-50 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Smartphone className="w-5 h-5 text-blue-600" />
          <h2 className="text-lg font-bold text-gray-900">Thiết bị đã biết</h2>
        </div>
        <span className="px-3 py-1 bg-gray-100 text-xs font-bold text-gray-500 rounded-full">
          {devices.length} thiết bị
        </span>
      </div>

      {devices.length === 0 ? (
        <div className="px-6 py-12 text-center text-gray-400 text-sm">
          Nhân viên chưa check-in từ thiết bị nào.
        </div>
      ) : (
        <div className="divide-y divide-gray-50">
          {devices.map((device) => (
            <div key={device.id} className="px-6 py-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className={`p-2 rounded-xl ${device.is_blocked ? "bg-red-50" : device.is_trusted ? "bg-green-50" : "bg-gray-50"}`}>
                  {device.is_blocked ? (
                    <ShieldBan className="w-5 h-5 text-red-500" />
                  ) : (
                    <Smartphone className="w-5 h-5 text-blue-500" />
                  )}
                </div>
                <div>
                  <div className="text-sm font-bold text-gray-900 flex items-center gap-2">
                    {device.device_name || "Thiết bị không xác định"}
                    {device.is_blocked ? (
                      <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold bg-red-50 text-red-700">
                        Đã chặn
                      </span>
                    ) : device.is_trusted ? (
                      <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold bg-green-50 text-green-700">
                        Tin cậy
                      </span>
                    ) : (
                      <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold bg-gray-100 text-gray-500">
                        Mới
                      </span>
                    )}
                  </div>
                  <div className="text-xs text-gray-400 mt-0.5">
                    <span className="font-mono">{device.fingerprint_hash.substring(0, 16)}...</span>
                    <span className="mx-2">|</span>
                    Lần cuối: {new Date(device.last_seen_at).toLocaleString("vi-VN")}
                  </div>
                  {device.user_agent && (
                    <div className="text-xs text-gray-300 mt-0.5 line-clamp-1 max-w-md" title={device.user_agent}>
                      {device.user_agent}
                    </div>
                  )}
                </div>
              </div>

              <div className="flex items-center gap-2">
                {device.is_blocked ? (
                  <button
                    onClick={() => handleUnblock(device.id)}
                    disabled={loading === device.id}
                    className="p-2 bg-green-50 text-green-600 rounded-xl hover:bg-green-600 hover:text-white transition-all"
                    title="Bỏ chặn"
                  >
                    <ShieldCheck className="w-4 h-4" />
                  </button>
                ) : (
                  <button
                    onClick={() => handleBlock(device.id)}
                    disabled={loading === device.id}
                    className="p-2 bg-red-50 text-red-600 rounded-xl hover:bg-red-600 hover:text-white transition-all"
                    title="Chặn thiết bị"
                  >
                    <ShieldBan className="w-4 h-4" />
                  </button>
                )}
                <button
                  onClick={() => handleDelete(device.id)}
                  disabled={loading === device.id}
                  className="p-2 bg-gray-50 text-gray-400 rounded-xl hover:bg-red-600 hover:text-white transition-all"
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
