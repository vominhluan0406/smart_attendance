"use client";

import { useState } from "react";
import { Eye, Ban } from "lucide-react";

interface Props {
  alertId: string;
  isReviewed: boolean;
}

export default function AlertActions({ alertId, isReviewed }: Props) {
  const [reviewed, setReviewed] = useState(isReviewed);
  const [loading, setLoading] = useState(false);

  const baseUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  async function handleReview() {
    setLoading(true);
    try {
      const res = await fetch(`${baseUrl}/api/alerts/${alertId}/review`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
      });
      const body = await res.json();
      if (body.success) setReviewed(true);
    } catch {
      // Silently fail
    } finally {
      setLoading(false);
    }
  }

  async function handleInvalidate() {
    setLoading(true);
    try {
      const res = await fetch(
        `${baseUrl}/api/alerts/${alertId}/invalidate`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
        }
      );
      const body = await res.json();
      if (body.success) setReviewed(true);
    } catch {
      // Silently fail
    } finally {
      setLoading(false);
    }
  }

  if (reviewed) {
    return (
      <span className="text-xs font-bold text-green-600 uppercase tracking-wider">
        Da xem xet
      </span>
    );
  }

  return (
    <div className="flex justify-end gap-2">
      <button
        onClick={handleReview}
        disabled={loading}
        className="p-2 bg-blue-50 text-blue-600 rounded-xl hover:bg-blue-600 hover:text-white transition-all shadow-sm disabled:opacity-50"
        title="Xem xet"
      >
        <Eye className="w-4 h-4" />
      </button>
      <button
        onClick={handleInvalidate}
        disabled={loading}
        className="p-2 bg-red-50 text-red-600 rounded-xl hover:bg-red-600 hover:text-white transition-all shadow-sm disabled:opacity-50"
        title="Huy diem danh"
      >
        <Ban className="w-4 h-4" />
      </button>
    </div>
  );
}
