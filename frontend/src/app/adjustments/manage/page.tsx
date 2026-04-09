import { redirect } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import type { AdjustmentRequest } from "@/lib/types";
import { Clock } from "lucide-react";
import AdjustmentManageTable from "./adjustment-manage-table";

interface SearchParams {
  status?: string;
}

export default async function ManageAdjustmentsPage({
  searchParams,
}: {
  searchParams: Promise<SearchParams>;
}) {
  const session = await getSession();
  if (!session) redirect("/login");
  if (session.role !== "admin" && session.role !== "manager") redirect("/");

  const cookie = await getCookieHeader();
  const params = await searchParams;
  const statusFilter = params.status || "pending";

  let requests: AdjustmentRequest[] = [];
  let error = "";

  try {
    const res = await apiGet<AdjustmentRequest[]>(
      `/api/adjustments/manage?status=${statusFilter}&limit=50`,
      cookie
    );
    if (res.data) requests = res.data;
  } catch (e) {
    error = e instanceof Error ? e.message : "Không thể tải dữ liệu";
  }

  return (
    <div className="min-h-full">
      <Nav session={session} />

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="mb-8 flex flex-col sm:flex-row sm:justify-between sm:items-end gap-4">
          <div>
            <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight flex items-center gap-3">
              <Clock className="w-8 h-8 text-primary-600" />
              Duyệt bổ sung công
            </h1>
            <p className="mt-2 text-lg text-gray-500">
              Xét duyệt yêu cầu chỉnh sửa giờ chấm công của nhân viên
            </p>
          </div>

          <div className="flex bg-gray-100 p-1 rounded-2xl w-max">
            {["pending", "approved", "rejected"].map((s) => (
              <a
                key={s}
                href={`?status=${s}`}
                className={`px-4 py-2 rounded-xl text-sm font-bold transition-all ${
                  statusFilter === s
                    ? "bg-white text-primary-600 shadow-sm"
                    : "text-gray-500 hover:text-gray-700"
                }`}
              >
                {s === "pending"
                  ? "Chờ duyệt"
                  : s === "approved"
                  ? "Đã duyệt"
                  : "Từ chối"}
              </a>
            ))}
          </div>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 rounded-xl p-4 mb-6">
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        <AdjustmentManageTable
          requests={requests}
          statusFilter={statusFilter}
        />
      </main>
    </div>
  );
}
