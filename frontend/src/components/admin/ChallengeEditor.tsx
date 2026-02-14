"use client";

import type { FormEvent } from "react";
import { useEffect, useState } from "react";
import type {
  Challenge,
  ChallengeDifficulty,
  ChallengeMode,
  CreateChallengeRequest,
} from "@/lib/types";

export type ChallengeEditorValue = CreateChallengeRequest & { id?: string };

type ChallengeEditorProps = {
  initial?: Challenge | null;
  loading: boolean;
  error?: string;
  onSubmit: (value: ChallengeEditorValue) => Promise<void>;
  onCancelEdit: () => void;
};

const difficulties: ChallengeDifficulty[] = ["easy", "medium", "hard"];
const modes: ChallengeMode[] = ["static", "dynamic"];

function toDefaultValue(challenge?: Challenge | null): ChallengeEditorValue {
  if (!challenge) {
    return {
      title: "",
      description: "",
      category: "",
      difficulty: "easy",
      mode: "static",
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
    points: challenge.points,
    flag: "",
    isPublished: challenge.isPublished,
  };
}

export function ChallengeEditor({
  initial,
  loading,
  error,
  onSubmit,
  onCancelEdit,
}: ChallengeEditorProps) {
  const [value, setValue] = useState<ChallengeEditorValue>(toDefaultValue(initial));

  useEffect(() => {
    setValue(toDefaultValue(initial));
  }, [initial]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await onSubmit({
      ...value,
      title: value.title.trim(),
      description: value.description.trim(),
      category: value.category.trim(),
      flag: value.flag.trim(),
    });
  }

  const editing = Boolean(initial);

  return (
    <section className="card">
      <h2>{editing ? "Edit challenge" : "Create challenge"}</h2>
      {error ? <p className="error-text">{error}</p> : null}

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

        <div className="row-actions">
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
        </div>
      </form>
    </section>
  );
}
