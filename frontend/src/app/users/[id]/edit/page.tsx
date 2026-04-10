import { redirect, notFound } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import type { User, Branch } from "@/lib/types";
import EditUserForm from "./edit-user-form";
import Link from "next/link";
import { ShieldCheck } from "lucide-react";

interface PageProps {
  params: Promise<{ id: string }>;
}

export default async function EditUserPage({ params }: PageProps) {
  const session = await getSession();
  if (!session) redirect("/login");

  if (session.role !== "admin" && session.role !== "manager") {
    redirect("/dashboard");
  }

  const cookie = await getCookieHeader();
  const { id } = await params;

  let user: User | null = null;
  let branches: Branch[] = [];

  try {
    const userRes = await apiGet<User>(`/api/users/${id}`, cookie);
    if (!userRes.success || !userRes.data) return notFound();
    user = userRes.data;
  } catch {
    return notFound();
  }

  try {
    const branchRes = await apiGet<Branch[]>("/api/branches?limit=100", cookie);
    if (branchRes.data) branches = branchRes.data;
  } catch {}

  return (
    <div className="min-h-full">
      <Nav session={session} />

      <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="mb-8">
          <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight">Chỉnh sửa nhân viên</h1>
          <p className="mt-2 text-sm text-gray-500">
            Cập nhật thông tin cho <span className="font-bold text-gray-700">{user.full_name}</span>
          </p>
        </div>

        {session.role === "admin" && (
          <Link
            href={`/users/${user.id}/security`}
            className="mb-6 flex items-center gap-3 p-4 bg-indigo-50 border border-indigo-100 rounded-2xl hover:bg-indigo-100 transition-all group"
          >
            <ShieldCheck className="w-5 h-5 text-indigo-600" />
            <div>
              <p className="text-sm font-bold text-indigo-700 group-hover:text-indigo-800">Quản lý thiết bị & sinh trắc học</p>
              <p className="text-xs text-indigo-400">Xem, phê duyệt, chặn thiết bị và credential WebAuthn</p>
            </div>
          </Link>
        )}

        <div className="bg-white shadow-2xl rounded-3xl overflow-hidden border border-gray-100">
          <div className="p-8">
            <EditUserForm user={user} branches={branches} />
          </div>
        </div>
      </main>
    </div>
  );
}
