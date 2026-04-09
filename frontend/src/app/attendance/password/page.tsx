import { redirect } from "next/navigation";
import { getSession } from "@/lib/auth";
import Nav from "@/components/nav";
import PasswordKiosk from "./password-kiosk";

export default async function PasswordAttendancePage() {
  const session = await getSession();
  if (!session) redirect("/login");

  // Only manager_device kiosks should be accessing the physical station password check-in
  if (session.role !== "manager_device") {
    redirect("/dashboard");
  }

  return (
    <div className="min-h-full flex flex-col bg-gray-50">
      <div className="print:hidden">
        <Nav session={session} />
      </div>

      <main className="flex-1 flex flex-col items-center justify-center p-4">
        <div className="w-full max-w-5xl">
          <PasswordKiosk branchName={session.branchName || "Chi nhánh chưa đặt tên"} />
        </div>
      </main>
    </div>
  );
}
