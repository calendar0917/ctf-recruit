import { afterEach, describe, expect, it, vi } from "vitest";
import { listAdminUsers, updateAdminUser } from "@/lib/api/admin-users";

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

describe("admin users api client", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("lists admin users with auth token", async () => {
    const fetchMock = vi.fn(async () =>
      mockJsonResponse({
        body: {
          items: [
            {
              id: "u-1",
              email: "player@example.com",
              displayName: "Player",
              role: "player",
              isDisabled: false,
            },
          ],
        },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const resp = await listAdminUsers("admin-token", { limit: 20, offset: 0 });
    expect(resp.items).toHaveLength(1);

    const call = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    const init = (call?.[1] ?? {}) as RequestInit;
    const headers = init.headers as Headers;
    expect(headers.get("Authorization")).toBe("Bearer admin-token");
  });

  it("patches role and disabled status", async () => {
    const fetchMock = vi.fn(async (_url: string, init: RequestInit) => {
      const payload = JSON.parse(String(init.body)) as { role?: string; isDisabled?: boolean };
      return mockJsonResponse({
        body: {
          id: "u-2",
          email: "user2@example.com",
          displayName: "User2",
          role: payload.role ?? "player",
          isDisabled: payload.isDisabled ?? false,
        },
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    const updated = await updateAdminUser("admin-token", "u-2", { role: "admin", isDisabled: true });
    expect(updated.role).toBe("admin");
    expect(updated.isDisabled).toBe(true);

    const call = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    expect(String(call?.[0])).toContain("/api/v1/admin/users/u-2");
    const init = (call?.[1] ?? {}) as RequestInit;
    const headers = init.headers as Headers;
    expect(init.method).toBe("PATCH");
    expect(headers.get("Authorization")).toBe("Bearer admin-token");
    expect(init.body).toBe(JSON.stringify({ role: "admin", isDisabled: true }));
  });

  it("patches role-only toggle contract", async () => {
    const fetchMock = vi.fn(async (_url: string, init: RequestInit) => {
      const payload = JSON.parse(String(init.body)) as { role?: string; isDisabled?: boolean };
      return mockJsonResponse({
        body: {
          id: "u-role",
          email: "role@example.com",
          displayName: "Role",
          role: payload.role ?? "player",
          isDisabled: payload.isDisabled ?? false,
        },
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    const updated = await updateAdminUser("admin-token", "u-role", { role: "player" });
    expect(updated.role).toBe("player");
    expect(updated.isDisabled).toBe(false);

    const call = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    const init = (call?.[1] ?? {}) as RequestInit;
    expect(init.body).toBe(JSON.stringify({ role: "player" }));
  });

  it("patches disabled-only toggle contract", async () => {
    const fetchMock = vi.fn(async (_url: string, init: RequestInit) => {
      const payload = JSON.parse(String(init.body)) as { role?: string; isDisabled?: boolean };
      return mockJsonResponse({
        body: {
          id: "u-disabled",
          email: "disabled@example.com",
          displayName: "Disabled",
          role: payload.role ?? "player",
          isDisabled: payload.isDisabled ?? false,
        },
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    const updated = await updateAdminUser("admin-token", "u-disabled", { isDisabled: true });
    expect(updated.role).toBe("player");
    expect(updated.isDisabled).toBe(true);

    const call = fetchMock.mock.calls.at(0) as unknown[] | undefined;
    const init = (call?.[1] ?? {}) as RequestInit;
    expect(init.body).toBe(JSON.stringify({ isDisabled: true }));
  });
});
