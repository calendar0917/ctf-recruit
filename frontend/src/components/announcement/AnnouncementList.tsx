"use client";

import Link from "next/link";
import React from "react";
import type { Announcement } from "@/lib/types";

type AnnouncementListProps = {
  items: Announcement[];
};

void React;

function formatDate(value?: string): string {
  if (!value) {
    return "Draft";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

export function AnnouncementList({ items }: AnnouncementListProps) {
  if (items.length === 0) {
    return <p className="empty-text">No announcements found.</p>;
  }

  return (
    <div className="list-grid">
      {items.map((item) => (
        <article key={item.id} className="card surface-card">
          <div className="surface-meta">
            <span>{item.isPublished ? "Published" : "Draft"}</span>
            <span>{formatDate(item.publishedAt)}</span>
          </div>
          <h2>{item.title}</h2>
          <p className="clamp-3 announcement-content">{item.content}</p>
          <Link className="button" href={`/announcements/${item.id}`}>
            Read announcement
          </Link>
        </article>
      ))}
    </div>
  );
}
