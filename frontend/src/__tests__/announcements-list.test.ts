import { describe, expect, it } from "vitest";
import React from "react";
import ReactDOMServer from "react-dom/server";
import { AnnouncementList } from "@/components/announcement/AnnouncementList";

describe("AnnouncementList", () => {
  it("renders announcement items", () => {
    const html = ReactDOMServer.renderToStaticMarkup(
      React.createElement(AnnouncementList, {
        items: [
          {
            id: "ann-1",
            title: "Welcome",
            content: "Published message",
            isPublished: true,
            publishedAt: "2026-02-16T08:00:00Z",
            createdAt: "2026-02-16T08:00:00Z",
            updatedAt: "2026-02-16T08:00:00Z",
          },
          {
            id: "ann-2",
            title: "Round 2",
            content: "Starts at 20:00",
            isPublished: true,
            publishedAt: "2026-02-16T09:00:00Z",
            createdAt: "2026-02-16T09:00:00Z",
            updatedAt: "2026-02-16T09:00:00Z",
          },
        ],
      }),
    );

    expect(html).toContain("Welcome");
    expect(html).toContain("Round 2");
    expect(html).toContain("/announcements/ann-1");
    expect(html).toContain("/announcements/ann-2");
  });
});
