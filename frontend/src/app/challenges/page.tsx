"use client";

import { useEffect, useState } from "react";
import { ChallengeList } from "@/components/challenge/ChallengeList";
import { listChallenges } from "@/lib/api/challenges";
import { HttpError } from "@/lib/http";
import { useRequireAuth } from "@/lib/use-auth";
import type { Challenge } from "@/lib/types";

const PAGE_LIMIT = 20;

export default function ChallengesPage() {
  const { session, ready, authorized } = useRequireAuth();
  const [items, setItems] = useState<Challenge[]>([]);
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
        const response = await listChallenges(accessToken, {
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
          setError("Failed to load challenges.");
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
        <section className="card">
          <p>Loading session...</p>
        </section>
      </main>
    );
  }

  return (
    <main className="page page-content">
      <section className="card">
        <h1>Challenges</h1>
        <p>Browse and solve available challenges.</p>
      </section>

      {loading ? (
        <section className="card">
          <p>Loading challenges...</p>
        </section>
      ) : null}

      {error ? (
        <section className="card">
          <p className="error-text">{error}</p>
        </section>
      ) : null}

      {!loading && !error ? <ChallengeList items={items} /> : null}
    </main>
  );
}
