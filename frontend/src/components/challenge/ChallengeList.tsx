"use client";

import Link from "next/link";
import type { Challenge } from "@/lib/types";

type ChallengeListProps = {
  items: Challenge[];
};

export function ChallengeList({ items }: ChallengeListProps) {
  if (items.length === 0) {
    return <p className="empty-text">No challenges found.</p>;
  }

  return (
    <div className="list-grid">
      {items.map((challenge) => (
        <article key={challenge.id} className="card challenge-card">
          <div className="challenge-meta">
            <span>{challenge.category}</span>
            <span>{challenge.difficulty}</span>
            <span>{challenge.mode}</span>
            <span>{challenge.points} pts</span>
          </div>
          <h2>{challenge.title}</h2>
          <p className="clamp-3">{challenge.description}</p>
          <Link className="button" href={`/challenges/${challenge.id}`}>
            View challenge
          </Link>
        </article>
      ))}
    </div>
  );
}
