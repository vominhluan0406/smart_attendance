import { redirect } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import type { Branch } from "@/lib/types";
import { FileText, MapPin, ChevronRight } from "lucide-react";
import Link from "next/link";

export default async function AdminReportsPage() {
  const session = await getSession();
  if (!session) redirect("/login");

  // Manager → redirect to their branch report
  if (session.role === "manager" && session.branchId) {
    redirect(`/reports/branch/${session.branchId}`);
  }

  // Employee → redirect to personal history
  if (session.role === "employee") {
    redirect("/reports/my-history");
  }

  const cookie = await getCookieHeader();
  let branches: Branch[] = [];
  let error = "";

  try {
    const res = await apiGet<Branch[]>("/api/branches?limit=100", cookie);
    if (res.data) branches = res.data;
  } catch (e) {
    error = e instanceof Error ? e.message : "Không thể tải dữ liệu";
  }

  return (
    <div className="min-h-full">
      <Nav session={session} />

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <div className="mb-6 flex items-center gap-3">
          <FileText className="w-6 h-6 text-primary-600" />
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Báo cáo chấm công</h1>
            <p className="text-sm text-gray-500 mt-1">Chọn chi nhánh để xem báo cáo</p>
          </div>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 rounded-xl p-4 mb-6">
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {branches.map((branch) => (
            <Link
              key={branch.id}
              href={`/reports/branch/${branch.id}`}
              className="group bg-white rounded-2xl shadow-sm border border-gray-100 p-6 hover:shadow-md hover:border-primary-200 transition-all"
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-primary-50 rounded-xl group-hover:bg-primary-100 transition-colors">
                    <MapPin className="w-5 h-5 text-primary-600" />
                  </div>
                  <div>
                    <h3 className="text-lg font-bold text-gray-900 group-hover:text-primary-600 transition-colors">
                      {branch.name}
                    </h3>
                    {branch.address && (
                      <p className="text-sm text-gray-400 mt-1 line-clamp-1">{branch.address}</p>
                    )}
                  </div>
                </div>
                <ChevronRight className="w-5 h-5 text-gray-300 group-hover:text-primary-400 transition-colors" />
              </div>
              <div className="mt-4 flex items-center gap-2">
                <span
                  className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold ${
                    branch.is_active
                      ? "bg-green-50 text-green-700"
                      : "bg-gray-100 text-gray-500"
                  }`}
                >
                  {branch.is_active ? "Hoạt động" : "Tắt"}
                </span>
                <span className="text-xs text-gray-400">
                  {branch.allowed_methods.split(",").length} phương thức
                </span>
              </div>
            </Link>
          ))}
        </div>

        {branches.length === 0 && !error && (
          <div className="text-center py-16 bg-white rounded-3xl shadow-sm border border-gray-100">
            <MapPin className="w-16 h-16 mx-auto mb-4 text-gray-200" />
            <p className="text-gray-400 text-lg font-medium">Chưa có chi nhánh nào</p>
          </div>
        )}
      </main>
    </div>
  );
}
