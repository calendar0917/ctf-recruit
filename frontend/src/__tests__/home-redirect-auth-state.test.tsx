import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

const replaceMock = vi.fn();
const useAuthSessionMock = vi.fn();

vi.mock("react", async () => {
  const actual = await vi.importActual<typeof import("react")>("react");

  return {
    ...actual,
    useEffect: (effect: () => unknown, _deps: unknown[]) => {
      effect();
    },
  };
});

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    replace: replaceMock,
  }),
}));

vi.mock("@/lib/use-auth", () => ({
  useAuthSession: () => useAuthSessionMock(),
}));

describe("home page redirect states", () => {
  beforeEach(() => {
    replaceMock.mockReset();
    useAuthSessionMock.mockReset();
  });

  it("shows deterministic pre-ready message and does not redirect yet", async () => {
    useAuthSessionMock.mockReturnValue({
      ready: false,
      session: null,
    });

    const pageModule = await import("@/app/page");
    const html = renderToStaticMarkup(createElement(pageModule.default));

    expect(html).toContain("Checking local session before redirect…");
    expect(replaceMock).not.toHaveBeenCalled();
  });

  it("redirects authenticated users to challenges with explicit message", async () => {
    useAuthSessionMock.mockReturnValue({
      ready: true,
      session: {
        accessToken: "player-token",
        tokenType: "Bearer",
        user: {
          id: "u-player",
          email: "player@example.com",
          displayName: "Player",
          role: "player",
        },
      },
    });

    const pageModule = await import("@/app/page");
    const html = renderToStaticMarkup(createElement(pageModule.default));

    expect(replaceMock).toHaveBeenCalledWith("/challenges");
    expect(html).toContain("Session resolved. Redirecting to challenges…");
  });

  it("redirects guests to login with explicit message", async () => {
    useAuthSessionMock.mockReturnValue({
      ready: true,
      session: null,
    });

    const pageModule = await import("@/app/page");
    const html = renderToStaticMarkup(createElement(pageModule.default));

    expect(replaceMock).toHaveBeenCalledWith("/login");
    expect(html).toContain("Session resolved. Redirecting to login…");
  });
});
