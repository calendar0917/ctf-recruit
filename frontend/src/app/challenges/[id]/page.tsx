"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { ChallengeDetail } from "@/components/challenge/ChallengeDetail";
import { getChallenge } from "@/lib/api/challenges";
import { listScoreboard } from "@/lib/api/scoreboard";
import { HttpError } from "@/lib/http";
import { useRequireAuth } from "@/lib/use-auth";
import type { Challenge, ScoreboardItem, SubmissionResponse } from "@/lib/types";

export default function ChallengeDetailPage() {
  const params = useParams<{ id: string }>();
  const challengeId = params?.id;

  const { session, ready, authorized } = useRequireAuth();

  const [challenge, setChallenge] = useState<Challenge | null>(null);
  const [scoreboard, setScoreboard] = useState<ScoreboardItem[]>([]);
  const [latestSubmission, setLatestSubmission] = useState<SubmissionResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const [refreshError, setRefreshError] = useState<string | undefined>();

  useEffect(() => {
    if (!ready || !authorized || !session || !challengeId) {
      return;
    }

    const accessToken = session.accessToken;
    let cancelled = false;

    async function run() {
      setLoading(true);
      setError(undefined);

      try {
        const [challengeResp, scoreboardResp] = await Promise.all([
          getChallenge(accessToken, challengeId),
          listScoreboard(accessToken, { limit: 5, offset: 0 }),
        ]);

        if (!cancelled) {
          setChallenge(challengeResp);
          setScoreboard(scoreboardResp.items);
        }
      } catch (err) {
        if (cancelled) {
          return;
        }
        if (err instanceof HttpError) {
          setError(err.message);
        } else {
          setError("Failed to load challenge.");
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
  }, [authorized, challengeId, ready, session]);

  async function handleManualRefresh(): Promise<void> {
    if (!session || !challengeId) {
      return;
    }

    setRefreshing(true);
    setRefreshError(undefined);

    try {
      const [challengeResp, scoreboardResp] = await Promise.all([
        getChallenge(session.accessToken, challengeId),
        listScoreboard(session.accessToken, { limit: 5, offset: 0 }),
      ]);
      setChallenge(challengeResp);
      setScoreboard(scoreboardResp.items);
    } catch (err) {
      if (err instanceof HttpError) {
        setRefreshError(err.message);
      } else {
        setRefreshError("Refresh failed.");
      }
    } finally {
      setRefreshing(false);
    }
  }

  if (!ready || !authorized || !session) {
    return (
      <main className="page">
        <section className="card">
          <p>Loading session...</p>
        </section>
      </main>
    );
  }

  if (!challengeId) {
    return (
      <main className="page">
        <section className="card">
          <p className="error-text">Challenge ID is missing.</p>
        </section>
      </main>
    );
  }

  if (loading) {
    return (
      <main className="page">
        <section className="card">
          <p>Loading challenge...</p>
        </section>
      </main>
    );
  }

  if (error || !challenge) {
    return (
      <main className="page">
        <section className="card">
          <p className="error-text">{error ?? "Challenge not found."}</p>
        </section>
      </main>
    );
  }

  return (
    <main className="page page-content">
      <ChallengeDetail
        token={session.accessToken}
        challenge={challenge}
        latestSubmission={latestSubmission}
        topScoreboard={scoreboard}
        refreshing={refreshing}
        refreshError={refreshError}
        onSubmitted={(result) => {
          setLatestSubmission(result);
        }}
        onManualRefresh={handleManualRefresh}
      />
    </main>
  );
}
