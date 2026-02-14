"use client";

import type { Challenge } from "@/lib/types";

type ChallengeTableProps = {
  items: Challenge[];
  workingId?: string;
  onEdit: (challenge: Challenge) => void;
  onDelete: (challenge: Challenge) => Promise<void>;
  onTogglePublish: (challenge: Challenge) => Promise<void>;
};

export function ChallengeTable({
  items,
  workingId,
  onEdit,
  onDelete,
  onTogglePublish,
}: ChallengeTableProps) {
  if (items.length === 0) {
    return <p className="empty-text">No challenges found.</p>;
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Title</th>
            <th>Category</th>
            <th>Difficulty</th>
            <th>Mode</th>
            <th>Points</th>
            <th>Published</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => {
            const busy = workingId === item.id;
            return (
              <tr key={item.id}>
                <td>{item.title}</td>
                <td>{item.category}</td>
                <td>{item.difficulty}</td>
                <td>{item.mode}</td>
                <td>{item.points}</td>
                <td>{item.isPublished ? "Yes" : "No"}</td>
                <td>
                  <div className="inline-actions">
                    <button
                      type="button"
                      className="button secondary"
                      onClick={() => onEdit(item)}
                      disabled={busy}
                    >                      Edit
                    </button>
                    <button
                      type="button"
                      className="button secondary"
                      onClick={() => {
                        void onTogglePublish(item);
                      }}
                      disabled={busy}
                    >
                      {item.isPublished ? "Unpublish" : "Publish"}
                    </button>
                    <button
                      type="button"
                      className="button danger"
                      onClick={() => {
                        void onDelete(item);
                      }}
                      disabled={busy}
                    >
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
