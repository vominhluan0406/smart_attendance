import { redirect } from "next/navigation";
import { getSession, getCookieHeader } from "@/lib/auth";
import { apiGet } from "@/lib/api";
import Nav from "@/components/nav";
import type { Branch } from "@/lib/types";
import QRDisplay from "./qr-display";

export default async function QRManagerPage() {
  const session = await getSession();
  if (!session) redirect("/login");

  // Only authorized personnel can access the QR Kiosk Manager
  if (!["admin", "manager", "manager_device"].includes(session.role)) {
    redirect("/dashboard");
  }

  const cookie = await getCookieHeader();
  
  // Fetch available branches
  let branches: Branch[] = [];
  try {
    const res = await apiGet<Branch[]>("/api/branches?limit=100", cookie);
    if (res.data) {
      branches = res.data;
    }
  } catch (error) {
    console.error("[QRManagerPage] Failed to fetch branches:", error);
  }

  // If user is manager/manager_device and has a branchId, they can only manage their own branch
  if (session.role !== "admin" && session.branchId) {
    branches = branches.filter(b => b.id === session.branchId);
  }

  return (
    <div className="min-h-full flex flex-col bg-gray-50">
      <div className="print:hidden">
        <Nav session={session} />
      </div>

      <main className="flex-1 flex flex-col items-center justify-center p-4">
        <div className="w-full max-w-5xl">
          <QRDisplay 
            branches={branches} 
            defaultBranchId={session.role !== "admin" ? session.branchId : undefined} 
          />
        </div>
      </main>
    </div>
  );
}
