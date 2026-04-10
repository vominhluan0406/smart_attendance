import { redirect } from "next/navigation";
import { getSession } from "@/lib/auth";
import Nav from "@/components/nav";
import LeaveClient from "./leave-client";

export default async function MyLeavePage() {
  const session = await getSession();
  if (!session) redirect("/login");

  return (
    <div className="min-h-full">
      <Nav session={session} />
      <LeaveClient />
    </div>
  );
}
