import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createRecruitmentSubmission,
  getRecruitmentSubmission,
  listRecruitmentSubmissions,
} from "@/lib/api/recruitments";

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

describe("recruitments api client", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("submits recruitment payload with auth token", async () => {
    const fetchMock = vi.fn(async (_url: string, init: RequestInit) => {
      const payload = JSON.parse(String(init.body)) as {
        name: string;
        school: string;
        grade: string;
      };
      return mockJsonResponse({
        status: 201,
        body: {
          id: "rec-1",
          userId: "u-1",
          name: payload.name,
          school: payload.school,
          grade: payload.grade,
          direction: "web",
          contact: "alice@example.com",
          bio: "intro",
          createdAt: "2026-02-16T10:00:00Z",
          updatedAt: "2026-02-16T10:00:00Z",
        },
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    const created = await createRecruitmentSubmission("player-token", {
      name: "Alice",
      school: "Test University",
      grade: "大二",
      direction: "web",
      contact: "alice@example.com",
      bio: "intro",
    });

    expect(created.id).toBe("rec-1");
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const call = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    const requestInit = (call?.[1] ?? {}) as RequestInit;
    const headers = requestInit.headers as Headers;
    expect(requestInit.method).toBe("POST");
    expect(headers.get("Authorization")).toBe("Bearer player-token");
  });

  it("lists and gets detail for admin flow", async () => {
    const fetchMock = vi
      .fn()
      .mockImplementationOnce(async () =>
        mockJsonResponse({
          body: {
            items: [
              {
                id: "rec-1",
                userId: "u-1",
                name: "Alice",
                school: "Test University",
                grade: "大二",
                direction: "web",
                contact: "alice@example.com",
                bio: "intro",
                createdAt: "2026-02-16T10:00:00Z",
                updatedAt: "2026-02-16T10:00:00Z",
              },
            ],
            limit: 20,
            offset: 0,
          },
        }),
      )
      .mockImplementationOnce(async () =>
        mockJsonResponse({
          body: {
            id: "rec-1",
            userId: "u-1",
            name: "Alice",
            school: "Test University",
            grade: "大二",
            direction: "web",
            contact: "alice@example.com",
            bio: "intro",
            createdAt: "2026-02-16T10:00:00Z",
            updatedAt: "2026-02-16T10:00:00Z",
          },
        }),
      );

    vi.stubGlobal("fetch", fetchMock);

    const list = await listRecruitmentSubmissions("admin-token", { limit: 20, offset: 0 });
    const detail = await getRecruitmentSubmission("admin-token", "rec-1");

    expect(list.items).toHaveLength(1);
    expect(detail.id).toBe("rec-1");
    expect(fetchMock).toHaveBeenCalledTimes(2);

    const listCall = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    const listInit = (listCall?.[1] ?? {}) as RequestInit;
    const listHeaders = listInit.headers as Headers;
    expect(listHeaders.get("Authorization")).toBe("Bearer admin-token");

    const detailCall = fetchMock.mock.calls.at(1) as unknown[] | undefined;
    const detailInit = (detailCall?.[1] ?? {}) as RequestInit;
    const detailHeaders = detailInit.headers as Headers;
    expect(detailHeaders.get("Authorization")).toBe("Bearer admin-token");
  });
});
