import { redirect } from "next/navigation";
import { getSession } from "@/lib/auth";
import Nav from "@/components/nav";
import CreateBranchForm from "./create-branch-form";

export default async function CreateBranchPage() {
  const session = await getSession();
  if (!session) redirect("/login");

  if (session.role !== "admin") {
    redirect("/dashboard");
  }

  return (
    <div className="min-h-full">
      <Nav session={session} />

      <main className="py-8 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto">
          <div className="mb-8">
            <h1 className="text-3xl font-extrabold text-gray-900 tracking-tight">Thêm chi nhánh mới</h1>
            <p className="mt-2 text-sm text-gray-500">
              Tạo chi nhánh và cấu hình phương thức chấm công.
            </p>
          </div>

          <div className="bg-white shadow-2xl rounded-3xl overflow-hidden border border-gray-100">
            <div className="p-8">
              <CreateBranchForm />
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
