"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  ScanFace,
  MapPin,
  History,
  Clock,
  CalendarOff,
  FileText,
  ClipboardCheck,
  ShieldAlert,
  Users,
  LogOut,
  Home,
  User,
  QrCode,
  ShieldCheck,
  KeyRound,
} from "lucide-react";
import type { Session } from "@/lib/types";

interface NavProps {
  session: Session;
}

export default function Nav({ session }: NavProps) {
  const pathname = usePathname();
  const role = session.role;

  const isActive = (href: string) =>
    pathname === href || pathname.startsWith(href + "/");

  const desktopLinkClass = (href: string) =>
    `inline-flex items-center border-b-2 px-1 pt-1 text-sm font-medium gap-2 transition-colors ${
      isActive(href)
        ? "border-primary-600 text-primary-600"
        : "border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700"
    }`;

  const mobileLinkClass = (href: string) =>
    `bottom-nav-item ${isActive(href) ? "active text-primary-600" : ""}`;

  return (
    <>
      {/* Desktop Top Nav */}
      <nav className="bg-white shadow-sm border-b border-gray-200 sticky top-0 z-40">
        <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
          <div className="flex h-16 justify-between">
            <div className="flex">
              <div className="flex flex-shrink-0 items-center">
                <Link
                  href="/"
                  className="text-xl font-bold text-primary-600 flex items-center gap-2"
                >
                  <ShieldCheck className="w-8 h-8" />
                  <span className="hidden sm:inline">Smart Attendance</span>
                </Link>
              </div>

              {/* Desktop Menu */}
              <div className="hidden sm:ml-6 sm:flex sm:space-x-8">
                {(role === "admin" || role === "manager") && (
                  <Link href="/dashboard" className={desktopLinkClass("/dashboard")}>
                    <LayoutDashboard className="w-4 h-4" /> Tong quan
                  </Link>
                )}

                {role === "employee" && (
                  <Link href="/attendance" className={desktopLinkClass("/attendance")}>
                    <ScanFace className="w-4 h-4" /> Cham cong
                  </Link>
                )}

                {role === "manager_device" && (
                  <>
                    <Link
                      href="/attendance/qr-manager"
                      className={desktopLinkClass("/attendance/qr-manager")}
                    >
                      <QrCode className="w-4 h-4" /> Ma QR
                    </Link>
                    <Link
                      href="/attendance/password"
                      className={desktopLinkClass("/attendance/password")}
                    >
                      <KeyRound className="w-4 h-4" /> Cham cong mat khau
                    </Link>
                  </>
                )}

                {role === "admin" && (
                  <Link href="/branches" className={desktopLinkClass("/branches")}>
                    <MapPin className="w-4 h-4" /> Chi nhanh
                  </Link>
                )}

                {role === "employee" && (
                  <>
                    <Link
                      href="/reports/my-history"
                      className={desktopLinkClass("/reports/my-history")}
                    >
                      <History className="w-4 h-4" /> Lich su
                    </Link>
                    <Link
                      href="/adjustments/my"
                      className={desktopLinkClass("/adjustments/my")}
                    >
                      <Clock className="w-4 h-4" /> Bo sung cong
                    </Link>
                    <Link
                      href="/leave/my"
                      className={desktopLinkClass("/leave/my")}
                    >
                      <CalendarOff className="w-4 h-4" /> Nghi phep
                    </Link>
                  </>
                )}

                {(role === "admin" || role === "manager") && (
                  <Link
                    href={
                      role === "admin"
                        ? "/reports"
                        : `/reports/branch/${session.branchId}`
                    }
                    className={desktopLinkClass("/reports")}
                  >
                    <FileText className="w-4 h-4" /> Bao cao
                  </Link>
                )}

                {role === "manager" && (
                  <>
                    <Link
                      href="/leave/manage"
                      className={desktopLinkClass("/leave/manage")}
                    >
                      <ClipboardCheck className="w-4 h-4" /> Duyet nghi
                    </Link>
                    <Link
                      href="/adjustments/manage"
                      className={desktopLinkClass("/adjustments/manage")}
                    >
                      <Clock className="w-4 h-4" /> Duyet cong
                    </Link>
                  </>
                )}

                {(role === "admin" || role === "manager") && (
                  <Link href="/alerts" className={desktopLinkClass("/alerts")}>
                    <ShieldAlert className="w-4 h-4" /> Gian lan
                  </Link>
                )}

                {role === "admin" && (
                  <Link href="/users" className={desktopLinkClass("/users")}>
                    <Users className="w-4 h-4" /> Nhan vien
                  </Link>
                )}
              </div>
            </div>

            <div className="flex items-center gap-3">
              <div className="hidden sm:block text-right mr-2">
                {role === "employee" ? (
                  <Link href="/profile" className="group">
                    <p className="text-sm font-bold text-gray-900 leading-tight group-hover:text-primary-600 transition-colors">
                      {session.fullName}
                    </p>
                    {session.branchName && (
                      <p className="text-xs text-gray-400">
                        {session.branchName}
                      </p>
                    )}
                  </Link>
                ) : (
                  <div>
                    <p className="text-sm font-bold text-gray-900 leading-tight">
                      {session.fullName}
                    </p>
                    {session.branchName && (
                      <p className="text-xs text-gray-400">
                        {session.branchName}
                      </p>
                    )}
                  </div>
                )}
              </div>
              <Link
                href="/api/auth/logout-action"
                className="text-sm font-medium text-gray-500 hover:text-gray-700 flex items-center gap-1"
                onClick={async (e) => {
                  e.preventDefault();
                  document.cookie =
                    "access_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT";
                  document.cookie =
                    "refresh_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT";
                  window.location.href = "/login";
                }}
              >
                <LogOut className="w-4 h-4" />
                <span className="hidden sm:inline">Dang xuat</span>
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Mobile Bottom Navigation */}
      <nav className="sm:hidden bottom-nav">
        <Link href="/" className={mobileLinkClass("/")}>
          <Home />
          <span>Trang chu</span>
        </Link>

        {(role === "admin" || role === "manager") && (
          <Link href="/dashboard" className={mobileLinkClass("/dashboard")}>
            <LayoutDashboard />
            <span>Tong quan</span>
          </Link>
        )}

        {role === "admin" && (
          <>
            <Link href="/branches" className={mobileLinkClass("/branches")}>
              <MapPin />
              <span>Chi nhanh</span>
            </Link>
            <Link href="/users" className={mobileLinkClass("/users")}>
              <Users />
              <span>Nhan vien</span>
            </Link>
          </>
        )}

        {role === "employee" && (
          <>
            <Link href="/attendance" className={mobileLinkClass("/attendance")}>
              <ScanFace />
              <span>Cham cong</span>
            </Link>
            <Link
              href="/reports/my-history"
              className={mobileLinkClass("/reports/my-history")}
            >
              <History />
              <span>Lich su</span>
            </Link>
            <Link
              href="/adjustments/my"
              className={mobileLinkClass("/adjustments/my")}
            >
              <Clock />
              <span>Bo sung</span>
            </Link>
            <Link href="/leave/my" className={mobileLinkClass("/leave/my")}>
              <CalendarOff />
              <span>Nghi phep</span>
            </Link>
            <Link href="/profile" className={mobileLinkClass("/profile")}>
              <User />
              <span>Ho so</span>
            </Link>
          </>
        )}

        {role === "manager" && (
          <>
            <Link
              href={`/reports/branch/${session.branchId}`}
              className={mobileLinkClass("/reports")}
            >
              <FileText />
              <span>Bao cao</span>
            </Link>
            <Link
              href="/leave/manage"
              className={mobileLinkClass("/leave/manage")}
            >
              <ClipboardCheck />
              <span>Duyet nghi</span>
            </Link>
            <Link
              href="/adjustments/manage"
              className={mobileLinkClass("/adjustments/manage")}
            >
              <Clock />
              <span>Duyet cong</span>
            </Link>
            <Link href="/alerts" className={mobileLinkClass("/alerts")}>
              <ShieldAlert />
              <span>Gian lan</span>
            </Link>
          </>
        )}

        {role === "manager_device" && (
          <>
            <Link
              href="/attendance/qr-manager"
              className={mobileLinkClass("/attendance/qr-manager")}
            >
              <QrCode />
              <span>QR</span>
            </Link>
            <Link
              href="/attendance/password"
              className={mobileLinkClass("/attendance/password")}
            >
              <KeyRound />
              <span>Mat khau</span>
            </Link>
          </>
        )}
      </nav>
    </>
  );
}
