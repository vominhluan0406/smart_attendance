import { redirect } from "next/navigation";
import { getSession } from "@/lib/auth";
import Nav from "@/components/nav";
import AttendanceClient from "./attendance-client";

export default async function AttendancePage() {
  const session = await getSession();
  if (!session) redirect("/login");

  return (
    <div className="min-h-full">
      <Nav session={session} />
      <AttendanceClient />
    </div>
  );
}
