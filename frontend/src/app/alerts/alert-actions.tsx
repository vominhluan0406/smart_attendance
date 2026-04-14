"use client";

import { useState } from "react";
import { Eye, Ban, AlertCircle } from "lucide-react";
import { apiPost } from "@/lib/api";

interface Props {
  alertId: string;
  isReviewed: boolean;
}

export default function AlertActions({ alertId, isReviewed }: Props) {
  const [reviewed, setReviewed] = useState(isReviewed);
  const [invalidated, setInvalidated] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleReview() {
    setLoading(true);
    setError(null);
    try {
      const res = await apiPost(`/api/alerts/${alertId}/review`, {});
      if (res.success) setReviewed(true);
      else setError(res.error?.message || "Xem xét thất bại");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Không thể kết nối");
    } finally {
      setLoading(false);
    }
  }

  async function handleInvalidate() {
    setLoading(true);
    setError(null);
    try {
      const res = await apiPost(`/api/alerts/${alertId}/invalidate`, {});
      if (res.success) {
        setReviewed(true);
        setInvalidated(true);
      } else {
        setError(res.error?.message || "Huỷ điểm danh thất bại");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Không thể kết nối");
    } finally {
      setLoading(false);
    }
  }

  if (reviewed) {
    return (
      <span className={`text-xs font-bold uppercase tracking-wider ${invalidated ? "text-red-600" : "text-green-600"}`}>
        {invalidated ? "Đã huỷ công" : "Đã xem xét"}
      </span>
    );
  }

  return (
    <div>
      {error && (
        <div className="mb-2 p-2 bg-red-50 border border-red-100 rounded-lg flex items-center gap-1 text-xs text-red-700">
          <AlertCircle className="w-3 h-3 flex-shrink-0" />
          <span>{error}</span>
        </div>
      )}
      <div className="flex justify-end gap-2">
        <button
          onClick={handleReview}
          disabled={loading}
          className="p-2 bg-blue-50 text-blue-600 rounded-xl hover:bg-blue-600 hover:text-white transition-all shadow-sm disabled:opacity-50"
          title="Xem xét"
        >
          <Eye className="w-4 h-4" />
        </button>
        <button
          onClick={handleInvalidate}
          disabled={loading}
          className="p-2 bg-red-50 text-red-600 rounded-xl hover:bg-red-600 hover:text-white transition-all shadow-sm disabled:opacity-50"
          title="Huỷ điểm danh"
        >
          <Ban className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}
