"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import Link from "next/link";
import { AuthForm } from "@/components/auth/AuthForm";
import { login } from "@/lib/api/auth";
import { setStoredSession } from "@/lib/auth-storage";
import { HttpError } from "@/lib/http";

export default function LoginPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | undefined>();

  async function handleLogin(payload: {
    email: string;
    password: string;
  }): Promise<void> {
    setLoading(true);
    setError(undefined);
    try {
      const session = await login(payload);
      setStoredSession(session);
      router.push("/challenges");
    } catch (err) {
      if (err instanceof HttpError) {
        setError(err.message);
      } else {
        setError("Login failed. Please try again.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="page">
      <AuthForm
        mode="login"
        title="Welcome back"
        submitLabel="Sign in"
        loading={loading}
        error={error}
        onSubmit={handleLogin}
      />
      <p className="helper-text">
        No account yet? <Link href="/register">Create one</Link>
      </p>
    </main>
  );
}
