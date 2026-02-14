"use client";

import { useEffect, useState } from "react";
import { ScoreboardTable } from "@/components/scoreboard/ScoreboardTable";
import { listScoreboard } from "@/lib/api/scoreboard";
import { HttpError } from "@/lib/http";
import { useRequireAuth } from "@/lib/use-auth";
import type { ScoreboardItem } from "@/lib/types";

const LIMIT = 20;

export default function ScoreboardPage() {
  const { session, ready, authorized } = useRequireAuth();

  const [items, setItems] = useState<ScoreboardItem[]>([]);
  const [offset, setOffset] = useState(0);
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
        const response = await listScoreboard(accessToken, {
          limit: LIMIT,
          offset,
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
          setError("Failed to load scoreboard.");
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
  }, [authorized, offset, ready, session]);

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
        <h1>Scoreboard</h1>
        <p>Ranked by total points, then earliest last accepted time.</p>
      </section>

      {loading ? (
        <section className="card">
          <p>Loading scoreboard...</p>
        </section>
      ) : null}

      {error ? (
        <section className="card">
          <p className="error-text">{error}</p>
        </section>
      ) : null}

      {!loading && !error ? (
        <section className="card">
          <ScoreboardTable items={items} />
          <div className="row-actions">
            <button
              type="button"
              className="button secondary"
              disabled={offset === 0 || loading}
              onClick={() => setOffset((prev) => Math.max(0, prev - LIMIT))}
            >
              Previous
            </button>
            <span>Offset: {offset}</span>
            <button
              type="button"
              className="button secondary"
              disabled={items.length < LIMIT || loading}
              onClick={() => setOffset((prev) => prev + LIMIT)}
            >
              Next
            </button>
          </div>
        </section>
      ) : null}
    </main>
  );
}
