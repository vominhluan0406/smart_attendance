import React from "react";

type BadgeVariant =
  | "on_time"
  | "late"
  | "absent"
  | "leave"
  | "invalidated"
  | "pending"
  | "approved"
  | "rejected"
  | "cancelled"
  | "critical"
  | "warning";

const variantStyles: Record<BadgeVariant, string> = {
  on_time:
    "bg-green-50 text-green-700 ring-1 ring-inset ring-green-600/20",
  late: "bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20",
  absent: "bg-red-50 text-red-700 ring-1 ring-inset ring-red-600/20",
  leave: "bg-purple-50 text-purple-700 ring-1 ring-inset ring-purple-600/20",
  invalidated:
    "bg-gray-50 text-gray-600 ring-1 ring-inset ring-gray-500/10",
  pending:
    "bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20",
  approved:
    "bg-green-50 text-green-700 ring-1 ring-inset ring-green-600/20",
  rejected: "bg-red-50 text-red-700 ring-1 ring-inset ring-red-600/20",
  cancelled:
    "bg-gray-50 text-gray-600 ring-1 ring-inset ring-gray-500/10",
  critical: "bg-red-50 text-red-700 ring-1 ring-inset ring-red-600/20",
  warning:
    "bg-amber-50 text-amber-700 ring-1 ring-inset ring-amber-600/20",
};

const variantLabels: Record<BadgeVariant, string> = {
  on_time: "Dung gio",
  late: "Tre",
  absent: "Vang",
  leave: "Nghi phep",
  invalidated: "Huy",
  pending: "Dang cho",
  approved: "Da duyet",
  rejected: "Tu choi",
  cancelled: "Da huy",
  critical: "Nghiem trong",
  warning: "Canh bao",
};

interface StatusBadgeProps {
  status: string;
  label?: string;
  className?: string;
}

export default function StatusBadge({
  status,
  label,
  className = "",
}: StatusBadgeProps) {
  const variant = status as BadgeVariant;
  const styles = variantStyles[variant] || variantStyles.invalidated;
  const displayLabel = label || variantLabels[variant] || status;

  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-bold ${styles} ${className}`}
    >
      {displayLabel}
    </span>
  );
}
