import type { Metadata } from "next";
import { Geist_Mono, Inter } from "next/font/google";
import "./globals.css";
import { cn } from "@/lib/utils";

const inter = Inter({ subsets: ["latin"], variable: "--font-inter" });

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Verifield — Pelacakan Job Order Inspeksi",
  description:
    "Status pekerjaan inspeksi lapangan terlihat langsung oleh klien, tanpa perantara manusia.",
};

// Ditempel sebelum paint agar tema tidak berkedip. Kelas ditulis di luar React,
// karena itu <html> memakai suppressHydrationWarning.
const themeScript = `
try {
  var t = localStorage.getItem("verifield-theme");
  if (t === "dark" || (t !== "light" && matchMedia("(prefers-color-scheme: dark)").matches)) {
    document.documentElement.classList.add("dark");
  }
} catch (e) {}
`;

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="id"
      suppressHydrationWarning
      className={cn("h-full antialiased font-sans", inter.variable, geistMono.variable)}
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
      </head>
      <body className="min-h-full flex flex-col">{children}</body>
    </html>
  );
}
