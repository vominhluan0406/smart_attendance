import { redirect, notFound } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import type { User, UserDevice, UserCredential } from "@/lib/types";
import DeviceList from "./device-list";
import CredentialList from "./credential-list";
import { ShieldCheck, ChevronRight } from "lucide-react";
import Link from "next/link";

interface PageProps {
  params: Promise<{ id: string }>;
}

export default async function UserSecurityPage({ params }: PageProps) {
  const session = await getSession();
  if (!session) redirect("/login");
  if (session.role !== "admin") redirect("/dashboard");

  const cookie = await getCookieHeader();
  const { id } = await params;

  let user: User | null = null;
  let devices: UserDevice[] = [];
  let credentials: UserCredential[] = [];

  try {
    const userRes = await apiGet<User>(`/api/users/${id}`, cookie);
    if (!userRes.success || !userRes.data) return notFound();
    user = userRes.data;
  } catch {
    return notFound();
  }

  try {
    const devRes = await apiGet<UserDevice[]>(`/api/users/${id}/devices`, cookie);
    if (devRes.data) devices = devRes.data;
  } catch {}

  try {
    const credRes = await apiGet<UserCredential[]>(`/api/users/${id}/credentials`, cookie);
    if (credRes.data) credentials = credRes.data;
  } catch {}

  return (
    <div className="min-h-full">
      <Nav session={session} />

      <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Breadcrumb */}
        <div className="flex items-center gap-2 text-sm text-primary-600 font-medium mb-2">
          <Link href="/users" className="hover:underline">Nhân viên</Link>
          <ChevronRight className="w-4 h-4" />
          <Link href={`/users/${id}/edit`} className="hover:underline">{user.full_name}</Link>
          <ChevronRight className="w-4 h-4" />
          <span>Bảo mật</span>
        </div>

        <div className="mb-8 flex items-center gap-3">
          <div className="p-3 bg-indigo-100 rounded-2xl">
            <ShieldCheck className="w-7 h-7 text-indigo-600" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">
              Thiết bị & Sinh trắc học
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              Quản lý thiết bị và xác thực WebAuthn của <span className="font-bold text-gray-700">{user.full_name}</span> ({user.email})
            </p>
          </div>
        </div>

        <div className="space-y-8">
          {/* WebAuthn Credentials */}
          <CredentialList userId={id} credentials={credentials} />

          {/* Devices */}
          <DeviceList userId={id} devices={devices} />
        </div>
      </main>
    </div>
  );
}
