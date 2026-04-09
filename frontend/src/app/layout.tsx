import type { Metadata, Viewport } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Smart Attendance",
  description: "He thong cham cong thong minh cho doanh nghiep",
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
  themeColor: "#4f46e5",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="vi" className="h-full bg-gray-50">
      <body className="h-full pb-16 sm:pb-0 font-sans antialiased">
        <div className="page-transition">{children}</div>
      </body>
    </html>
  );
}
