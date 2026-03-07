"use client";

import { useEffect, useState } from "react";
import { AnnouncementList } from "@/components/announcement/AnnouncementList";
import { ErrorStateCard, LoadingStateCard } from "@/components/ui/StateFeedback";
import { listAnnouncements } from "@/lib/api/announcements";
import { HttpError } from "@/lib/http";
import type { Announcement } from "@/lib/types";
import { useRequireAuth } from "@/lib/use-auth";

const PAGE_LIMIT = 20;

export default function AnnouncementsPage() {
  const { session, ready, authorized } = useRequireAuth();
  const [items, setItems] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | undefined>();

  useEffect(() => {
    if (!ready || !authorized || !session) {
      return;
    }

    const accessToken = session.accessToken;
    let cancelled = false;

    async function run() {
      setLoading(true);
      setError(undefined);

      try {
        const response = await listAnnouncements(accessToken, {
          limit: PAGE_LIMIT,
          offset: 0,
        });

        if (!cancelled) {
          setItems(response.items);
        }
      } catch (err) {
        if (cancelled) {
          return;
        }
        if (err instanceof HttpError) {
          setError(err.message);
        } else {
          setError("Failed to load announcements.");
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
  }, [authorized, ready, session]);

  if (!ready || !authorized || !session) {
    return (
      <main className="page">
        <LoadingStateCard message="Loading session..." />
      </main>
    );
  }

  return (
    <main className="page page-content">
      <section className="card">
        <h1>Announcements</h1>
        <p>Latest competition updates and notices.</p>
      </section>

      {loading ? (
        <LoadingStateCard message="Loading announcements..." />
      ) : null}

      {error ? (
        <ErrorStateCard message={error} />
      ) : null}

      {!loading && !error ? <AnnouncementList items={items} /> : null}
    </main>
  );
}
