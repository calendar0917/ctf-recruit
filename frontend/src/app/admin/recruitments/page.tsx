"use client";

import { useEffect, useState } from "react";
import {
  getRecruitmentSubmission,
  listRecruitmentSubmissions,
} from "@/lib/api/recruitments";
import { HttpError } from "@/lib/http";
import { useRequireAuth } from "@/lib/use-auth";
import type { RecruitmentSubmission } from "@/lib/types";

export default function AdminRecruitmentsPage() {
  const { session, ready } = useRequireAuth();
  const [items, setItems] = useState<RecruitmentSubmission[]>([]);
  const [selected, setSelected] = useState<RecruitmentSubmission | null>(null);
  const [loadingList, setLoadingList] = useState(false);
  const [loadingDetail, setLoadingDetail] = useState(false);
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
        const response = await listRecruitmentSubmissions(accessToken, { limit: 100, offset: 0 });
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
          setError("Failed to load recruitment submissions.");
        }
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

  async function handleOpenDetail(item: RecruitmentSubmission): Promise<void> {
    if (!session || !isAdmin) {
      return;
    }

    setLoadingDetail(true);
    setError(undefined);
    try {
      const detail = await getRecruitmentSubmission(session.accessToken, item.id);
      setSelected(detail);
    } catch (err) {
      if (err instanceof HttpError) {
        setError(err.message);
      } else {
        setError("Failed to load recruitment detail.");
      }
    } finally {
      setLoadingDetail(false);
    }
  }

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
      <section className="card">
        <h1>Admin: Recruitment submissions</h1>
        <p>View player recruitment form submissions.</p>
      </section>

      {error ? (
        <section className="card">
          <p className="error-text">{error}</p>
        </section>
      ) : null}

      <section className="card">
        <h2>Submissions</h2>
        {loadingList ? <p>Loading recruitment submissions...</p> : null}
        {!loadingList && items.length === 0 ? <p className="empty-text">No submissions yet.</p> : null}

        {!loadingList && items.length > 0 ? (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>姓名</th>
                  <th>学校</th>
                  <th>年级</th>
                  <th>方向</th>
                  <th>提交时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id}>
                    <td>{item.name}</td>
                    <td>{item.school}</td>
                    <td>{item.grade}</td>
                    <td>{item.direction}</td>
                    <td>{new Date(item.createdAt).toLocaleString()}</td>
                    <td>
                      <button
                        type="button"
                        className="button secondary"
                        onClick={() => void handleOpenDetail(item)}
                        disabled={loadingDetail}
                      >
                        查看详情
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </section>

      <section className="card">
        <h2>Submission detail</h2>
        {loadingDetail ? <p>Loading detail...</p> : null}
        {!loadingDetail && !selected ? <p className="empty-text">Select one submission to view full details.</p> : null}

        {!loadingDetail && selected ? (
          <div className="stack-sm">
            <p>
              <strong>姓名：</strong>
              {selected.name}
            </p>
            <p>
              <strong>学校：</strong>
              {selected.school}
            </p>
            <p>
              <strong>年级：</strong>
              {selected.grade}
            </p>
            <p>
              <strong>方向：</strong>
              {selected.direction}
            </p>
            <p>
              <strong>联系方式：</strong>
              {selected.contact}
            </p>
            <p className="announcement-content-wrap">
              <strong>个人简介：</strong>
              {selected.bio}
            </p>
          </div>
        ) : null}
      </section>
    </main>
  );
}
