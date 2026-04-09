"use client";

import { useState, useEffect, useCallback } from "react";
import { Clock, CheckCircle, AlertCircle, LogIn, LogOut } from "lucide-react";
import StatusBadge from "@/components/status-badge";

interface AdjustmentItem {
  id: string;
  work_date: string;
  requested_check_in?: string;
  requested_check_out?: string;
  reason: string;
  status: string;
  reviewer?: { full_name: string };
  reviewer_note?: string;
  created_at: string;
}

export default function MyAdjustmentsPage() {
  const [requests, setRequests] = useState<AdjustmentItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [success, setSuccess] = useState("");
  const [error, setError] = useState("");

  // Form state
  const [workDate, setWorkDate] = useState("");
  const [checkIn, setCheckIn] = useState("");
  const [checkOut, setCheckOut] = useState("");
  const [reason, setReason] = useState("");

  const baseUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch(
        `${baseUrl}/api/adjustments/my?page=${page}&limit=20`,
        { credentials: "include" }
      );
      const body = await res.json();
      if (body.success && body.data) {
        setRequests(body.data);
        if (body.meta) setTotal(body.meta.total);
      }
    } catch {
      setError("Không thể tải dữ liệu");
    } finally {
      setLoading(false);
    }
  }, [baseUrl, page]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitLoading(true);
    setError("");
    setSuccess("");

    try {
      const res = await fetch(`${baseUrl}/api/adjustments/my`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          work_date: workDate,
          check_in: checkIn || undefined,
          check_out: checkOut || undefined,
          reason,
        }),
      });

      const body = await res.json();
      if (body.success) {
        setSuccess("Yêu cầu bổ sung công đã được gửi thành công!");
        setWorkDate("");
        setCheckIn("");
        setCheckOut("");
        setReason("");
        fetchData();
      } else {
        setError(body.error?.message || "Gửi yêu cầu thất bại");
      }
    } catch {
      setError("Không thể kết nối đến server");
    } finally {
      setSubmitLoading(false);
    }
  }

  function formatTime(isoString?: string): string {
    if (!isoString) return "";
    try {
      return new Date(isoString).toLocaleTimeString("vi-VN", {
        hour: "2-digit",
        minute: "2-digit",
      });
    } catch {
      return isoString;
    }
  }

  return (
    <div className="min-h-full">
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="mb-8">
          <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight flex items-center gap-3">
            <Clock className="w-8 h-8 text-primary-600" />
            Bổ sung công
          </h1>
          <p className="mt-2 text-lg text-gray-500">
            Gửi yêu cầu chỉnh sửa giờ chấm công trong quá khứ
          </p>
        </div>

        {success && (
          <div className="mb-6 rounded-2xl bg-green-50 p-4 border border-green-100 flex items-center gap-3">
            <CheckCircle className="w-5 h-5 text-green-600" />
            <p className="text-sm font-bold text-green-800">{success}</p>
          </div>
        )}

        {error && (
          <div className="mb-6 rounded-2xl bg-red-50 p-4 border border-red-100 flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-red-600" />
            <p className="text-sm font-bold text-red-800">{error}</p>
          </div>
        )}

        <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
          {/* Form */}
          <div className="lg:col-span-1">
            <div className="bg-white rounded-3xl shadow-xl border border-gray-100 overflow-hidden sticky top-24">
              <div className="p-6 sm:p-8">
                <h2 className="text-xl font-bold text-gray-900 mb-6">
                  Yêu cầu mới
                </h2>

                <form onSubmit={handleSubmit} className="space-y-5">
                  <div>
                    <label className="block text-sm font-bold text-gray-700 mb-2">
                      Ngày cần bổ sung
                    </label>
                    <input
                      type="date"
                      value={workDate}
                      onChange={(e) => setWorkDate(e.target.value)}
                      required
                      className="block w-full rounded-2xl border-gray-200 bg-gray-50 py-3 px-4 text-gray-900 focus:border-primary-500 focus:ring-primary-500 sm:text-sm outline-none border focus:bg-white"
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-bold text-gray-700 mb-2">
                        Giờ vào
                      </label>
                      <input
                        type="time"
                        value={checkIn}
                        onChange={(e) => setCheckIn(e.target.value)}
                        className="block w-full rounded-2xl border-gray-200 bg-gray-50 py-3 px-4 text-gray-900 focus:border-primary-500 focus:ring-primary-500 sm:text-sm outline-none border focus:bg-white"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-bold text-gray-700 mb-2">
                        Giờ ra
                      </label>
                      <input
                        type="time"
                        value={checkOut}
                        onChange={(e) => setCheckOut(e.target.value)}
                        className="block w-full rounded-2xl border-gray-200 bg-gray-50 py-3 px-4 text-gray-900 focus:border-primary-500 focus:ring-primary-500 sm:text-sm outline-none border focus:bg-white"
                      />
                    </div>
                  </div>
                  <p className="text-xs text-gray-400 -mt-3">
                    Nhập ít nhất giờ vào hoặc giờ ra
                  </p>

                  <div>
                    <label className="block text-sm font-bold text-gray-700 mb-2">
                      Lý do
                    </label>
                    <textarea
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      rows={3}
                      required
                      placeholder="Ví dụ: Quên chấm công, lỗi hệ thống..."
                      className="block w-full rounded-2xl border-gray-200 bg-gray-50 py-3 px-4 text-gray-900 focus:border-primary-500 focus:ring-primary-500 sm:text-sm outline-none border focus:bg-white"
                    />
                  </div>

                  <button
                    type="submit"
                    disabled={submitLoading}
                    className="w-full flex items-center justify-center gap-3 rounded-2xl bg-primary-600 px-6 py-4 text-base font-bold text-white shadow-lg shadow-primary-200 hover:bg-primary-700 active:scale-95 transition-all disabled:opacity-50"
                  >
                    {submitLoading && <span className="spinner" />}
                    Gửi yêu cầu
                  </button>
                </form>
              </div>
            </div>
          </div>

          {/* List */}
          <div className="lg:col-span-2">
            <div className="bg-white rounded-3xl shadow-sm border border-gray-100 overflow-hidden">
              <div className="p-6 border-b border-gray-50 flex justify-between items-center">
                <h2 className="text-xl font-bold text-gray-900">
                  Lịch sử yêu cầu
                </h2>
                <span className="px-3 py-1 bg-gray-100 text-xs font-bold text-gray-500 rounded-full">
                  {total} yêu cầu
                </span>
              </div>

              {loading ? (
                <div className="p-8 text-center">
                  <span className="spinner-sm inline-block" />
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-100">
                    <thead className="bg-gray-50/50">
                      <tr>
                        <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                          Ngày
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                          Giờ yêu cầu
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                          Lý do
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                          Trạng thái
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                          Người duyệt
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-50 bg-white">
                      {requests.length === 0 ? (
                        <tr>
                          <td
                            colSpan={5}
                            className="px-6 py-12 text-center text-gray-400 italic text-sm"
                          >
                            Chưa có yêu cầu bổ sung công nào.
                          </td>
                        </tr>
                      ) : (
                        requests.map((req) => (
                          <tr
                            key={req.id}
                            className="hover:bg-gray-50/50 transition-colors"
                          >
                            <td className="px-6 py-4 whitespace-nowrap">
                              <div className="text-sm font-bold text-gray-900">
                                {req.work_date}
                              </div>
                              <div className="text-xs text-gray-400">
                                Gửi lúc{" "}
                                {new Date(req.created_at).toLocaleDateString(
                                  "vi-VN",
                                  {
                                    day: "2-digit",
                                    month: "2-digit",
                                    hour: "2-digit",
                                    minute: "2-digit",
                                  }
                                )}
                              </div>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm">
                              <div className="flex items-center gap-2">
                                {req.requested_check_in && (
                                  <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-blue-50 text-blue-700 rounded-lg text-xs font-bold">
                                    <LogIn className="w-3 h-3" />{" "}
                                    {formatTime(req.requested_check_in)}
                                  </span>
                                )}
                                {req.requested_check_out && (
                                  <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-orange-50 text-orange-700 rounded-lg text-xs font-bold">
                                    <LogOut className="w-3 h-3" />{" "}
                                    {formatTime(req.requested_check_out)}
                                  </span>
                                )}
                              </div>
                            </td>
                            <td className="px-6 py-4 text-sm text-gray-600 max-w-xs">
                              <p className="line-clamp-2" title={req.reason}>
                                {req.reason}
                              </p>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap">
                              <StatusBadge status={req.status} />
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                              {req.reviewer ? (
                                <div>
                                  {req.reviewer.full_name}
                                  {req.reviewer_note && (
                                    <div className="text-xs text-gray-400 italic">
                                      {req.reviewer_note}
                                    </div>
                                  )}
                                </div>
                              ) : (
                                <span className="text-gray-300">---</span>
                              )}
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              )}

              {total > 20 && (
                <div className="bg-gray-50/50 px-6 py-4 flex items-center justify-between border-t border-gray-50">
                  <p className="text-xs text-gray-500">
                    Trang{" "}
                    <span className="font-bold text-gray-900">{page}</span>
                  </p>
                  <div className="flex gap-2">
                    {page > 1 && (
                      <button
                        onClick={() => setPage((p) => p - 1)}
                        className="px-4 py-2 bg-white border border-gray-200 rounded-xl text-xs font-bold text-gray-600 hover:bg-gray-50 transition-all"
                      >
                        Trước
                      </button>
                    )}
                    {page * 20 < total && (
                      <button
                        onClick={() => setPage((p) => p + 1)}
                        className="px-4 py-2 bg-white border border-gray-200 rounded-xl text-xs font-bold text-gray-600 hover:bg-gray-50 transition-all"
                      >
                        Sau
                      </button>
                    )}
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
