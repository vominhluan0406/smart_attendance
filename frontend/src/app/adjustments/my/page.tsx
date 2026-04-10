import { redirect } from "next/navigation";
import { getSession } from "@/lib/auth";
import Nav from "@/components/nav";
import AdjustmentsClient from "./adjustments-client";

export default async function MyAdjustmentsPage() {
  const session = await getSession();
  if (!session) redirect("/login");

  return (
    <div className="min-h-full">
      <Nav session={session} />
      <AdjustmentsClient />
    </div>
  );
}
