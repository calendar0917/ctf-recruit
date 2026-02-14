"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AuthForm } from "@/components/auth/AuthForm";
import { register } from "@/lib/api/auth";
import { HttpError } from "@/lib/http";

export default function RegisterPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | undefined>();
  const [info, setInfo] = useState<string | undefined>();

  async function handleRegister(payload: {
    email: string;
    password: string;
    displayName?: string;
  }): Promise<void> {
    setLoading(true);
    setError(undefined);
    setInfo(undefined);
    try {
      await register({
        email: payload.email,
        password: payload.password,
        displayName: payload.displayName ?? "",
      });
      setInfo("Registration successful. Please sign in.");
      router.push("/login");
    } catch (err) {
      if (err instanceof HttpError) {
        setError(err.message);
      } else {
        setError("Registration failed. Please try again.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="page">
      <AuthForm
        mode="register"
        title="Create account"
        submitLabel="Register"
        loading={loading}
        error={error}
        info={info}
        onSubmit={handleRegister}
      />
      <p className="helper-text">
        Already registered? <Link href="/login">Sign in</Link>
      </p>
    </main>
  );
}
