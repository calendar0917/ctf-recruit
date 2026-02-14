"use client";

import type { FormEvent } from "react";
import { useState } from "react";

type AuthFormMode = "login" | "register";

type AuthFormValue = {
  email: string;
  password: string;
  displayName?: string;
};

type AuthFormProps = {
  mode: AuthFormMode;
  title: string;
  submitLabel: string;
  loading: boolean;
  error?: string;
  info?: string;
  onSubmit: (value: AuthFormValue) => Promise<void>;
};

export function AuthForm(props: AuthFormProps) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");

  const isRegister = props.mode === "register";

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    await props.onSubmit({
      email: email.trim(),
      password,
      displayName: isRegister ? displayName.trim() : undefined,
    });
  }

  return (
    <section className="card auth-card">
      <h1>{props.title}</h1>

      {props.info ? <p className="info-text">{props.info}</p> : null}
      {props.error ? <p className="error-text">{props.error}</p> : null}

      <form className="form-grid" onSubmit={handleSubmit}>
        {isRegister ? (
          <label>
            <span>Display Name</span>
            <input
              type="text"
              value={displayName}
              onChange={(event) => setDisplayName(event.target.value)}
              required
              disabled={props.loading}
            />
          </label>
        ) : null}

        <label>
          <span>Email</span>
          <input
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            required
            disabled={props.loading}
          />
        </label>

        <label>
          <span>Password</span>
          <input
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
            disabled={props.loading}
          />
        </label>

        <button type="submit" className="button" disabled={props.loading}>
          {props.loading ? "Submitting..." : props.submitLabel}
        </button>
      </form>
    </section>
  );
}
