"use client";

import type { FormEvent } from "react";
import { useState } from "react";
import { createSubmission } from "@/lib/api/submissions";
import { HttpError } from "@/lib/http";
import type { SubmissionResponse } from "@/lib/types";

type SubmissionFormProps = {
  token: string;
  challengeId: string;
  onSubmitted: (result: SubmissionResponse) => void;
};

export function SubmissionForm({ token, challengeId, onSubmitted }: SubmissionFormProps) {
  const [flag, setFlag] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | undefined>();

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError(undefined);

    try {
      const response = await createSubmission(token, {
        challengeId,
        flag: flag.trim(),
      });
      onSubmitted(response);
      setFlag("");
    } catch (err) {
      if (err instanceof HttpError) {
        setError(err.message);
      } else {
        setError("Submission failed. Please try again.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <section className="card">
      <h3>Submit flag</h3>
      {error ? <p className="error-text">{error}</p> : null}
      <form className="form-grid" onSubmit={handleSubmit}>
        <label>
          <span>Flag</span>
          <input
            value={flag}
            onChange={(event) => setFlag(event.target.value)}
            placeholder="CTF{...}"
            required
            disabled={loading}
          />
        </label>
        <button type="submit" className="button" disabled={loading}>
          {loading ? "Submitting..." : "Submit"}
        </button>
      </form>
    </section>
  );
}
