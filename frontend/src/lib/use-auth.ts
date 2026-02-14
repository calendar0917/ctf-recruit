"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  getStoredSession,
  subscribeAuthChange,
} from "@/lib/auth-storage";
import type { AuthSession } from "@/lib/types";

type UseRequireAuthOptions = {
  adminOnly?: boolean;
};

export function useAuthSession() {
  const [session, setSession] = useState<AuthSession | null>(null);
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const sync = () => {
      setSession(getStoredSession());
      setReady(true);
    };

    sync();

    const unsubscribe = subscribeAuthChange(sync);
    window.addEventListener("storage", sync);

    return () => {
      unsubscribe();
      window.removeEventListener("storage", sync);
    };
  }, []);

  return { session, ready };
}

export function useRequireAuth(options: UseRequireAuthOptions = {}) {
  const router = useRouter();
  const { session, ready } = useAuthSession();

  const isAdmin = session?.user.role === "admin";

  useEffect(() => {
    if (!ready) {
      return;
    }

    if (!session) {
      router.replace("/login");
      return;
    }

    if (options.adminOnly && !isAdmin) {
      router.replace("/challenges");
    }
  }, [isAdmin, options.adminOnly, ready, router, session]);

  const authorized = useMemo(() => {
    if (!ready || !session) {
      return false;
    }

    if (options.adminOnly) {
      return isAdmin;
    }

    return true;
  }, [isAdmin, options.adminOnly, ready, session]);

  return {
    session,
    ready,
    authorized,
  };
}
