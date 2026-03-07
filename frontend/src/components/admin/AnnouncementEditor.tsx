"use client";

import type { FormEvent } from "react";
import { useEffect, useState } from "react";
import { AdminActionGroup, AdminEditorShell } from "@/components/admin/AdminPrimitives";
import type { Announcement, CreateAnnouncementRequest } from "@/lib/types";

export type AnnouncementEditorValue = CreateAnnouncementRequest & { id?: string };

type AnnouncementEditorProps = {
  initial?: Announcement | null;
  loading: boolean;
  error?: string;
  onSubmit: (value: AnnouncementEditorValue) => Promise<void>;
  onCancelEdit: () => void;
};

function toDefaultValue(announcement?: Announcement | null): AnnouncementEditorValue {
  if (!announcement) {
    return {
      title: "",
      content: "",
      isPublished: false,
      publishedAt: undefined,
    };
  }

  return {
    id: announcement.id,
    title: announcement.title,
    content: announcement.content,
    isPublished: announcement.isPublished,
    publishedAt: announcement.publishedAt,
  };
}

export function AnnouncementEditor({
  initial,
  loading,
  error,
  onSubmit,
  onCancelEdit,
}: AnnouncementEditorProps) {
  const [value, setValue] = useState<AnnouncementEditorValue>(toDefaultValue(initial));

  useEffect(() => {
    setValue(toDefaultValue(initial));
  }, [initial]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await onSubmit({
      ...value,
      title: value.title.trim(),
      content: value.content.trim(),
      publishedAt: value.publishedAt?.trim() || undefined,
    });
  }

  const editing = Boolean(initial);

  return (
    <AdminEditorShell title={editing ? "Edit announcement" : "Create announcement"} error={error}>
      <form className="form-grid" onSubmit={handleSubmit}>
        <label>
          <span>Title</span>
          <input
            value={value.title}
            onChange={(event) => setValue((prev) => ({ ...prev, title: event.target.value }))}
            required
            disabled={loading}
          />
        </label>

        <label>
          <span>Content</span>
          <textarea
            value={value.content}
            onChange={(event) => setValue((prev) => ({ ...prev, content: event.target.value }))}
            rows={8}
            required
            disabled={loading}
          />
        </label>

        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={value.isPublished}
            onChange={(event) =>
              setValue((prev) => ({
                ...prev,
                isPublished: event.target.checked,
                publishedAt: event.target.checked ? prev.publishedAt : undefined,
              }))
            }
            disabled={loading}
          />
          <span>Published</span>
        </label>

        <label>
          <span>Published at (RFC3339, optional)</span>
          <input
            placeholder="2026-02-16T10:00:00Z"
            value={value.publishedAt ?? ""}
            onChange={(event) => setValue((prev) => ({ ...prev, publishedAt: event.target.value }))}
            disabled={loading || !value.isPublished}
          />
        </label>

        <AdminActionGroup layout="row">
          <button className="button" type="submit" disabled={loading}>
            {loading ? "Saving..." : editing ? "Update announcement" : "Create announcement"}
          </button>

          {editing ? (
            <button
              className="button secondary"
              type="button"
              onClick={onCancelEdit}
              disabled={loading}
            >
              Cancel edit
            </button>
          ) : null}
        </AdminActionGroup>
      </form>
    </AdminEditorShell>
  );
}
