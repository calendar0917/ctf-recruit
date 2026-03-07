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

vi.mock("@/lib/api/challenges", () => ({
  listChallenges: vi.fn(),
  createChallenge: vi.fn(),
  updateChallenge: vi.fn(),
  deleteChallenge: vi.fn(),
}));

vi.mock("@/components/admin/AdminPrimitives", async () => {
	const React = await vi.importActual<typeof import("react")>("react");
	return {
		AdminSection: ({ children }: { children: React.ReactNode }) =>
			React.createElement("section", { className: "card" }, children),
	};
});

vi.mock("@/components/admin/ChallengeEditor", async () => {
	const React = await vi.importActual<typeof import("react")>("react");
	return {
		ChallengeEditor: () => React.createElement("section", undefined, "challenge-editor-mock"),
	};
});

vi.mock("@/components/admin/ChallengeTable", async () => {
	const React = await vi.importActual<typeof import("react")>("react");
	return {
		ChallengeTable: () => React.createElement("section", undefined, "challenge-table-mock"),
	};
});

describe("admin/challenges guard rendering", () => {
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
		const pageModule = await import("@/app/admin/challenges/page");
		ReactDOMServer.renderToStaticMarkup(React.createElement(pageModule.default));

		expect(useRequireAuthMock).toHaveBeenCalledWith({ adminOnly: true });
		expect(replaceMock).toHaveBeenCalledWith("/login");
	});

	it("renders 403 hint for authenticated non-admin session", async () => {
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
		const pageModule = await import("@/app/admin/challenges/page");
    const html = ReactDOMServer.renderToStaticMarkup(
      React.createElement(pageModule.default),
    );

		expect(useRequireAuthMock).toHaveBeenCalledWith({ adminOnly: true });
		expect(replaceMock).toHaveBeenCalledWith("/challenges");
		expect(html).toContain("403");
		expect(html).toContain("Admin role required.");
	});

	it("does not redirect and renders admin screen for admin session", async () => {
		useRequireAuthMock.mockImplementation((_options?: { adminOnly?: boolean }) => ({
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
		}));

		const React = await import("react");
		const ReactDOMServer = await import("react-dom/server");
		const pageModule = await import("@/app/admin/challenges/page");
		const html = ReactDOMServer.renderToStaticMarkup(
			React.createElement(pageModule.default),
		);

		expect(useRequireAuthMock).toHaveBeenCalledWith({ adminOnly: true });
		expect(replaceMock).not.toHaveBeenCalled();
		expect(html).toContain("Admin: Challenge management");
		expect(html).toContain("challenge-editor-mock");
		expect(html).toContain("challenge-table-mock");
		expect(html).not.toContain("Admin role required.");
	});
});
