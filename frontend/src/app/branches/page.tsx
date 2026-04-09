import { redirect } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import DataTable, { Column } from "@/components/data-table";
import Pagination from "@/components/pagination";
import type { Branch, PaginationMeta } from "@/lib/types";
import { MapPin, Plus, Search, Pencil } from "lucide-react";
import Link from "next/link";

interface SearchParams {
  page?: string;
  search?: string;
}

export default async function BranchesPage({
  searchParams,
}: {
  searchParams: Promise<SearchParams>;
}) {
  const session = await getSession();
  if (!session) redirect("/login");
  if (session.role !== "admin") redirect("/");

  const cookie = await getCookieHeader();
  const params = await searchParams;
  const page = parseInt(params.page || "1", 10);
  const search = params.search || "";

  let branches: Branch[] = [];
  let meta: PaginationMeta = { page: 1, limit: 20, total: 0 };
  let error = "";

  try {
    const query = new URLSearchParams();
    query.set("page", String(page));
    query.set("limit", "20");
    if (search) query.set("search", search);

    const res = await apiGet<Branch[]>(
      `/api/branches?${query.toString()}`,
      cookie
    );
    if (res.data) branches = res.data;
    if (res.meta) meta = res.meta;
  } catch (e) {
    error = e instanceof Error ? e.message : "Khong the tai du lieu";
  }

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
      header: "Ten chi nhanh",
      render: (item) => (
        <div>
          <div className="text-sm font-bold text-gray-900">{item.name}</div>
          <div className="text-xs text-gray-400">{item.address}</div>
        </div>
      ),
    },
    {
      key: "methods",
      header: "Phuong thuc",
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
      header: "Trang thai",
      render: (item) => (
        <span
          className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-bold ${
            item.is_active
              ? "bg-green-50 text-green-700"
              : "bg-gray-50 text-gray-500"
          }`}
        >
          {item.is_active ? "Hoat dong" : "Tat"}
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
          Sua
        </Link>
      ),
    },
  ];

  return (
    <div className="min-h-full">
      <Nav session={session} />

      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        <div className="sm:flex sm:items-center sm:justify-between mb-6">
          <div className="flex items-center gap-3">
            <MapPin className="w-6 h-6 text-primary-600" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">
                Quan ly chi nhanh
              </h1>
              <p className="mt-1 text-sm text-gray-500">
                Quan ly cac chi nhanh cong ty va cau hinh cham cong
              </p>
            </div>
          </div>
          <div className="mt-4 sm:mt-0">
            <Link
              href="/branches/create"
              className="rounded-md bg-primary-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-primary-500 inline-flex items-center gap-2"
            >
              <Plus className="w-4 h-4" />
              Them chi nhanh
            </Link>
          </div>
        </div>

        {/* Search */}
        <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 mb-6">
          <form className="flex flex-wrap gap-4 items-end">
            <div className="flex-1 min-w-[200px]">
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Tim kiem
              </label>
              <input
                type="text"
                name="search"
                defaultValue={search}
                placeholder="Ten chi nhanh hoac dia chi..."
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 sm:text-sm py-1.5 px-3 ring-1 ring-inset ring-gray-300"
              />
            </div>
            <button
              type="submit"
              className="flex items-center gap-2 rounded-md bg-gray-100 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-200"
            >
              <Search className="w-4 h-4" />
              Tim kiem
            </button>
          </form>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 rounded-xl p-4 mb-6">
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}

        <DataTable
          columns={columns}
          data={branches}
          emptyMessage="Chua co chi nhanh nao."
          keyExtractor={(item) => item.id}
        />

        <Pagination page={meta.page} total={meta.total} limit={meta.limit} />
      </main>
    </div>
  );
}
