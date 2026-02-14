"use client";

import type { ScoreboardItem } from "@/lib/types";

type ScoreboardTableProps = {
  items: ScoreboardItem[];
};

export function ScoreboardTable({ items }: ScoreboardTableProps) {
  if (items.length === 0) {
    return <p className="empty-text">No scoreboard entries yet.</p>;
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Rank</th>
            <th>Player</th>
            <th>Total Points</th>
            <th>Solved</th>
          </tr>
        </thead>
        <tbody>
          {items.map((row) => (
            <tr key={row.userId}>
              <td>{row.rank}</td>
              <td>{row.displayName}</td>
              <td>{row.totalPoints}</td>
              <td>{row.solvedCount}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
