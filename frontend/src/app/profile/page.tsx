import { redirect } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import type { User } from "@/lib/types";
import { User as UserIcon, Fingerprint } from "lucide-react";
import WebAuthnButton from "./webauthn-button";

export default async function ProfilePage() {
  const session = await getSession();
  if (!session) redirect("/login");

  const cookie = await getCookieHeader();

  let user: User | null = null;
  let error = "";

  try {
    const res = await apiGet<User>("/api/profile", cookie);
    if (res.data) user = res.data;
  } catch (e) {
    error = e instanceof Error ? e.message : "Khong the tai du lieu";
  }

  return (
    <div className="min-h-full bg-gray-50 pb-12">
      <Nav session={session} />

      <main className="mx-auto max-w-lg px-4 py-12 sm:px-6 lg:px-8">
        <div className="bg-white rounded-3xl shadow-xl overflow-hidden border border-gray-100">
          <div className="p-8">
            <div className="flex items-center gap-4 mb-8">
              <div className="p-3 bg-primary-100 rounded-2xl">
                <UserIcon className="w-8 h-8 text-primary-600" />
              </div>
              <div>
                <h2 className="text-2xl font-bold text-gray-900">
                  {user?.full_name || session.fullName}
                </h2>
                <p className="text-sm text-gray-500">
                  {user?.email || session.email}
                </p>
              </div>
            </div>

            {error && (
              <div className="bg-red-50 border border-red-200 rounded-xl p-4 mb-6">
                <p className="text-sm text-red-700">{error}</p>
              </div>
            )}

            {user && (
              <div className="space-y-4 mb-8">
                <div className="flex justify-between py-3 border-b border-gray-50">
                  <span className="text-sm text-gray-500">Ma nhan vien</span>
                  <span className="text-sm font-bold text-gray-900">
                    {user.employee_code || "---"}
                  </span>
                </div>
                <div className="flex justify-between py-3 border-b border-gray-50">
                  <span className="text-sm text-gray-500">Vai tro</span>
                  <span className="text-sm font-bold text-gray-900">
                    {user.role}
                  </span>
                </div>
                <div className="flex justify-between py-3 border-b border-gray-50">
                  <span className="text-sm text-gray-500">So dien thoai</span>
                  <span className="text-sm font-bold text-gray-900">
                    {user.phone || "---"}
                  </span>
                </div>
                <div className="flex justify-between py-3 border-b border-gray-50">
                  <span className="text-sm text-gray-500">Phong ban</span>
                  <span className="text-sm font-bold text-gray-900">
                    {user.department?.name || "---"}
                  </span>
                </div>
                <div className="flex justify-between py-3 border-b border-gray-50">
                  <span className="text-sm text-gray-500">Chuc vu</span>
                  <span className="text-sm font-bold text-gray-900">
                    {user.position || "---"}
                  </span>
                </div>
              </div>
            )}

            <div className="space-y-6">
              <div className="bg-gray-50 rounded-2xl p-6 border border-gray-100">
                <h3 className="text-sm font-bold text-gray-700 uppercase mb-4 flex items-center gap-2">
                  <Fingerprint className="w-4 h-4 text-emerald-600" />
                  Xac thuc Sinh trac hoc
                </h3>
                <p className="text-sm text-gray-500 mb-6">
                  Dang ky van tay hoac khuon mat tren thiet bi nay de cham
                  cong nhanh hon va bao mat hon.
                </p>

                <WebAuthnButton />
              </div>

              <div className="p-4 border border-primary-50 rounded-2xl bg-primary-50/20">
                <p className="text-xs text-primary-400 leading-relaxed text-center">
                  Luu y: Thiet bi cua ban can ho tro bao mat sinh trac hoc va
                  ban phai thiet lap san khoa man hinh (PIN, Pattern, hoac
                  Biometric).
                </p>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
