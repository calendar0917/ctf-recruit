"use client";

import { type FormEvent, useMemo, useState } from "react";
import { createRecruitmentSubmission } from "@/lib/api/recruitments";
import { HttpError } from "@/lib/http";
import {
  getMissingRecruitmentFields,
  normalizeRecruitmentFormValue,
} from "@/lib/recruitment-form";
import { useRequireAuth } from "@/lib/use-auth";

type FormState = {
  name: string;
  school: string;
  grade: string;
  direction: string;
  contact: string;
  bio: string;
};

const INITIAL_FORM: FormState = {
  name: "",
  school: "",
  grade: "",
  direction: "",
  contact: "",
  bio: "",
};

export default function RecruitmentPage() {
  const { session, ready, authorized } = useRequireAuth();
  const [form, setForm] = useState<FormState>(INITIAL_FORM);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const [successMessage, setSuccessMessage] = useState<string | undefined>();

  const missingFields = useMemo(() => getMissingRecruitmentFields(form), [form]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();

    if (!session) {
      setError("Unauthorized. Please login.");
      return;
    }

    if (missingFields.length > 0) {
      setError(`请填写必填项：${missingFields.join("、")}`);
      setSuccessMessage(undefined);
      return;
    }

    setSaving(true);
    setError(undefined);
    setSuccessMessage(undefined);

    try {
      await createRecruitmentSubmission(session.accessToken, normalizeRecruitmentFormValue(form));

      setForm(INITIAL_FORM);
      setSuccessMessage("报名提交成功，我们会尽快查看你的信息。");
    } catch (err) {
      if (err instanceof HttpError) {
        setError(err.message);
      } else {
        setError("Failed to submit recruitment form.");
      }
    } finally {
      setSaving(false);
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

  return (
    <main className="page page-content">
      <section className="card">
        <h1>招新报名</h1>
        <p>填写以下信息完成报名提交。</p>
      </section>

      <section className="card">
        <form className="form-grid" onSubmit={handleSubmit}>
          <label>
            姓名
            <input
              name="name"
              value={form.name}
              onChange={(event) => setForm((prev) => ({ ...prev, name: event.target.value }))}
            />
          </label>

          <label>
            学校
            <input
              name="school"
              value={form.school}
              onChange={(event) => setForm((prev) => ({ ...prev, school: event.target.value }))}
            />
          </label>

          <label>
            年级
            <input
              name="grade"
              value={form.grade}
              onChange={(event) => setForm((prev) => ({ ...prev, grade: event.target.value }))}
            />
          </label>

          <label>
            方向
            <input
              name="direction"
              value={form.direction}
              onChange={(event) => setForm((prev) => ({ ...prev, direction: event.target.value }))}
            />
          </label>

          <label>
            联系方式
            <input
              name="contact"
              value={form.contact}
              onChange={(event) => setForm((prev) => ({ ...prev, contact: event.target.value }))}
            />
          </label>

          <label>
            个人简介
            <textarea
              name="bio"
              rows={5}
              value={form.bio}
              onChange={(event) => setForm((prev) => ({ ...prev, bio: event.target.value }))}
            />
          </label>

          {error ? <p className="error-text">{error}</p> : null}
          {successMessage ? <p className="info-text">{successMessage}</p> : null}

          <button type="submit" className="button" disabled={saving}>
            {saving ? "Submitting..." : "提交报名"}
          </button>
        </form>
      </section>
    </main>
  );
}
