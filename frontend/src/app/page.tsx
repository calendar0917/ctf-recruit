"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthSession } from "@/lib/use-auth";

export default function HomePage() {
  const router = useRouter();
  const { session, ready } = useAuthSession();

  useEffect(() => {
    if (!ready) {
      return;
    }

    if (session) {
      router.replace("/challenges");
    } else {
      router.replace("/login");
    }
  }, [ready, router, session]);

  return (
    <main className="page">
      <section className="card">
        <h1>CTF Recruit Platform</h1>
        <p>Redirecting…</p>
      </section>
    </main>
  );
}
