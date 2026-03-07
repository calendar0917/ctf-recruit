import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createAnnouncement,
  listAnnouncements,
  updateAnnouncement,
} from "@/lib/api/announcements";

type MockResponseInit = {
  status?: number;
  body?: unknown;
};

function mockJsonResponse({ status = 200, body }: MockResponseInit): Response {
  return new Response(body === undefined ? undefined : JSON.stringify(body), {
    status,
    headers: {
      "Content-Type": "application/json",
    },
  });
}

describe("announcements api client", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("uses auth token on list and returns payload", async () => {
    const fetchMock = vi.fn(async () =>
      mockJsonResponse({
        body: {
          items: [
            {
              id: "ann-1",
              title: "Visible",
              content: "published",
              isPublished: true,
              createdAt: "2026-02-16T10:00:00Z",
              updatedAt: "2026-02-16T10:00:00Z",
            },
          ],
          limit: 20,
          offset: 0,
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const response = await listAnnouncements("player-token", { limit: 20, offset: 0 });

    expect(response.items).toHaveLength(1);
    expect(response.items[0]?.id).toBe("ann-1");
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const firstCall = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    const requestInit = (firstCall?.[1] ?? {}) as RequestInit;
    const headers = requestInit.headers as Headers;
    expect(headers.get("Authorization")).toBe("Bearer player-token");
  });

  it("supports admin create/update publish status flow", async () => {
    const fetchMock = vi
      .fn()
      .mockImplementationOnce(async (_url: string, init: RequestInit) => {
        const payload = JSON.parse(String(init.body)) as { isPublished: boolean; title: string };
        return mockJsonResponse({
          status: 201,
          body: {
            id: "ann-admin",
            title: payload.title,
            content: "notice",
            isPublished: payload.isPublished,
            createdAt: "2026-02-16T10:00:00Z",
            updatedAt: "2026-02-16T10:00:00Z",
          },
        });
      })
      .mockImplementationOnce(async (_url: string, init: RequestInit) => {
        const payload = JSON.parse(String(init.body)) as { isPublished: boolean };
        return mockJsonResponse({
          body: {
            id: "ann-admin",
            title: "Admin notice",
            content: "notice",
            isPublished: payload.isPublished,
            createdAt: "2026-02-16T10:00:00Z",
            updatedAt: "2026-02-16T11:00:00Z",
          },
        });
      });

    vi.stubGlobal("fetch", fetchMock);

    const created = await createAnnouncement("admin-token", {
      title: "Admin notice",
      content: "notice",
      isPublished: true,
    });

    const updated = await updateAnnouncement("admin-token", created.id, {
      isPublished: false,
    });

    expect(created.isPublished).toBe(true);
    expect(updated.isPublished).toBe(false);
    expect(fetchMock).toHaveBeenCalledTimes(2);

    const createCall = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    const createInit = (createCall?.[1] ?? {}) as RequestInit;
    const createHeaders = createInit.headers as Headers;
    expect(createInit.method).toBe("POST");
    expect(createHeaders.get("Authorization")).toBe("Bearer admin-token");
    expect(JSON.parse(String(createInit.body))).toMatchObject({
      title: "Admin notice",
      isPublished: true,
    });

    const updateCall = fetchMock.mock.calls.at(1) as unknown[] | undefined;
    const updateInit = (updateCall?.[1] ?? {}) as RequestInit;
    const updateHeaders = updateInit.headers as Headers;
    expect(updateInit.method).toBe("PUT");
    expect(updateHeaders.get("Authorization")).toBe("Bearer admin-token");
    expect(JSON.parse(String(updateInit.body))).toMatchObject({
      isPublished: false,
    });
  });
});
