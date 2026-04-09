import { redirect, notFound } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import type { Branch } from "@/lib/types";
import BranchForm from "./branch-form";

interface EditBranchPageProps {
  params: { id: string };
}

export default async function EditBranchPage({ params }: EditBranchPageProps) {
  const session = await getSession();
  if (!session) redirect("/login");

  // Only admin/manager can edit branches
  if (session.role !== "admin" && session.role !== "manager") {
    redirect("/dashboard");
  }

  const cookie = await getCookieHeader();
  const { id } = params;
  
  try {
    const res = await apiGet<Branch>(`/api/branches/${id}`, cookie);
    if (!res.success || !res.data) {
      return notFound();
    }

    return (
      <div className="min-h-full py-8 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto">
          <div className="mb-8">
            <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight">Cài đặt chi nhánh</h1>
            <p className="mt-2 text-sm text-gray-500">
              Cập nhật thông tin và các phương thức xác thực cho chi nhánh này.
            </p>
          </div>

          <div className="bg-white shadow-2xl rounded-3xl overflow-hidden border border-gray-100">
            <div className="p-8">
              <BranchForm branch={res.data} />
            </div>
          </div>
        </div>
      </div>
    );
  } catch (error) {
    console.error("[EditBranchPage] fetch error:", error);
    return notFound();
  }
}
