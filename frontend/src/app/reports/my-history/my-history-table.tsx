"use client";

import DataTable, { Column } from "@/components/data-table";
import StatusBadge from "@/components/status-badge";
import type { Attendance } from "@/lib/types";

function formatTime(isoString?: string): string {
  if (!isoString) return "--:--";
  try {
    return new Date(isoString).toLocaleTimeString("vi-VN", {
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return "--:--";
  }
}

const columns: Column<Attendance>[] = [
  {
    key: "work_date",
    header: "Ngày",
    render: (item) => (
      <span className="text-sm font-bold text-gray-900">
        {item.work_date}
      </span>
    ),
  },
  {
    key: "check_in",
    header: "Giờ vào",
    render: (item) => (
      <span className="text-sm font-mono font-bold text-primary-600">
        {formatTime(item.check_in_at)}
      </span>
    ),
  },
  {
    key: "check_out",
    header: "Giờ ra",
    render: (item) => (
      <span className="text-sm font-mono font-bold text-gray-700">
        {formatTime(item.check_out_at)}
      </span>
    ),
  },
  {
    key: "status",
    header: "Trạng thái",
    render: (item) => <StatusBadge status={item.status} />,
  },
  {
    key: "method",
    header: "Phương thức",
    render: (item) => {
      const badges = [];
      if (item.totp_verified)
        badges.push(
          <span
            key="qr"
            className="text-xs bg-blue-50 text-blue-600 px-1.5 py-0.5 rounded"
          >
            QR
          </span>
        );
      if (item.ip_verified)
        badges.push(
          <span
            key="ip"
            className="text-xs bg-green-50 text-green-600 px-1.5 py-0.5 rounded"
          >
            IP
          </span>
        );
      if (item.loc_verified)
        badges.push(
          <span
            key="gps"
            className="text-xs bg-purple-50 text-purple-600 px-1.5 py-0.5 rounded"
          >
            GPS
          </span>
        );
      if (item.face_verified)
        badges.push(
          <span
            key="face"
            className="text-xs bg-violet-50 text-violet-600 px-1.5 py-0.5 rounded"
          >
            Face
          </span>
        );
      if (item.password_verified)
        badges.push(
          <span
            key="pass"
            className="text-xs bg-amber-50 text-amber-600 px-1.5 py-0.5 rounded"
          >
            Pass
          </span>
        );
      return <div className="flex gap-1">{badges}</div>;
    },
  },
];

interface MyHistoryTableProps {
  data: Attendance[];
}

export default function MyHistoryTable({ data }: MyHistoryTableProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      emptyMessage="Chưa có lịch sử chấm công."
      keyExtractor={(item) => item.id}
    />
  );
}
