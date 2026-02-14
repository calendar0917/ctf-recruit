import type { Metadata } from "next";
import type { ReactNode } from "react";
import { AppNav } from "@/components/layout/AppNav";
import "./globals.css";

type Props = {
  children: ReactNode;
};

export const metadata: Metadata = {
  title: "CTF Recruit",
  description: "CTF recruitment platform bootstrap",
};

export default function RootLayout({ children }: Props) {
  return (
    <html lang="en">
      <body>
        <AppNav />
        {children}
      </body>
    </html>
  );
}
