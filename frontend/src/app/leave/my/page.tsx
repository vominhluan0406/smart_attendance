"use client";

import { useState, useEffect, useCallback } from "react";
import { CalendarOff, CheckCircle, AlertCircle } from "lucide-react";
import StatusBadge from "@/components/status-badge";

interface LeaveTypeItem {
  id: string;
  name: string;
  is_paid: boolean;
}

interface LeaveRequestItem {
  id: string;
  start_date: string;
  end_date: string;
  total_days: number;
  reason: string;
  status: string;
  leave_type?: { name: string; color: string };
  reviewer?: { full_name: string };
}

export default function MyLeavePage() {
  const [leaveTypes, setLeaveTypes] = useState<LeaveTypeItem[]>([]);
  const [requests, setRequests] = useState<LeaveRequestItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [success, setSuccess] = useState("");
  const [error, setError] = useState("");

  // Form state
  const [leaveTypeId, setLeaveTypeId] = useState("");
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [reason, setReason] = useState("");

  const baseUrl =
    process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [typesRes, reqsRes] = await Promise.all([
        fetch(`${baseUrl}/api/leave/types`, { credentials: "include" }),
        fetch(`${baseUrl}/api/leave/my?page=${page}&limit=20`, {
          credentials: "include",
        }),
      ]);

      const typesBody = await typesRes.json();
      if (typesBody.success && typesBody.data) {
        setLeaveTypes(typesBody.data);
        if (typesBody.data.length > 0 && !leaveTypeId) {
          setLeaveTypeId(typesBody.data[0].id);
        }
      }

      const reqsBody = await reqsRes.json();
      if (reqsBody.success && reqsBody.data) {
        setRequests(reqsBody.data);
        if (reqsBody.meta) setTotal(reqsBody.meta.total);
      }
    } catch {
      setError("Khong the tai du lieu");
    } finally {
      setLoading(false);
    }
  }, [baseUrl, page, leaveTypeId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitLoading(true);
    setError("");
    setSuccess("");

    try {
      const res = await fetch(`${baseUrl}/api/leave/my`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          leave_type_id: leaveTypeId,
          start_date: startDate,
          end_date: endDate,
          reason,
        }),
      });

      const body = await res.json();
      if (body.success) {
        setSuccess("Yeu cau nghi phep da duoc gui thanh cong!");
        setStartDate("");
        setEndDate("");
        setReason("");
        fetchData();
      } else {
        setError(body.error?.message || "Gui yeu cau that bai");
      }
    } catch {
      setError("Khong the ket noi den server");
    } finally {
      setSubmitLoading(false);
    }
  }

  return (
    <div className="min-h-full">
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="mb-8">
          <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight flex items-center gap-3">
            <CalendarOff className="w-8 h-8 text-primary-600" />
            Nghi phep
          </h1>
          <p className="mt-2 text-lg text-gray-500">
            Gui va theo doi cac yeu cau nghi phep cua ban
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
          {/* Form Section */}
          <div className="lg:col-span-1">
            <div className="bg-white rounded-3xl shadow-xl border border-gray-100 overflow-hidden sticky top-24">
              <div className="p-6 sm:p-8">
                <h2 className="text-xl font-bold text-gray-900 mb-6">
                  Dang ky nghi moi
                </h2>

                <form onSubmit={handleSubmit} className="space-y-5">
                  <div>
                    <label className="block text-sm font-bold text-gray-700 mb-2">
                      Loai nghi
                    </label>
                    <select
                      value={leaveTypeId}
                      onChange={(e) => setLeaveTypeId(e.target.value)}
                      required
                      className="block w-full rounded-2xl border-gray-200 bg-gray-50 py-3 px-4 text-gray-900 focus:border-primary-500 focus:ring-primary-500 sm:text-sm outline-none border focus:bg-white"
                    >
                      {leaveTypes.map((lt) => (
                        <option key={lt.id} value={lt.id}>
                          {lt.name}{" "}
                          {!lt.is_paid ? "(Khong luong)" : ""}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-bold text-gray-700 mb-2">
                        Tu ngay
                      </label>
                      <input
                        type="date"
                        value={startDate}
                        onChange={(e) => setStartDate(e.target.value)}
                        required
                        className="block w-full rounded-2xl border-gray-200 bg-gray-50 py-3 px-4 text-gray-900 focus:border-primary-500 focus:ring-primary-500 sm:text-sm outline-none border focus:bg-white"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-bold text-gray-700 mb-2">
                        Den ngay
                      </label>
                      <input
                        type="date"
                        value={endDate}
                        onChange={(e) => setEndDate(e.target.value)}
                        required
                        className="block w-full rounded-2xl border-gray-200 bg-gray-50 py-3 px-4 text-gray-900 focus:border-primary-500 focus:ring-primary-500 sm:text-sm outline-none border focus:bg-white"
                      />
                    </div>
                  </div>

                  <div>
                    <label className="block text-sm font-bold text-gray-700 mb-2">
                      Ly do
                    </label>
                    <textarea
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      rows={3}
                      placeholder="Ly do xin nghi..."
                      className="block w-full rounded-2xl border-gray-200 bg-gray-50 py-3 px-4 text-gray-900 focus:border-primary-500 focus:ring-primary-500 sm:text-sm outline-none border focus:bg-white"
                    />
                  </div>

                  <button
                    type="submit"
                    disabled={submitLoading}
                    className="w-full flex items-center justify-center gap-3 rounded-2xl bg-primary-600 px-6 py-4 text-base font-bold text-white shadow-lg shadow-primary-200 hover:bg-primary-700 active:scale-95 transition-all disabled:opacity-50"
                  >
                    {submitLoading && <span className="spinner" />}
                    Gui yeu cau
                  </button>
                </form>
              </div>
            </div>
          </div>

          {/* List Section */}
          <div className="lg:col-span-2">
            <div className="bg-white rounded-3xl shadow-sm border border-gray-100 overflow-hidden">
              <div className="p-6 border-b border-gray-50 flex justify-between items-center">
                <h2 className="text-xl font-bold text-gray-900">
                  Lich su yeu cau
                </h2>
                <span className="px-3 py-1 bg-gray-100 text-xs font-bold text-gray-500 rounded-full">
                  {total} yeu cau
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
                          Thoi gian
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                          Loai
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                          So ngay
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                          Trang thai
                        </th>
                        <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                          Nguoi duyet
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
                            Chua co yeu cau nghi phep nao.
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
                                {req.start_date}
                              </div>
                              <div className="text-xs text-gray-400">
                                den {req.end_date}
                              </div>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap">
                              <span
                                className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-bold"
                                style={{
                                  backgroundColor: req.leave_type?.color
                                    ? `${req.leave_type.color}15`
                                    : "#f3f4f6",
                                  color:
                                    req.leave_type?.color || "#6b7280",
                                }}
                              >
                                {req.leave_type?.name || "Nghi phep"}
                              </span>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm font-bold text-gray-700">
                              {req.total_days} ngay
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap">
                              <StatusBadge status={req.status} />
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                              {req.reviewer?.full_name || (
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

              {/* Simple pagination */}
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
                        Truoc
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
