"use client";

import { useEffect, useMemo, useState } from "react";
import { AdminSection } from "@/components/admin/AdminPrimitives";
import { AnnouncementEditor, type AnnouncementEditorValue } from "@/components/admin/AnnouncementEditor";
import { AnnouncementTable } from "@/components/admin/AnnouncementTable";
import {
  createAnnouncement,
  deleteAnnouncement,
  listAnnouncements,
  updateAnnouncement,
} from "@/lib/api/announcements";
import { HttpError } from "@/lib/http";
import type { Announcement, UpdateAnnouncementRequest } from "@/lib/types";
import { useRequireAuth } from "@/lib/use-auth";

function getAdminErrorMessage(err: unknown, fallback: string): string {
  if (!(err instanceof HttpError)) {
    return fallback;
  }

  if (err.status === 401) {
    return "Unauthorized. Please login again.";
  }

  if (err.status === 403) {
    return "Forbidden. Admin privileges are required for announcement management.";
  }

  return err.message;
}

export default function AdminAnnouncementsPage() {
  const { session, ready } = useRequireAuth();

  const [items, setItems] = useState<Announcement[]>([]);
  const [editing, setEditing] = useState<Announcement | null>(null);
  const [loadingList, setLoadingList] = useState(false);
  const [saving, setSaving] = useState(false);
  const [workingId, setWorkingId] = useState<string | undefined>();
  const [error, setError] = useState<string | undefined>();

  const isAdmin = session?.user.role === "admin";

  useEffect(() => {
    if (!ready || !session || !isAdmin) {
      return;
    }

    const accessToken = session.accessToken;
    let cancelled = false;

    async function run() {
      setLoadingList(true);
      setError(undefined);
      try {
        const response = await listAnnouncements(accessToken, {
          limit: 100,
          offset: 0,
        });
        if (!cancelled) {
          setItems(response.items);
        }
      } catch (err) {
        if (cancelled) {
          return;
        }
        setError(getAdminErrorMessage(err, "Failed to load announcements."));
      } finally {
        if (!cancelled) {
          setLoadingList(false);
        }
      }
    }

    void run();

    return () => {
      cancelled = true;
    };
  }, [isAdmin, ready, session]);

  async function refreshList(): Promise<void> {
    if (!session || !isAdmin) {
      return;
    }

    const response = await listAnnouncements(session.accessToken, {
      limit: 100,
      offset: 0,
    });
    setItems(response.items);
  }

  async function handleSave(value: AnnouncementEditorValue): Promise<void> {
    if (!session || !isAdmin) {
      return;
    }

    setSaving(true);
    setError(undefined);

    try {
      if (value.id) {
        const payload: UpdateAnnouncementRequest = {
          title: value.title,
          content: value.content,
          isPublished: value.isPublished,
          publishedAt: value.publishedAt,
        };
        await updateAnnouncement(session.accessToken, value.id, payload);
      } else {
        await createAnnouncement(session.accessToken, {
          title: value.title,
          content: value.content,
          isPublished: value.isPublished,
          publishedAt: value.publishedAt,
        });
      }

      await refreshList();
      setEditing(null);
    } catch (err) {
      setError(getAdminErrorMessage(err, "Failed to save announcement."));
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(announcement: Announcement): Promise<void> {
    if (!session || !isAdmin) {
      return;
    }

    setWorkingId(announcement.id);
    setError(undefined);
    try {
      await deleteAnnouncement(session.accessToken, announcement.id);
      await refreshList();
      if (editing?.id === announcement.id) {
        setEditing(null);
      }
    } catch (err) {
      setError(getAdminErrorMessage(err, "Failed to delete announcement."));
    } finally {
      setWorkingId(undefined);
    }
  }

  async function handleTogglePublish(announcement: Announcement): Promise<void> {
    if (!session || !isAdmin) {
      return;
    }

    setWorkingId(announcement.id);
    setError(undefined);

    try {
      await updateAnnouncement(session.accessToken, announcement.id, {
        isPublished: !announcement.isPublished,
      });
      await refreshList();
    } catch (err) {
      setError(getAdminErrorMessage(err, "Failed to update publish status."));
    } finally {
      setWorkingId(undefined);
    }
  }

  const sorted = useMemo(
    () => [...items].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt)),
    [items],
  );

  if (!ready) {
    return (
      <main className="page">
        <section className="card">
          <p>Loading session...</p>
        </section>
      </main>
    );
  }

  if (!session) {
    return (
      <main className="page">
        <section className="card">
          <p className="error-text">Unauthorized. Please login.</p>
        </section>
      </main>
    );
  }

  if (!isAdmin) {
    return (
      <main className="page">
        <section className="card">
          <h1>403</h1>
          <p className="error-text">Admin role required.</p>
        </section>
      </main>
    );
  }

  return (
    <main className="page page-content">
      <AdminSection>
        <h1>Admin: Announcement management</h1>
        <p>Create, update, delete, and publish announcements.</p>
      </AdminSection>

      <AnnouncementEditor
        initial={editing}
        loading={saving}
        error={error}
        onSubmit={handleSave}
        onCancelEdit={() => setEditing(null)}
      />

      <AdminSection>
        <h2>Existing announcements</h2>
        {loadingList ? <p>Loading announcements...</p> : null}
        {!loadingList ? (
          <AnnouncementTable
            items={sorted}
            workingId={workingId}
            onEdit={(announcement) => setEditing(announcement)}
            onDelete={handleDelete}
            onTogglePublish={handleTogglePublish}
          />
        ) : null}
      </AdminSection>
    </main>
  );
}
