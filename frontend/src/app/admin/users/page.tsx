"use client";

import React, { useEffect, useMemo, useState } from "react";
import { AdminActionGroup, AdminDataTable, AdminSection } from "@/components/admin/AdminPrimitives";
import { ErrorStateCard, LoadingStateCard } from "@/components/ui/StateFeedback";
import { listAdminUsers, updateAdminUser } from "@/lib/api/admin-users";
import { HttpError } from "@/lib/http";
import type { Role, User } from "@/lib/types";
import { useRequireAuth } from "@/lib/use-auth";

function nextRole(role: Role): Role {
  return role === "admin" ? "player" : "admin";
}

void React;

export default function AdminUsersPage() {
  const { session, ready } = useRequireAuth({ adminOnly: true });
  const [items, setItems] = useState<User[]>([]);
  const [loadingList, setLoadingList] = useState(false);
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
        const resp = await listAdminUsers(accessToken, { limit: 100, offset: 0 });
        if (!cancelled) {
          setItems(resp.items);
        }
      } catch (err) {
        if (cancelled) {
          return;
        }
        setError(err instanceof HttpError ? err.message : "Failed to load users.");
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
    const resp = await listAdminUsers(session.accessToken, { limit: 100, offset: 0 });
    setItems(resp.items);
  }

  async function handleToggleRole(user: User): Promise<void> {
    if (!session || !isAdmin) {
      return;
    }
    setWorkingId(user.id);
    setError(undefined);
    try {
      await updateAdminUser(session.accessToken, user.id, { role: nextRole(user.role) });
      await refreshList();
    } catch (err) {
      setError(err instanceof HttpError ? err.message : "Failed to update role.");
    } finally {
      setWorkingId(undefined);
    }
  }

  async function handleToggleDisable(user: User): Promise<void> {
    if (!session || !isAdmin) {
      return;
    }
    setWorkingId(user.id);
    setError(undefined);
    try {
      await updateAdminUser(session.accessToken, user.id, { isDisabled: !user.isDisabled });
      await refreshList();
    } catch (err) {
      setError(err instanceof HttpError ? err.message : "Failed to update user status.");
    } finally {
      setWorkingId(undefined);
    }
  }

  const sorted = useMemo(() => [...items].sort((a, b) => a.email.localeCompare(b.email)), [items]);

  if (!ready) {
    return (
      <main className="page">
        <LoadingStateCard message="Loading session..." />
      </main>
    );
  }

  if (!session) {
    return (
      <main className="page">
        <ErrorStateCard message="Unauthorized. Please login." />
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
        <h1>Admin: User management</h1>
        <p>List users, change roles, and disable/enable accounts.</p>
      </AdminSection>

      {error ? (
        <ErrorStateCard message={error} />
      ) : null}

      <AdminSection>
        <h2>Users</h2>
        <AdminDataTable
          loading={loadingList}
          loadingText="Loading users..."
          isEmpty={!loadingList && sorted.length === 0}
          emptyText="No users found."
        >
          <table>
            <thead>
              <tr>
                <th>Email</th>
                <th>Display Name</th>
                <th>Role</th>
                <th>Disabled</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((user) => (
                <tr key={user.id}>
                  <td>{user.email}</td>
                  <td>{user.displayName}</td>
                  <td>{user.role}</td>
                  <td>{user.isDisabled ? "yes" : "no"}</td>
                  <td>
                    <AdminActionGroup layout="stack" className="table-actions">
                      <button
                        type="button"
                        className="button secondary"
                        onClick={() => void handleToggleRole(user)}
                        disabled={workingId === user.id}
                      >
                        Set {nextRole(user.role)}
                      </button>
                      <button
                        type="button"
                        className="button secondary"
                        onClick={() => void handleToggleDisable(user)}
                        disabled={workingId === user.id}
                      >
                        {user.isDisabled ? "Enable" : "Disable"}
                      </button>
                    </AdminActionGroup>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </AdminDataTable>
      </AdminSection>
    </main>
  );
}
