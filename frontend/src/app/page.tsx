"use client";

import { useRouter } from "next/navigation";
import React, { useEffect } from "react";
import { useAuthSession } from "@/lib/use-auth";

void React;

export default function HomePage() {
  const router = useRouter();
  const { session, ready } = useAuthSession();

  const destination = session ? "/challenges" : "/login";
  const destinationLabel = session ? "challenges" : "login";

  useEffect(() => {
    if (!ready) {
      return;
    }

    router.replace(destination);
  }, [destination, ready, router]);

  return (
    <main className="page">
      <section className="card">
        <h1>CTF Recruit Platform</h1>
        {!ready ? (
          <p className="info-text">Checking local session before redirect…</p>
        ) : (
          <p className="info-text">Session resolved. Redirecting to {destinationLabel}…</p>
        )}
      </section>
    </main>
  );
}
