"use client";

import DataTable, { Column } from "@/components/data-table";
import type { Branch } from "@/lib/types";
import { Pencil } from "lucide-react";
import Link from "next/link";

const methodLabels: Record<string, string> = {
  qr_totp: "QR",
  ip: "IP",
  location: "GPS",
  face: "Face",
  password: "Password",
  wifi_gps: "WiFi/GPS",
  nfc: "NFC",
  ble: "BLE",
};

const columns: Column<Branch>[] = [
  {
    key: "name",
    header: "Tên chi nhánh",
    render: (item) => (
      <div>
        <div className="text-sm font-bold text-gray-900">{item.name}</div>
        <div className="text-xs text-gray-400">{item.address}</div>
      </div>
    ),
  },
  {
    key: "methods",
    header: "Phương thức",
    render: (item) => (
      <div className="flex flex-wrap gap-1">
        {item.allowed_methods?.split(",").map((m) => (
          <span
            key={m}
            className="text-xs bg-primary-50 text-primary-700 px-2 py-0.5 rounded font-bold"
          >
            {methodLabels[m.trim()] || m.trim()}
          </span>
        ))}
      </div>
    ),
  },
  {
    key: "status",
    header: "Trạng thái",
    render: (item) => (
      <span
        className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-bold ${
          item.is_active
            ? "bg-green-50 text-green-700"
            : "bg-gray-50 text-gray-500"
        }`}
      >
        {item.is_active ? "Hoạt động" : "Tắt"}
      </span>
    ),
  },
  {
    key: "actions",
    header: "",
    headerClassName: "text-right",
    className: "text-right",
    render: (item) => (
      <Link
        href={`/branches/${item.id}/edit`}
        className="inline-flex items-center gap-1 px-3 py-1.5 bg-gray-100 text-gray-700 rounded-lg text-xs font-bold hover:bg-gray-200 transition-colors"
      >
        <Pencil className="w-3 h-3" />
        Sửa
      </Link>
    ),
  },
];

interface BranchesTableProps {
  branches: Branch[];
}

export default function BranchesTable({ branches }: BranchesTableProps) {
  return (
    <DataTable
      columns={columns}
      data={branches}
      emptyMessage="Chưa có chi nhánh nào."
      keyExtractor={(item) => item.id}
    />
  );
}
