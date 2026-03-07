"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { AnnouncementDetail } from "@/components/announcement/AnnouncementDetail";
import { getAnnouncement } from "@/lib/api/announcements";
import { HttpError } from "@/lib/http";
import { useRequireAuth } from "@/lib/use-auth";
import type { Announcement } from "@/lib/types";

export default function AnnouncementDetailPage() {
  const params = useParams<{ id: string }>();
  const announcementId = params?.id;

  const { session, ready, authorized } = useRequireAuth();
  const [announcement, setAnnouncement] = useState<Announcement | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | undefined>();

  useEffect(() => {
    if (!ready || !authorized || !session || !announcementId) {
      return;
    }

    const accessToken = session.accessToken;
    let cancelled = false;

    async function run() {
      setLoading(true);
      setError(undefined);

      try {
        const response = await getAnnouncement(accessToken, announcementId);
        if (!cancelled) {
          setAnnouncement(response);
        }
      } catch (err) {
        if (cancelled) {
          return;
        }

        if (err instanceof HttpError) {
          setError(err.message);
        } else {
          setError("Failed to load announcement.");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void run();

    return () => {
      cancelled = true;
    };
  }, [announcementId, authorized, ready, session]);

  if (!ready || !authorized || !session) {
    return (
      <main className="page">
        <section className="card">
          <p>Loading session...</p>
        </section>
      </main>
    );
  }

  if (!announcementId) {
    return (
      <main className="page">
        <section className="card">
          <p className="error-text">Announcement ID is missing.</p>
        </section>
      </main>
    );
  }

  if (loading) {
    return (
      <main className="page">
        <section className="card">
          <p>Loading announcement...</p>
        </section>
      </main>
    );
  }

  if (error || !announcement) {
    return (
      <main className="page">
        <section className="card">
          <p className="error-text">{error ?? "Announcement not found."}</p>
          <p className="helper-text">
            <Link href="/announcements">Back to announcements</Link>
          </p>
        </section>
      </main>
    );
  }

  return (
    <main className="page page-content">
      <p>
        <Link href="/announcements">← Back to announcements</Link>
      </p>
      <AnnouncementDetail announcement={announcement} />
    </main>
  );
}
