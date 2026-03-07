"use client";

import { AdminActionGroup, AdminDataTable } from "@/components/admin/AdminPrimitives";
import type { Announcement } from "@/lib/types";

type AnnouncementTableProps = {
  items: Announcement[];
  workingId?: string;
  onEdit: (announcement: Announcement) => void;
  onDelete: (announcement: Announcement) => Promise<void>;
  onTogglePublish: (announcement: Announcement) => Promise<void>;
};

function formatDate(value?: string): string {
  if (!value) {
    return "-";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

export function AnnouncementTable({
  items,
  workingId,
  onEdit,
  onDelete,
  onTogglePublish,
}: AnnouncementTableProps) {
  return (
    <AdminDataTable isEmpty={items.length === 0} emptyText="No announcements found.">
      <table>
        <thead>
          <tr>
            <th>Title</th>
            <th>Published</th>
            <th>Published At</th>
            <th>Updated At</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => {
            const busy = workingId === item.id;
            return (
              <tr key={item.id}>
                <td>{item.title}</td>
                <td>{item.isPublished ? "Yes" : "No"}</td>
                <td>{formatDate(item.publishedAt)}</td>
                <td>{formatDate(item.updatedAt)}</td>
                <td>
                  <AdminActionGroup className="table-actions">
                    <button
                      type="button"
                      className="button secondary"
                      onClick={() => onEdit(item)}
                      disabled={busy}
                    >
                      Edit
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
                  </AdminActionGroup>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </AdminDataTable>
  );
}
