import { redirect } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import MyHistoryTable from "./my-history-table";
import Pagination from "@/components/pagination";
import type { Attendance, PaginationMeta } from "@/lib/types";
import { History, Download, Search } from "lucide-react";

interface SearchParams {
  page?: string;
  date_from?: string;
  date_to?: string;
  status?: string;
}

export default async function MyHistoryPage({
  searchParams,
}: {
  searchParams: Promise<SearchParams>;
}) {
  const session = await getSession();
  if (!session) redirect("/login");

  const cookie = await getCookieHeader();
  const params = await searchParams;
  const page = parseInt(params.page || "1", 10);
  const dateFrom = params.date_from || "";
  const dateTo = params.date_to || "";
  const status = params.status || "";

  let data: Attendance[] = [];
  let meta: PaginationMeta = { page: 1, limit: 20, total: 0 };
  let error = "";

  try {
    const query = new URLSearchParams();
    query.set("page", String(page));
    query.set("limit", "20");
    if (dateFrom) query.set("date_from", dateFrom);
    if (dateTo) query.set("date_to", dateTo);
    if (status) query.set("status", status);

    const res = await apiGet<Attendance[]>(
      `/api/reports/my-history?${query.toString()}`,
      cookie
    );
    if (res.data) data = res.data;
    if (res.meta) meta = res.meta;
  } catch (e) {
    error = e instanceof Error ? e.message : "Không thể tải dữ liệu";
  }



  return (
    <div className="min-h-full">
      <Nav session={session} />

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <div className="mb-6 flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4">
          <div className="flex items-center gap-3">
            <History className="w-6 h-6 text-primary-600" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">
                Lịch sử chấm công
              </h1>
              <p className="text-sm text-gray-500 mt-1">
                Xem lịch sử check-in / check-out của bạn
              </p>
            </div>
          </div>
          <a
            href={`/api/reports/my-history/export?date_from=${dateFrom}&date_to=${dateTo}&status=${status}`}
            className="hidden sm:inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-primary-600 text-sm font-bold text-white shadow-sm hover:bg-primary-700 transition-all"
          >
            <Download className="w-4 h-4" />
            Xuất Excel
          </a>
        </div>

        {/* Filters */}
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-4 mb-6">
          <form className="flex flex-wrap gap-4 items-end">
            <div className="flex-1 min-w-[140px]">
              <label className="block text-xs font-medium text-gray-700 mb-1">
                Từ ngày
              </label>
              <input
                type="date"
                name="date_from"
                defaultValue={dateFrom}
                className="block w-full rounded-lg border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm py-2 px-3 ring-1 ring-inset ring-gray-300"
              />
            </div>
            <div className="flex-1 min-w-[140px]">
              <label className="block text-xs font-medium text-gray-700 mb-1">
                Đến ngày
              </label>
              <input
                type="date"
                name="date_to"
                defaultValue={dateTo}
                className="block w-full rounded-lg border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm py-2 px-3 ring-1 ring-inset ring-gray-300"
              />
            </div>
            <div className="w-36">
              <label className="block text-xs font-medium text-gray-700 mb-1">
                Trạng thái
              </label>
              <select
                name="status"
                defaultValue={status}
                className="block w-full rounded-lg border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm py-2 px-3 ring-1 ring-inset ring-gray-300"
              >
                <option value="">Tất cả</option>
                <option value="on_time">Đúng giờ</option>
                <option value="late">Trễ</option>
                <option value="absent">Vắng</option>
              </select>
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                className="flex items-center gap-2 rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200 transition-colors"
              >
                <Search className="w-4 h-4" />
                Lọc
              </button>
              <a
                href="/reports/my-history"
                className="rounded-lg bg-white border border-gray-200 px-4 py-2 text-sm font-medium text-gray-500 hover:bg-gray-50 transition-colors"
              >
                Xoá lọc
              </a>
            </div>
          </form>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 rounded-xl p-4 mb-6">
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        <MyHistoryTable data={data} />

        <Pagination page={meta.page} total={meta.total} limit={meta.limit} />
      </main>
    </div>
  );
}
