import { redirect } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import StatusBadge from "@/components/status-badge";
import Pagination from "@/components/pagination";
import type { FraudAlert, PaginationMeta } from "@/lib/types";
import { ShieldAlert, Search } from "lucide-react";
import AlertActions from "./alert-actions";

interface SearchParams {
  page?: string;
  date_from?: string;
  date_to?: string;
  alert_type?: string;
  severity?: string;
  reviewed?: string;
}

export default async function AlertsPage({
  searchParams,
}: {
  searchParams: Promise<SearchParams>;
}) {
  const session = await getSession();
  if (!session) redirect("/login");
  if (session.role !== "admin" && session.role !== "manager") redirect("/");

  const cookie = await getCookieHeader();
  const params = await searchParams;
  const page = parseInt(params.page || "1", 10);
  const dateFrom = params.date_from || "";
  const dateTo = params.date_to || "";
  const alertType = params.alert_type || "";
  const severity = params.severity || "";
  const reviewed = params.reviewed || "";

  let alerts: FraudAlert[] = [];
  let meta: PaginationMeta = { page: 1, limit: 20, total: 0 };
  let error = "";

  try {
    const query = new URLSearchParams();
    query.set("page", String(page));
    query.set("limit", "20");
    if (dateFrom) query.set("date_from", dateFrom);
    if (dateTo) query.set("date_to", dateTo);
    if (alertType) query.set("alert_type", alertType);
    if (severity) query.set("severity", severity);
    if (reviewed) query.set("reviewed", reviewed);

    const res = await apiGet<FraudAlert[]>(
      `/api/alerts?${query.toString()}`,
      cookie
    );
    if (res.data) alerts = res.data;
    if (res.meta) meta = res.meta;
  } catch (e) {
    error = e instanceof Error ? e.message : "Khong the tai du lieu";
  }

  const alertTypeLabels: Record<string, string> = {
    gps_accuracy: "GPS gia mao",
    totp_reuse: "QR tai su dung",
    impossible_travel: "Di chuyen bat thuong",
    new_device: "Thiet bi moi",
    ip_location_mismatch: "IP/GPS lech",
    cloned_authenticator: "Clone authenticator",
    anomaly_time: "Thoi gian bat thuong",
    concurrent_session: "Da phien",
  };

  return (
    <div className="min-h-full">
      <Nav session={session} />

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-3">
            <ShieldAlert className="w-6 h-6 text-primary-600" />
            Canh bao gian lan
          </h1>
          <p className="text-sm text-gray-500 mt-1">
            Phat hien hanh vi bat thuong khi cham cong
          </p>
        </div>

        {/* Filters */}
        <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-4 mb-6">
          <form className="flex flex-wrap gap-4 items-end">
            <div className="flex-1 min-w-[140px]">
              <label className="block text-xs font-medium text-gray-700 mb-1">
                Tu ngay
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
                Den ngay
              </label>
              <input
                type="date"
                name="date_to"
                defaultValue={dateTo}
                className="block w-full rounded-lg border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm py-2 px-3 ring-1 ring-inset ring-gray-300"
              />
            </div>
            <div className="w-44">
              <label className="block text-xs font-medium text-gray-700 mb-1">
                Loai canh bao
              </label>
              <select
                name="alert_type"
                defaultValue={alertType}
                className="block w-full rounded-lg border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm py-2 px-3 ring-1 ring-inset ring-gray-300"
              >
                <option value="">Tat ca</option>
                {Object.entries(alertTypeLabels).map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </div>
            <div className="w-32">
              <label className="block text-xs font-medium text-gray-700 mb-1">
                Muc do
              </label>
              <select
                name="severity"
                defaultValue={severity}
                className="block w-full rounded-lg border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm py-2 px-3 ring-1 ring-inset ring-gray-300"
              >
                <option value="">Tat ca</option>
                <option value="critical">Nghiem trong</option>
                <option value="warning">Canh bao</option>
              </select>
            </div>
            <div className="w-36">
              <label className="block text-xs font-medium text-gray-700 mb-1">
                Trang thai
              </label>
              <select
                name="reviewed"
                defaultValue={reviewed}
                className="block w-full rounded-lg border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 text-sm py-2 px-3 ring-1 ring-inset ring-gray-300"
              >
                <option value="">Tat ca</option>
                <option value="false">Chua xem xet</option>
                <option value="true">Da xem xet</option>
              </select>
            </div>
            <div className="flex gap-2">
              <button
                type="submit"
                className="flex items-center gap-2 rounded-lg bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200 transition-colors"
              >
                <Search className="w-4 h-4" />
                Loc
              </button>
              <a
                href="/alerts"
                className="rounded-lg bg-white border border-gray-200 px-4 py-2 text-sm font-medium text-gray-500 hover:bg-gray-50 transition-colors"
              >
                Xoa loc
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
        <div className="bg-white rounded-3xl shadow-sm border border-gray-100 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-100">
              <thead className="bg-gray-50/50">
                <tr>
                  <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                    Nhan vien
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                    Loai
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                    Muc do
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                    Mo ta
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                    IP
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-bold text-gray-400 uppercase tracking-wider">
                    Thoi gian
                  </th>
                  <th className="px-6 py-4 text-right text-xs font-bold text-gray-400 uppercase tracking-wider">
                    Thao tac
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50 bg-white">
                {alerts.length === 0 ? (
                  <tr>
                    <td
                      colSpan={7}
                      className="px-6 py-16 text-center text-gray-400 italic text-sm"
                    >
                      Khong co canh bao nao.
                    </td>
                  </tr>
                ) : (
                  alerts.map((alert) => (
                    <tr
                      key={alert.id}
                      className="hover:bg-gray-50/50 transition-colors"
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-primary-50 flex items-center justify-center text-primary-600 font-bold text-xs">
                            {alert.user?.full_name?.charAt(0) || "?"}
                          </div>
                          <div>
                            <div className="text-sm font-bold text-gray-900">
                              {alert.user?.full_name || "N/A"}
                            </div>
                            <div className="text-xs text-gray-400">
                              {alert.user?.email}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="text-xs font-bold text-gray-700 bg-gray-100 px-2 py-1 rounded">
                          {alertTypeLabels[alert.alert_type] ||
                            alert.alert_type}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <StatusBadge status={alert.severity} />
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-600 max-w-xs">
                        <p className="line-clamp-2">{alert.description}</p>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-xs font-mono text-gray-500">
                        {alert.ip_address}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-xs text-gray-500">
                        {new Date(alert.created_at).toLocaleString("vi-VN")}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right">
                        <AlertActions
                          alertId={alert.id}
                          isReviewed={alert.is_reviewed}
                        />
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          <Pagination page={meta.page} total={meta.total} limit={meta.limit} />
        </div>
      </main>
    </div>
  );
}
