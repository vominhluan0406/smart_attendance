"use client";

import DataTable, { Column } from "@/components/data-table";
import type { User } from "@/lib/types";
import { Pencil } from "lucide-react";
import Link from "next/link";

const roleLabels: Record<string, string> = {
  admin: "Admin",
  manager: "Quản lý",
  manager_device: "Manager Máy",
  employee: "Nhân viên",
};

const roleStyles: Record<string, string> = {
  admin: "bg-purple-50 text-purple-700",
  manager: "bg-blue-50 text-blue-700",
  manager_device: "bg-cyan-50 text-cyan-700",
  employee: "bg-gray-50 text-gray-700",
};

const columns: Column<User>[] = [
  {
    key: "name",
    header: "Họ tên",
    render: (item) => (
      <div className="flex items-center gap-3">
        <div className="w-10 h-10 rounded-full bg-primary-50 flex items-center justify-center text-primary-600 font-bold">
          {item.full_name?.charAt(0) || "?"}
        </div>
        <div>
          <div className="text-sm font-bold text-gray-900">
            {item.full_name}
          </div>
          <div className="text-xs text-gray-400">{item.email}</div>
        </div>
      </div>
    ),
  },
  {
    key: "role",
    header: "Vai trò",
    render: (item) => (
      <span
        className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-bold ${
          roleStyles[item.role] || roleStyles.employee
        }`}
      >
        {roleLabels[item.role] || item.role}
      </span>
    ),
  },
  {
    key: "department",
    header: "Phòng ban",
    render: (item) => (
      <span className="text-sm text-gray-600">
        {item.department?.name || (
          <span className="text-gray-300">---</span>
        )}
      </span>
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
            : "bg-red-50 text-red-700"
        }`}
      >
        {item.is_active ? "Hoạt động" : "Ngừng"}
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
        href={`/users/${item.id}/edit`}
        className="inline-flex items-center gap-1 px-3 py-1.5 bg-gray-100 text-gray-700 rounded-lg text-xs font-bold hover:bg-gray-200 transition-colors"
      >
        <Pencil className="w-3 h-3" />
        Sửa
      </Link>
    ),
  },
];

interface UsersTableProps {
  users: User[];
}

export default function UsersTable({ users }: UsersTableProps) {
  return (
    <DataTable
      columns={columns}
      data={users}
      emptyMessage="Chưa có nhân viên nào."
      keyExtractor={(item) => item.id}
    />
  );
}
