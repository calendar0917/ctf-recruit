import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createSubmission,
  listMySubmissions,
  listMySubmissionsByChallenge,
} from "@/lib/api/submissions";

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

describe("submissions api client", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("submits flag payload with auth token", async () => {
    const fetchMock = vi.fn(async () =>
      mockJsonResponse({
        status: 201,
        body: {
          id: "sub-1",
          challengeId: "ch-1",
          status: "pending",
          awardedPoints: 0,
          judgeJobId: "job-1",
          createdAt: "2026-02-16T10:00:00Z",
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const created = await createSubmission("player-token", {
      challengeId: "ch-1",
      flag: "flag{pending}",
    });

    expect(created.id).toBe("sub-1");
    expect(created.status).toBe("pending");

    const call = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    const requestInit = (call?.[1] ?? {}) as RequestInit;
    const headers = requestInit.headers as Headers;
    expect(requestInit.method).toBe("POST");
    expect(headers.get("Authorization")).toBe("Bearer player-token");
  });

  it("lists submission history from /submissions/me", async () => {
    const fetchMock = vi.fn(async () =>
      mockJsonResponse({
        body: {
          items: [
            {
              id: "sub-2",
              challengeId: "ch-2",
              status: "correct",
              awardedPoints: 100,
              createdAt: "2026-02-16T10:10:00Z",
            },
          ],
          limit: 20,
          offset: 0,
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const list = await listMySubmissions("player-token", { limit: 20, offset: 0 });
    expect(list.items).toHaveLength(1);

    const call = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    expect(String(call?.[0])).toContain("/api/v1/submissions/me?limit=20&offset=0");
  });

  it("lists challenge submission history from /submissions/challenge/:id", async () => {
    const fetchMock = vi.fn(async () =>
      mockJsonResponse({
        body: {
          items: [
            {
              id: "sub-3",
              challengeId: "ch-9",
              status: "wrong",
              awardedPoints: 0,
              createdAt: "2026-02-16T10:20:00Z",
            },
          ],
          limit: 10,
          offset: 0,
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const list = await listMySubmissionsByChallenge("player-token", "ch-9", {
      limit: 10,
      offset: 0,
    });
    expect(list.items[0]?.challengeId).toBe("ch-9");

    const call = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    expect(String(call?.[0])).toContain("/api/v1/submissions/challenge/ch-9?limit=10&offset=0");
  });
});
