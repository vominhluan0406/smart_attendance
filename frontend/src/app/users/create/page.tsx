import { redirect } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import type { Branch } from "@/lib/types";
import UserForm from "./user-form";

export default async function CreateUserPage() {
  const session = await getSession();
  if (!session) redirect("/login");

  // Only admin and manager can create users
  if (session.role !== "admin" && session.role !== "manager") {
    redirect("/dashboard");
  }

  const cookie = await getCookieHeader();
  
  // Fetch branches for the branch selection dropdown
  let branches: Branch[] = [];
  try {
    const res = await apiGet<Branch[]>("/api/branches?limit=100", cookie);
    if (res.data) {
      branches = res.data;
    }
  } catch (error) {
    console.error("[CreateUserPage] Failed to fetch branches:", error);
  }

  return (
    <div className="min-h-full">
      <Nav session={session} />

      <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="mb-8">
          <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight">Thêm nhân viên mới</h1>
          <p className="mt-2 text-sm text-gray-500">
            Điền thông tin bên dưới để tạo tài khoản nhân viên. Mật khẩu mặc định có thể được cấp cho nhân viên để đăng nhập.
          </p>
        </div>

        <div className="bg-white shadow-2xl rounded-3xl overflow-hidden border border-gray-100">
          <div className="p-8">
            <UserForm branches={branches} />
          </div>
        </div>
      </main>
    </div>
  );
}
