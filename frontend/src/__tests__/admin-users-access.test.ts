import { beforeEach, describe, expect, it, vi } from "vitest";

const replaceMock = vi.fn();
const useRequireAuthMock = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    replace: replaceMock,
  }),
}));

vi.mock("@/lib/use-auth", () => ({
  useRequireAuth: (...args: unknown[]) => useRequireAuthMock(...args),
}));

vi.mock("@/lib/api/admin-users", () => ({
  listAdminUsers: vi.fn(),
  updateAdminUser: vi.fn(),
}));

vi.mock("@/components/admin/AdminPrimitives", async () => {
  const React = await vi.importActual<typeof import("react")>("react");
  return {
    AdminSection: ({ children }: { children: React.ReactNode }) =>
      React.createElement("section", { className: "card" }, children),
    AdminDataTable: ({ children }: { children: React.ReactNode }) =>
      React.createElement("div", { className: "table-wrap" }, children),
    AdminActionGroup: ({ children }: { children: React.ReactNode }) =>
      React.createElement("div", { className: "stack-sm" }, children),
  };
});

vi.mock("@/components/ui/StateFeedback", async () => {
  const React = await vi.importActual<typeof import("react")>("react");
  return {
    LoadingStateCard: ({ message }: { message: string }) =>
      React.createElement("section", { className: "card" }, message),
    ErrorStateCard: ({ message }: { message: string }) =>
      React.createElement("section", { className: "card" }, message),
  };
});

describe("admin/users guard rendering", () => {
  beforeEach(() => {
    replaceMock.mockReset();
    useRequireAuthMock.mockReset();
  });

  it("requests adminOnly guard and redirects to login when session is missing", async () => {
    useRequireAuthMock.mockImplementation((options?: { adminOnly?: boolean }) => {
      if (options?.adminOnly) {
        replaceMock("/login");
      }
      return {
        ready: true,
        session: null,
      };
    });

    const React = await import("react");
    const ReactDOMServer = await import("react-dom/server");
    const pageModule = await import("@/app/admin/users/page");
    ReactDOMServer.renderToStaticMarkup(React.createElement(pageModule.default));

    expect(useRequireAuthMock).toHaveBeenCalledWith({ adminOnly: true });
    expect(replaceMock).toHaveBeenCalledWith("/login");
  });

  it("renders 403 hint and challenges redirect for authenticated non-admin", async () => {
    useRequireAuthMock.mockImplementation((options?: { adminOnly?: boolean }) => {
      if (options?.adminOnly) {
        replaceMock("/challenges");
      }
      return {
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
      };
    });

    const React = await import("react");
    const ReactDOMServer = await import("react-dom/server");
    const pageModule = await import("@/app/admin/users/page");
    const html = ReactDOMServer.renderToStaticMarkup(
      React.createElement(pageModule.default),
    );

    expect(useRequireAuthMock).toHaveBeenCalledWith({ adminOnly: true });
    expect(replaceMock).toHaveBeenCalledWith("/challenges");
    expect(html).toContain("403");
    expect(html).toContain("Admin role required.");
  });

  it("does not redirect and renders admin user page for admin session", async () => {
    useRequireAuthMock.mockReturnValue({
      ready: true,
      session: {
        accessToken: "admin-token",
        tokenType: "Bearer",
        user: {
          id: "u-admin",
          email: "admin@example.com",
          displayName: "Admin",
          role: "admin",
        },
      },
    });

    const React = await import("react");
    const ReactDOMServer = await import("react-dom/server");
    const pageModule = await import("@/app/admin/users/page");
    const html = ReactDOMServer.renderToStaticMarkup(
      React.createElement(pageModule.default),
    );

    expect(useRequireAuthMock).toHaveBeenCalledWith({ adminOnly: true });
    expect(replaceMock).not.toHaveBeenCalled();
    expect(html).toContain("Admin: User management");
    expect(html).not.toContain("Admin role required.");
  });
});
