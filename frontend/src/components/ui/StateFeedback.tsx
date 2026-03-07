"use client";

import type { ReactNode } from "react";

type StateVariant = "loading" | "error" | "empty" | "info";

type StateTextProps = {
  variant: StateVariant;
  children: ReactNode;
  className?: string;
};

type StateCardProps = {
  variant: StateVariant;
  children: ReactNode;
  className?: string;
};

type StateMessageCardProps = {
  message: ReactNode;
  className?: string;
};

const textClassByVariant: Record<StateVariant, string | undefined> = {
  loading: undefined,
  error: "error-text",
  empty: "empty-text",
  info: "info-text",
};

function mergeClassNames(...classes: Array<string | undefined>): string | undefined {
  const merged = classes.filter(Boolean).join(" ").trim();
  return merged.length > 0 ? merged : undefined;
}

export function StateText({ variant, children, className }: StateTextProps) {
  const combinedClassName = mergeClassNames(textClassByVariant[variant], className);
  return combinedClassName ? <p className={combinedClassName}>{children}</p> : <p>{children}</p>;
}

export function StateCard({ variant, children, className }: StateCardProps) {
  const combinedClassName = mergeClassNames("card", className);

  return (
    <section className={combinedClassName}>
      <StateText variant={variant}>{children}</StateText>
    </section>
  );
}

export function LoadingStateCard({ message, className }: StateMessageCardProps) {
  return (
    <StateCard variant="loading" className={className}>
      {message}
    </StateCard>
  );
}

export function ErrorStateCard({ message, className }: StateMessageCardProps) {
  return (
    <StateCard variant="error" className={className}>
      {message}
    </StateCard>
  );
}

export function EmptyStateCard({ message, className }: StateMessageCardProps) {
  return (
    <StateCard variant="empty" className={className}>
      {message}
    </StateCard>
  );
}

export function InfoStateCard({ message, className }: StateMessageCardProps) {
  return (
    <StateCard variant="info" className={className}>
      {message}
    </StateCard>
  );
}
