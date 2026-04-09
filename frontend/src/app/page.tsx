import { redirect } from "next/navigation";
import { getSession } from "@/lib/auth";

export default async function HomePage() {
  const session = await getSession();

  if (!session) {
    redirect("/login");
  }

  switch (session.role) {
    case "admin":
    case "manager":
      redirect("/dashboard");
      break;
    case "manager_device":
      redirect("/attendance/qr-manager");
      break;
    case "employee":
    default:
      redirect("/attendance");
      break;
  }
}
