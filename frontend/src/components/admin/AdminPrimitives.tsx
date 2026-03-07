"use client";

import type { ReactNode } from "react";

type AdminSectionProps = {
  children: ReactNode;
  className?: string;
};

type AdminDataTableProps = {
  children: ReactNode;
  loading?: boolean;
  loadingText?: string;
  isEmpty?: boolean;
  emptyText?: string;
};

type AdminEditorShellProps = {
  title: ReactNode;
  error?: string;
  children: ReactNode;
};

type AdminActionGroupProps = {
  children: ReactNode;
  layout?: "inline" | "row" | "stack";
  className?: string;
};

function mergeClassNames(...classes: Array<string | undefined>): string {
  return classes.filter(Boolean).join(" ").trim();
}

export function AdminSection({ children, className }: AdminSectionProps) {
  return <section className={mergeClassNames("card", className)}>{children}</section>;
}

export function AdminDataTable({
  children,
  loading = false,
  loadingText,
  isEmpty = false,
  emptyText,
}: AdminDataTableProps) {
  if (loading) {
    return loadingText ? <p>{loadingText}</p> : null;
  }

  if (isEmpty) {
    return emptyText ? <p className="empty-text">{emptyText}</p> : null;
  }

  return <div className="table-wrap">{children}</div>;
}

export function AdminEditorShell({ title, error, children }: AdminEditorShellProps) {
  return (
    <AdminSection>
      <h2>{title}</h2>
      {error ? <p className="error-text">{error}</p> : null}
      {children}
    </AdminSection>
  );
}

const actionGroupClassByLayout: Record<NonNullable<AdminActionGroupProps["layout"]>, string> = {
  inline: "inline-actions",
  row: "row-actions",
  stack: "stack-sm",
};

export function AdminActionGroup({
  children,
  layout = "inline",
  className,
}: AdminActionGroupProps) {
  return <div className={mergeClassNames(actionGroupClassByLayout[layout], className)}>{children}</div>;
}
