"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import React from "react";
import { clearStoredSession } from "@/lib/auth-storage";
import { useAuthSession } from "@/lib/use-auth";

void React;

function isActive(pathname: string, href: string): boolean {
  if (href === "/") {
    return pathname === href;
  }
  return pathname === href || pathname.startsWith(`${href}/`);
}

export function AppNav() {
  const pathname = usePathname();
  const router = useRouter();
  const { session, ready } = useAuthSession();

  if (!ready) {
    return (
      <header className="app-nav" data-auth-state="loading">
        <div className="app-nav-inner">
          <Link className="brand" href="/">
            CTF Recruit
          </Link>

          <nav className="nav-links" aria-label="Primary navigation loading state">
            <span className="empty-text">Loading navigation…</span>
          </nav>

          <div className="nav-user">
            <span className="info-text">Checking session…</span>
          </div>
        </div>
      </header>
    );
  }

  const loggedIn = Boolean(session);

  return (
    <header className="app-nav">
      <div className="app-nav-inner">
        <Link className="brand" href={loggedIn ? "/challenges" : "/login"}>
          CTF Recruit
        </Link>

        {loggedIn ? (
          <>
            <nav className="nav-links">
              <Link className={isActive(pathname, "/challenges") ? "active" : ""} href="/challenges">
                Challenges
              </Link>
              <Link
                className={isActive(pathname, "/announcements") ? "active" : ""}
                href="/announcements"
              >
                Announcements
              </Link>
              <Link className={isActive(pathname, "/recruitment") ? "active" : ""} href="/recruitment">
                Recruitment
              </Link>
              <Link className={isActive(pathname, "/scoreboard") ? "active" : ""} href="/scoreboard">
                Scoreboard
              </Link>
              {session?.user.role === "admin" ? (
                <Link className={isActive(pathname, "/admin") ? "active" : ""} href="/admin/users">
                  Admin
                </Link>
              ) : null}
            </nav>

            <div className="nav-user">
              <span>
                {session?.user.displayName} ({session?.user.role})
              </span>
              <button
                type="button"
                className="button secondary"
                onClick={() => {
                  clearStoredSession();
                  router.push("/login");
                }}
              >
                Logout
              </button>
            </div>
          </>
        ) : (
          <>
            <nav className="nav-links">
              <Link className={isActive(pathname, "/login") ? "active" : ""} href="/login">
                Login
              </Link>
              <Link className={isActive(pathname, "/register") ? "active" : ""} href="/register">
                Register
              </Link>
            </nav>

            <div className="nav-user">
              <span className="empty-text">Session missing or expired.</span>
            </div>
          </>
        )}
      </div>
    </header>
  );
}
