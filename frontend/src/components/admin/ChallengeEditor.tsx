"use client";

import React, { type FormEvent, useEffect, useState } from "react";
import { AdminActionGroup, AdminEditorShell } from "@/components/admin/AdminPrimitives";
import type {
  AdminChallengeEditorPayload,
  Challenge,
  ChallengeDifficulty,
  ChallengeMode,
  ChallengeRuntimeFieldErrors,
} from "@/lib/types";

export type ChallengeEditorValue = AdminChallengeEditorPayload;

type ChallengeEditorProps = {
  initial?: Challenge | null;
  loading: boolean;
  error?: string;
  fieldErrors?: ChallengeRuntimeFieldErrors;
  onSubmit: (value: ChallengeEditorValue) => Promise<void>;
  onCancelEdit: () => void;
};

const difficulties: ChallengeDifficulty[] = ["easy", "medium", "hard"];
const modes: ChallengeMode[] = ["static", "dynamic"];

void React;

function toDefaultValue(challenge?: Challenge | null): ChallengeEditorValue {
  if (!challenge) {
    return {
      title: "",
      description: "",
      category: "",
      difficulty: "easy",
      mode: "static",
      runtimeImage: "",
      runtimeCommand: "",
      runtimeExposedPort: undefined,
      points: 100,
      flag: "",
      isPublished: false,
    };
  }

  return {
    id: challenge.id,
    title: challenge.title,
    description: challenge.description,
    category: challenge.category,
    difficulty: challenge.difficulty,
    mode: challenge.mode,
    runtimeImage: challenge.runtimeImage ?? "",
    runtimeCommand: challenge.runtimeCommand ?? "",
    runtimeExposedPort: challenge.runtimeExposedPort,
    points: challenge.points,
    flag: "",
    isPublished: challenge.isPublished,
  };
}

export function ChallengeEditor({
  initial,
  loading,
  error,
  fieldErrors,
  onSubmit,
  onCancelEdit,
}: ChallengeEditorProps) {
  const [value, setValue] = useState<ChallengeEditorValue>(toDefaultValue(initial));

  useEffect(() => {
    setValue(toDefaultValue(initial));
  }, [initial]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await onSubmit(value);
  }

  const editing = Boolean(initial);

  return (
    <AdminEditorShell title={editing ? "Edit challenge" : "Create challenge"} error={error}>
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
          <span>Description</span>
          <textarea
            value={value.description}
            onChange={(event) =>
              setValue((prev) => ({ ...prev, description: event.target.value }))
            }
            rows={5}
            required
            disabled={loading}
          />
        </label>

        <label>
          <span>Category</span>
          <input
            value={value.category}
            onChange={(event) => setValue((prev) => ({ ...prev, category: event.target.value }))}
            required
            disabled={loading}
          />
        </label>

        <div className="row-fields">
          <label>
            <span>Difficulty</span>
            <select
              value={value.difficulty}
              onChange={(event) =>
                setValue((prev) => ({
                  ...prev,
                  difficulty: event.target.value as ChallengeDifficulty,
                }))
              }
              disabled={loading}
            >
              {difficulties.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
          </label>

          <label>
            <span>Mode</span>
            <select
              value={value.mode}
              onChange={(event) =>
                setValue((prev) => ({ ...prev, mode: event.target.value as ChallengeMode }))
              }
              disabled={loading}
            >
              {modes.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
          </label>

          <label>
            <span>Points</span>
            <input
              type="number"
              min={1}
              value={value.points}
              onChange={(event) =>
                setValue((prev) => ({ ...prev, points: Number(event.target.value) }))
              }
              required
              disabled={loading}
            />
          </label>
        </div>

        <div className="row-fields">
          <label>
            <span>Runtime image (optional)</span>
            <input
              value={value.runtimeImage ?? ""}
              onChange={(event) =>
                setValue((prev) => ({ ...prev, runtimeImage: event.target.value }))
              }
              disabled={loading}
              placeholder="busybox:1.36"
            />
            {fieldErrors?.runtimeImage ? (
              <small className="error-text">{fieldErrors.runtimeImage}</small>
            ) : null}
          </label>

          <label>
            <span>Runtime command (optional)</span>
            <input
              value={value.runtimeCommand ?? ""}
              onChange={(event) =>
                setValue((prev) => ({ ...prev, runtimeCommand: event.target.value }))
              }
              disabled={loading}
              placeholder="httpd -f -p 8080"
            />
            {fieldErrors?.runtimeCommand ? (
              <small className="error-text">{fieldErrors.runtimeCommand}</small>
            ) : null}
          </label>

          <label>
            <span>Runtime exposed port (optional)</span>
            <input
              type="number"
              min={1}
              value={value.runtimeExposedPort ?? ""}
              onChange={(event) => {
                const next = event.target.value;
                setValue((prev) => ({
                  ...prev,
                  runtimeExposedPort: next === "" ? undefined : Number(next),
                }));
              }}
              disabled={loading}
              placeholder="8080"
            />
            {fieldErrors?.runtimeExposedPort ? (
              <small className="error-text">{fieldErrors.runtimeExposedPort}</small>
            ) : null}
          </label>
        </div>

        <label>
          <span>{editing ? "Flag (leave empty to keep)" : "Flag"}</span>
          <input
            value={value.flag}
            onChange={(event) => setValue((prev) => ({ ...prev, flag: event.target.value }))}
            required={!editing}
            disabled={loading}
          />
        </label>

        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={value.isPublished}
            onChange={(event) =>
              setValue((prev) => ({ ...prev, isPublished: event.target.checked }))
            }
            disabled={loading}
          />
          <span>Published</span>
        </label>

        <AdminActionGroup layout="row">
          <button className="button" type="submit" disabled={loading}>
            {loading ? "Saving..." : editing ? "Update challenge" : "Create challenge"}
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
