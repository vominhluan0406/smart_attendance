import { redirect } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import Pagination from "@/components/pagination";
import StatusBadge from "@/components/status-badge";
import type { Attendance, Branch, PaginationMeta } from "@/lib/types";
import { FileText, Download, Search, ChevronRight } from "lucide-react";
import Link from "next/link";

interface PageProps {
  params: Promise<{ branchId: string }>;
  searchParams: Promise<{
    page?: string;
    date_from?: string;
    date_to?: string;
    status?: string;
  }>;
}

export default async function BranchReportPage({ params, searchParams }: PageProps) {
  const session = await getSession();
  if (!session) redirect("/login");

  const cookie = await getCookieHeader();
  const { branchId } = await params;
  const sp = await searchParams;
  const page = parseInt(sp.page || "1", 10);
  const dateFrom = sp.date_from || "";
  const dateTo = sp.date_to || "";
  const status = sp.status || "";

  let branch: Branch | null = null;
  let data: Attendance[] = [];
  let meta: PaginationMeta = { page: 1, limit: 20, total: 0 };
  let error = "";

  try {
    const branchRes = await apiGet<Branch>(`/api/branches/${branchId}`, cookie);
    if (branchRes.data) branch = branchRes.data;
  } catch {
    error = "Không tìm thấy chi nhánh";
  }

  if (branch) {
    try {
      const query = new URLSearchParams();
      query.set("page", String(page));
      query.set("limit", "20");
      query.set("branch_id", branchId);
      if (dateFrom) query.set("date_from", dateFrom);
      if (dateTo) query.set("date_to", dateTo);
      if (status) query.set("status", status);

      const res = await apiGet<Attendance[]>(
        `/api/reports/branch/${branchId}?${query.toString()}`,
        cookie
      );
      if (res.data) data = res.data;
      if (res.meta) meta = res.meta;
    } catch (e) {
      error = e instanceof Error ? e.message : "Không thể tải dữ liệu";
    }
  }

  const exportUrl = `/api/reports/branch/${branchId}/export?date_from=${dateFrom}&date_to=${dateTo}&status=${status}`;

  return (
    <div className="min-h-full">
      <Nav session={session} />

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <div className="mb-6 flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4">
          <div>
            <div className="flex items-center gap-2 text-sm text-primary-600 font-medium mb-1">
              {session.role === "admin" && (
                <>
                  <Link href="/reports" className="hover:underline">Báo cáo</Link>
                  <ChevronRight className="w-4 h-4" />
                </>
              )}
              <span>{branch?.name || "Chi nhánh"}</span>
            </div>
            <h1 className="text-2xl font-bold text-gray-900">
              {branch?.name || "Báo cáo chi nhánh"}
            </h1>
            <p className="text-sm text-gray-500 mt-1">Báo cáo chấm công chi nhánh</p>
          </div>
          <a
            href={exportUrl}
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
              <label className="block text-xs font-medium text-gray-700 mb-1">Từ ngày</label>
              <input
                type="date"
                name="date_from"
                defaultValue={dateFrom}
                className="block w-full rounded-lg border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm py-2 px-3 ring-1 ring-inset ring-gray-300"
              />
            </div>
            <div className="flex-1 min-w-[140px]">
              <label className="block text-xs font-medium text-gray-700 mb-1">Đến ngày</label>
              <input
                type="date"
                name="date_to"
                defaultValue={dateTo}
                className="block w-full rounded-lg border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm py-2 px-3 ring-1 ring-inset ring-gray-300"
              />
            </div>
            <div className="w-36">
              <label className="block text-xs font-medium text-gray-700 mb-1">Trạng thái</label>
              <select
                name="status"
                defaultValue={status}
                className="block w-full rounded-lg border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm py-2 px-3 ring-1 ring-inset ring-gray-300"
              >
                <option value="">Tất cả</option>
                <option value="on_time">Đúng giờ</option>
                <option value="late">Trễ</option>
                <option value="absent">Vắng</option>
                <option value="leave">Nghỉ phép</option>
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
                href={`/reports/branch/${branchId}`}
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

        {/* Table */}
        {data.length > 0 ? (
          <div className="bg-white rounded-3xl shadow-sm border border-gray-100 overflow-hidden">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-100">
                <thead className="bg-gray-50/50">
                  <tr>
                    <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">Ngày</th>
                    <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">Nhân viên</th>
                    <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">Giờ vào</th>
                    <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">Giờ ra</th>
                    <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">Trạng thái</th>
                    <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">Phương thức</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-50 bg-white">
                  {data.map((att) => (
                    <tr key={att.id} className="hover:bg-gray-50/50 transition-colors">
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-bold text-gray-900">
                        {att.work_date}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        {att.user ? (
                          <div>
                            <div className="text-sm font-bold text-gray-900">{att.user.full_name}</div>
                            <div className="text-xs text-gray-400">{att.user.email}</div>
                          </div>
                        ) : (
                          <span className="text-sm text-gray-400">{att.user_id}</span>
                        )}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {att.check_in_at
                          ? new Date(att.check_in_at).toLocaleTimeString("vi-VN", { hour: "2-digit", minute: "2-digit", second: "2-digit" })
                          : "-"}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {att.check_out_at
                          ? new Date(att.check_out_at).toLocaleTimeString("vi-VN", { hour: "2-digit", minute: "2-digit", second: "2-digit" })
                          : "-"}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <StatusBadge status={att.status} />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 uppercase">
                        {att.method}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        ) : (
          !error && (
            <div className="text-center py-16 bg-white rounded-3xl shadow-sm border border-gray-100">
              <FileText className="w-16 h-16 mx-auto mb-4 text-gray-200" />
              <p className="text-gray-400 text-lg font-medium">Chưa có dữ liệu chấm công</p>
            </div>
          )
        )}

        <Pagination page={meta.page} total={meta.total} limit={meta.limit} />
      </main>
    </div>
  );
}
