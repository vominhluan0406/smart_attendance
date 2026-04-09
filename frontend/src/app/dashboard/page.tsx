import { redirect } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import StatusBadge from "@/components/status-badge";
import type {
  DashboardStats,
  RecentActivity,
  TopLateUser,
  Branch,
} from "@/lib/types";
import {
  LayoutDashboard,
  Users,
  CheckCircle,
  Clock,
  AlertTriangle,
  CalendarOff,
  Activity,
  RefreshCw,
} from "lucide-react";

interface DashboardSearchParams {
  branch_id?: string;
}

export default async function DashboardPage({
  searchParams,
}: {
  searchParams: Promise<DashboardSearchParams>;
}) {
  const session = await getSession();
  if (!session) redirect("/login");
  if (session.role !== "admin" && session.role !== "manager") redirect("/");

  const cookie = await getCookieHeader();
  const params = await searchParams;
  const branchFilter = params.branch_id || "";

  let stats: DashboardStats | null = null;
  let recent: RecentActivity[] = [];
  let topLate: TopLateUser[] = [];
  let branches: Branch[] = [];
  let error = "";

  try {
    const query = branchFilter ? `?branch_id=${branchFilter}` : "";

    const [statsRes, recentRes, topLateRes] = await Promise.all([
      apiGet<DashboardStats>(`/api/dashboard/stats${query}`, cookie).catch(
        () => null
      ),
      apiGet<RecentActivity[]>(
        `/api/dashboard/recent${query}`,
        cookie
      ).catch(() => null),
      apiGet<TopLateUser[]>(
        `/api/dashboard/top-late${query}`,
        cookie
      ).catch(() => null),
    ]);

    if (statsRes?.data) stats = statsRes.data;
    if (recentRes?.data) recent = recentRes.data;
    if (topLateRes?.data) topLate = topLateRes.data;

    if (session.role === "admin") {
      const branchRes = await apiGet<Branch[]>(
        "/api/branches",
        cookie
      ).catch(() => null);
      if (branchRes?.data) branches = branchRes.data;
    }
  } catch (e) {
    error = e instanceof Error ? e.message : "Khong the tai du lieu";
  }

  const kpiCards = [
    {
      label: "Tong nhan vien",
      value: stats?.total_employees ?? 0,
      icon: Users,
      accent: "card-accent-indigo",
      iconColor: "text-primary-600",
      iconBg: "bg-primary-50",
    },
    {
      label: "Check-in hom nay",
      value: stats?.today_checkins ?? 0,
      icon: CheckCircle,
      accent: "card-accent-emerald",
      iconColor: "text-emerald-600",
      iconBg: "bg-emerald-50",
    },
    {
      label: "Di tre hom nay",
      value: stats?.late_count ?? 0,
      icon: Clock,
      accent: "card-accent-amber",
      iconColor: "text-amber-600",
      iconBg: "bg-amber-50",
    },
    {
      label: "Vang hom nay",
      value: stats?.absent_count ?? 0,
      icon: AlertTriangle,
      accent: "card-accent-rose",
      iconColor: "text-rose-600",
      iconBg: "bg-rose-50",
    },
    {
      label: "Cho duyet nghi phep",
      value: stats?.pending_leave ?? 0,
      icon: CalendarOff,
      accent: "",
      iconColor: "text-purple-600",
      iconBg: "bg-purple-50",
    },
    {
      label: "Canh bao gian lan",
      value: stats?.fraud_alerts_today ?? 0,
      icon: AlertTriangle,
      accent: "",
      iconColor: "text-red-600",
      iconBg: "bg-red-50",
    },
  ];

  return (
    <div className="min-h-full bg-mesh pb-12">
      <Nav session={session} />

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 mb-10">
          <div className="flex items-center gap-4">
            <div className="p-4 bg-white rounded-3xl shadow-sm border border-primary-100 animate-float">
              <LayoutDashboard className="w-8 h-8 text-primary-600" />
            </div>
            <div>
              <h1 className="text-3xl font-black text-gray-900 tracking-tight">
                Tong quan
              </h1>
              <p className="text-sm font-bold text-gray-400 uppercase tracking-widest mt-1">
                He thong quan ly cham cong
              </p>
            </div>
          </div>

          {session.role === "admin" && branches.length > 0 && (
            <div className="flex flex-col gap-2">
              <label className="text-[10px] font-black text-gray-400 uppercase tracking-widest ml-1">
                Loc theo chi nhanh
              </label>
              <form>
                <select
                  name="branch_id"
                  defaultValue={branchFilter}
                  onChange={(e) => {
                    const form = e.target.closest("form");
                    if (form) form.submit();
                  }}
                  className="appearance-none rounded-2xl border-gray-100 text-sm font-bold focus:ring-primary-500 focus:border-primary-500 py-3 pl-4 pr-10 bg-white shadow-sm hover:shadow transition-all min-w-[240px] ring-1 ring-inset ring-gray-200"
                >
                  <option value="">Tat ca chi nhanh</option>
                  {branches.map((b) => (
                    <option key={b.id} value={b.id}>
                      {b.name}
                    </option>
                  ))}
                </select>
              </form>
            </div>
          )}
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 rounded-xl p-4 mb-6">
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        {/* KPI Cards */}
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4 mb-8">
          {kpiCards.map((card) => (
            <div
              key={card.label}
              className={`bg-white rounded-3xl shadow-sm border border-gray-100 p-5 glass hover-lift ${card.accent}`}
            >
              <div
                className={`p-2 ${card.iconBg} rounded-xl w-max mb-3`}
              >
                <card.icon className={`w-5 h-5 ${card.iconColor}`} />
              </div>
              <p className="text-2xl font-black text-gray-900">
                {card.value}
              </p>
              <p className="text-[10px] font-bold text-gray-400 uppercase tracking-widest mt-1">
                {card.label}
              </p>
            </div>
          ))}
        </div>

        {/* On-time rate bar */}
        {stats && (
          <div className="bg-white rounded-3xl shadow-sm border border-gray-100 p-6 glass mb-6">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm font-bold text-gray-700">
                Ty le dung gio
              </span>
              <span className="text-lg font-black text-primary-600">
                {stats.on_time_rate?.toFixed(1) ?? 0}%
              </span>
            </div>
            <div className="w-full bg-gray-100 rounded-full h-3">
              <div
                className="bg-primary-600 h-3 rounded-full transition-all duration-500"
                style={{
                  width: `${Math.min(stats.on_time_rate ?? 0, 100)}%`,
                }}
              />
            </div>
          </div>
        )}

        {/* Recent Activity + Top Late */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Recent Activity (2/3) */}
          <div className="lg:col-span-2 bg-white rounded-3xl shadow-sm border border-gray-100 p-6 glass">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-lg font-black text-gray-900 flex items-center gap-2 uppercase tracking-tight">
                <Activity className="w-5 h-5 text-primary-600" />
                Hoat dong gan day
              </h2>
              <RefreshCw className="w-4 h-4 text-gray-400" />
            </div>

            {recent.length === 0 ? (
              <p className="text-center text-sm text-gray-400 py-8">
                Chua co hoat dong nao hom nay.
              </p>
            ) : (
              <div className="space-y-3">
                {recent.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center justify-between p-3 rounded-2xl hover:bg-gray-50/50 transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-9 h-9 rounded-xl bg-primary-50 flex items-center justify-center text-primary-600 font-bold text-xs">
                        {item.user_name?.charAt(0) || "?"}
                      </div>
                      <div>
                        <p className="text-sm font-bold text-gray-900">
                          {item.user_name}
                        </p>
                        <p className="text-xs text-gray-400">
                          {item.time} - {item.method}
                        </p>
                      </div>
                    </div>
                    <StatusBadge status={item.status} />
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Top Late (1/3) */}
          <div className="bg-white rounded-3xl shadow-sm border border-gray-100 p-6 glass">
            <h2 className="text-lg font-black text-gray-900 flex items-center gap-2 mb-6 uppercase tracking-tight">
              <Clock className="w-5 h-5 text-amber-500" />
              Top di tre
            </h2>

            {topLate.length === 0 ? (
              <div className="text-center py-12 bg-emerald-50/30 rounded-3xl border border-dashed border-emerald-100">
                <CheckCircle className="w-6 h-6 text-emerald-500 mx-auto mb-3" />
                <p className="text-[10px] text-emerald-600 font-black uppercase tracking-widest">
                  Khong co ai di tre!
                </p>
              </div>
            ) : (
              <div className="space-y-3">
                {topLate.map((user, i) => (
                  <div
                    key={user.email}
                    className="flex items-center justify-between p-4 rounded-2xl bg-gray-50/50 border border-gray-100/50 hover:bg-white hover:shadow-md transition-all group"
                  >
                    <div className="flex items-center gap-3">
                      <div className="flex items-center justify-center w-9 h-9 rounded-xl bg-amber-50 text-amber-600 text-xs font-black border border-amber-100 group-hover:bg-amber-600 group-hover:text-white transition-colors">
                        {i + 1}
                      </div>
                      <div>
                        <p className="text-sm font-bold text-gray-900 leading-none mb-1">
                          {user.full_name}
                        </p>
                        <p className="text-[10px] text-gray-400 font-bold uppercase tracking-widest">
                          {user.email}
                        </p>
                      </div>
                    </div>
                    <span className="inline-flex items-center px-2.5 py-1 rounded-lg text-[10px] font-black bg-rose-50 text-rose-600 border border-rose-100 uppercase tracking-widest">
                      {user.late_count} lan
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
