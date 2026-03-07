"use client";

import React from "react";
import type { Announcement } from "@/lib/types";

type AnnouncementDetailProps = {
  announcement: Announcement;
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

export function AnnouncementDetail({ announcement }: AnnouncementDetailProps) {
  return (
    <section className="card stack-sm surface-card">
      <div className="surface-meta">
        <span>{announcement.isPublished ? "Published" : "Draft"}</span>
        <span>{formatDate(announcement.publishedAt)}</span>
      </div>
      <h1>{announcement.title}</h1>
      <p className="announcement-content-wrap">{announcement.content}</p>
    </section>
  );
}
