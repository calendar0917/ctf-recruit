"use client";

import type { Challenge, ScoreboardItem, SubmissionResponse } from "@/lib/types";
import { SubmissionForm } from "@/components/submission/SubmissionForm";

type ChallengeDetailProps = {
  token: string;
  challenge: Challenge;
  latestSubmission: SubmissionResponse | null;
  topScoreboard: ScoreboardItem[];
  refreshing: boolean;
  refreshError?: string;
  onSubmitted: (result: SubmissionResponse) => void;
  onManualRefresh: () => Promise<void>;
};

export function ChallengeDetail({
  token,
  challenge,
  latestSubmission,
  topScoreboard,
  refreshing,
  refreshError,
  onSubmitted,
  onManualRefresh,
}: ChallengeDetailProps) {
  return (
    <div className="stack-lg">
      <section className="card">
        <div className="challenge-meta">
          <span>{challenge.category}</span>
          <span>{challenge.difficulty}</span>
          <span>{challenge.mode}</span>
          <span>{challenge.points} pts</span>
        </div>
        <h1>{challenge.title}</h1>
        <p>{challenge.description}</p>
      </section>

      <SubmissionForm token={token} challengeId={challenge.id} onSubmitted={onSubmitted} />

      {latestSubmission ? (
        <section className="card">
          <h3>Latest submission result</h3>
          <p>
            Status: <strong>{latestSubmission.status}</strong>
          </p>
          <p>Awarded points: {latestSubmission.awardedPoints}</p>
          {latestSubmission.judgeJobId ? <p>Judge job: {latestSubmission.judgeJobId}</p> : null}

          {latestSubmission.status === "pending" ? (
            <p className="info-text">
              Dynamic judging is pending. Click refresh to fetch latest challenge and scoreboard status.
            </p>
          ) : null}

          {refreshError ? <p className="error-text">{refreshError}</p> : null}

          <button
            type="button"
            className="button secondary"
            onClick={() => {
              void onManualRefresh();
            }}
            disabled={refreshing}
          >
            {refreshing ? "Refreshing..." : "Manual refresh"}
          </button>
        </section>
      ) : null}

      <section className="card">
        <h3>Top scoreboard (preview)</h3>
        {topScoreboard.length === 0 ? (
          <p className="empty-text">No ranking data yet.</p>
        ) : (
          <ul className="score-list">
            {topScoreboard.map((item) => (
              <li key={item.userId}>
                #{item.rank} {item.displayName} — {item.totalPoints} pts ({item.solvedCount} solved)
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
