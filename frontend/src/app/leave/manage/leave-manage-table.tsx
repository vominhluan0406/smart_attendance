"use client";

import { useState } from "react";
import { Check, X, Inbox } from "lucide-react";
import type { LeaveRequest } from "@/lib/types";

interface Props {
  requests: LeaveRequest[];
  statusFilter: string;
}

export default function LeaveManageTable({ requests: initial, statusFilter }: Props) {
  const [requests, setRequests] = useState(initial);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const baseUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  async function handleReview(id: string, status: "approved" | "rejected") {
    setActionLoading(id);
    try {
      const res = await fetch(`${baseUrl}/api/leave/manage/${id}/review`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ status }),
      });
      const body = await res.json();
      if (body.success) {
        setRequests((prev) => prev.filter((r) => r.id !== id));
      }
    } catch {
      // Silently fail - user can retry
    } finally {
      setActionLoading(null);
    }
  }

  return (
    <div className="bg-white rounded-3xl shadow-sm border border-gray-100 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-100">
          <thead className="bg-gray-50/50">
            <tr>
              <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                Nhân viên
              </th>
              <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                Thời gian
              </th>
              <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                Loại & Lý do
              </th>
              <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                Số ngày
              </th>
              <th className="px-6 py-4 text-right text-xs font-bold text-gray-400 uppercase tracking-wider">
                Thao tác
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-50 bg-white">
            {requests.length === 0 ? (
              <tr>
                <td
                  colSpan={5}
                  className="px-6 py-20 text-center text-gray-400 italic"
                >
                  <Inbox className="w-12 h-12 mx-auto mb-4 opacity-20" />
                  Không có yêu cầu nào cần xử lý.
                </td>
              </tr>
            ) : (
              requests.map((req) => (
                <tr
                  key={req.id}
                  className="hover:bg-gray-50/50 transition-colors"
                >
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 rounded-full bg-primary-50 flex items-center justify-center text-primary-600 font-bold">
                        {req.user?.full_name?.charAt(0) || "?"}
                      </div>
                      <div>
                        <div className="text-sm font-bold text-gray-900">
                          {req.user?.full_name}
                        </div>
                        <div className="text-xs text-gray-400">
                          {req.user?.email}
                        </div>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-bold text-gray-900">
                      {req.start_date}
                    </div>
                    <div className="text-xs text-gray-400">
                      đến {req.end_date}
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-bold mb-1"
                      style={{
                        backgroundColor: req.leave_type?.color
                          ? `${req.leave_type.color}15`
                          : "#f3f4f6",
                        color: req.leave_type?.color || "#6b7280",
                      }}
                    >
                      {req.leave_type?.name || "Nghỉ phép"}
                    </span>
                    <p
                      className="text-sm text-gray-600 line-clamp-1"
                      title={req.reason}
                    >
                      {req.reason}
                    </p>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-bold text-gray-700">
                    {req.total_days} ngày
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right">
                    {statusFilter === "pending" ? (
                      <div className="flex justify-end gap-2">
                        <button
                          onClick={() => handleReview(req.id, "approved")}
                          disabled={actionLoading === req.id}
                          className="p-2 bg-green-50 text-green-600 rounded-xl hover:bg-green-600 hover:text-white transition-all shadow-sm disabled:opacity-50"
                        >
                          <Check className="w-5 h-5" />
                        </button>
                        <button
                          onClick={() => handleReview(req.id, "rejected")}
                          disabled={actionLoading === req.id}
                          className="p-2 bg-red-50 text-red-600 rounded-xl hover:bg-red-600 hover:text-white transition-all shadow-sm disabled:opacity-50"
                        >
                          <X className="w-5 h-5" />
                        </button>
                      </div>
                    ) : (
                      <div>
                        <div
                          className={`text-xs font-bold uppercase tracking-wider ${
                            req.status === "approved"
                              ? "text-green-600"
                              : "text-red-600"
                          }`}
                        >
                          {req.status === "approved"
                            ? "Đã duyệt"
                            : "Từ chối"}
                        </div>
                        <div className="text-[10px] text-gray-400 mt-1">
                          bởi{" "}
                          {req.reviewer?.full_name || "Admin"}
                        </div>
                      </div>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
