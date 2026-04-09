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
                    <LayoutDashboard className="w-4 h-4" /> Tổng quan
                  </Link>
                )}

                {role === "employee" && (
                  <Link href="/attendance" className={desktopLinkClass("/attendance")}>
                    <ScanFace className="w-4 h-4" /> Chấm công
                  </Link>
                )}

                {role === "manager_device" && (
                  <>
                    <Link
                      href="/attendance/qr-manager"
                      className={desktopLinkClass("/attendance/qr-manager")}
                    >
                      <QrCode className="w-4 h-4" /> Mã QR
                    </Link>
                    <Link
                      href="/attendance/password"
                      className={desktopLinkClass("/attendance/password")}
                    >
                      <KeyRound className="w-4 h-4" /> Chấm công mật khẩu
                    </Link>
                  </>
                )}

                {role === "admin" && (
                  <Link href="/branches" className={desktopLinkClass("/branches")}>
                    <MapPin className="w-4 h-4" /> Chi nhánh
                  </Link>
                )}

                {role === "employee" && (
                  <>
                    <Link
                      href="/reports/my-history"
                      className={desktopLinkClass("/reports/my-history")}
                    >
                      <History className="w-4 h-4" /> Lịch sử
                    </Link>
                    <Link
                      href="/adjustments/my"
                      className={desktopLinkClass("/adjustments/my")}
                    >
                      <Clock className="w-4 h-4" /> Bổ sung công
                    </Link>
                    <Link
                      href="/leave/my"
                      className={desktopLinkClass("/leave/my")}
                    >
                      <CalendarOff className="w-4 h-4" /> Nghỉ phép
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
                    <FileText className="w-4 h-4" /> Báo cáo
                  </Link>
                )}

                {role === "manager" && (
                  <>
                    <Link
                      href="/leave/manage"
                      className={desktopLinkClass("/leave/manage")}
                    >
                      <ClipboardCheck className="w-4 h-4" /> Duyệt nghỉ
                    </Link>
                    <Link
                      href="/adjustments/manage"
                      className={desktopLinkClass("/adjustments/manage")}
                    >
                      <Clock className="w-4 h-4" /> Duyệt công
                    </Link>
                  </>
                )}

                {(role === "admin" || role === "manager") && (
                  <Link href="/alerts" className={desktopLinkClass("/alerts")}>
                    <ShieldAlert className="w-4 h-4" /> Gian lận
                  </Link>
                )}

                {role === "admin" && (
                  <Link href="/users" className={desktopLinkClass("/users")}>
                    <Users className="w-4 h-4" /> Nhân viên
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
                <span className="hidden sm:inline">Đăng xuất</span>
              </Link>
            </div>
          </div>
        </div>
      </nav>

      {/* Mobile Bottom Navigation */}
      <nav className="sm:hidden bottom-nav">
        <Link href="/" className={mobileLinkClass("/")}>
          <Home />
          <span>Trang chủ</span>
        </Link>

        {(role === "admin" || role === "manager") && (
          <Link href="/dashboard" className={mobileLinkClass("/dashboard")}>
            <LayoutDashboard />
            <span>Tổng quan</span>
          </Link>
        )}

        {role === "admin" && (
          <>
            <Link href="/branches" className={mobileLinkClass("/branches")}>
              <MapPin />
              <span>Chi nhánh</span>
            </Link>
            <Link href="/users" className={mobileLinkClass("/users")}>
              <Users />
              <span>Nhân viên</span>
            </Link>
          </>
        )}

        {role === "employee" && (
          <>
            <Link href="/attendance" className={mobileLinkClass("/attendance")}>
              <ScanFace />
              <span>Chấm công</span>
            </Link>
            <Link
              href="/reports/my-history"
              className={mobileLinkClass("/reports/my-history")}
            >
              <History />
              <span>Lịch sử</span>
            </Link>
            <Link
              href="/adjustments/my"
              className={mobileLinkClass("/adjustments/my")}
            >
              <Clock />
              <span>Bổ sung</span>
            </Link>
            <Link href="/leave/my" className={mobileLinkClass("/leave/my")}>
              <CalendarOff />
              <span>Nghỉ phép</span>
            </Link>
            <Link href="/profile" className={mobileLinkClass("/profile")}>
              <User />
              <span>Hồ sơ</span>
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
              <span>Báo cáo</span>
            </Link>
            <Link
              href="/leave/manage"
              className={mobileLinkClass("/leave/manage")}
            >
              <ClipboardCheck />
              <span>Duyệt nghỉ</span>
            </Link>
            <Link
              href="/adjustments/manage"
              className={mobileLinkClass("/adjustments/manage")}
            >
              <Clock />
              <span>Duyệt công</span>
            </Link>
            <Link href="/alerts" className={mobileLinkClass("/alerts")}>
              <ShieldAlert />
              <span>Gian lận</span>
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
              <span>Mật khẩu</span>
            </Link>
          </>
        )}
      </nav>
    </>
  );
}
