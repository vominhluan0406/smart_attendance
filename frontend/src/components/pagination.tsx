"use client";

import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";

interface PaginationProps {
  page: number;
  total: number;
  limit: number;
}

export default function Pagination({ page, total, limit }: PaginationProps) {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const totalPages = Math.ceil(total / limit) || 1;
  const hasPrev = page > 1;
  const hasNext = page < totalPages;

  function buildUrl(targetPage: number) {
    const params = new URLSearchParams(searchParams.toString());
    params.set("page", String(targetPage));
    return `${pathname}?${params.toString()}`;
  }

  if (totalPages <= 1) return null;

  return (
    <div className="bg-gray-50/50 px-6 py-4 flex items-center justify-between border-t border-gray-50">
      <div>
        <p className="text-xs text-gray-500">
          Trang{" "}
          <span className="font-bold text-gray-900">{page}</span> /{" "}
          <span className="font-bold text-gray-900">{totalPages}</span>
        </p>
      </div>
      <div className="flex gap-2">
        {hasPrev && (
          <Link
            href={buildUrl(page - 1)}
            className="px-4 py-2 bg-white border border-gray-200 rounded-xl text-xs font-bold text-gray-600 hover:bg-gray-50 transition-all"
          >
            Trước
          </Link>
        )}
        {hasNext && (
          <Link
            href={buildUrl(page + 1)}
            className="px-4 py-2 bg-white border border-gray-200 rounded-xl text-xs font-bold text-gray-600 hover:bg-gray-50 transition-all"
          >
            Sau
          </Link>
        )}
      </div>
    </div>
  );
}
