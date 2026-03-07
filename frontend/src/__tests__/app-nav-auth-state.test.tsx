import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";

void React;

const useAuthSessionMock = vi.fn();
const usePathnameMock = vi.fn(() => "/");

vi.mock("next/navigation", () => ({
	usePathname: () => usePathnameMock(),
	useRouter: () => ({
		push: vi.fn(),
	}),
}));

vi.mock("@/lib/use-auth", () => ({
	useAuthSession: () => useAuthSessionMock(),
}));

describe("AppNav auth-state rendering", () => {
	beforeEach(() => {
		useAuthSessionMock.mockReset();
		usePathnameMock.mockReset();
		usePathnameMock.mockReturnValue("/");
	});

	it("renders deterministic visible loading nav while auth session initializes", async () => {
		useAuthSessionMock.mockReturnValue({
			ready: false,
			session: null,
		});

		const navModule = await import("@/components/layout/AppNav");
		const html = renderToStaticMarkup(React.createElement(navModule.AppNav));

		expect(html).toContain("Loading navigation…");
		expect(html).toContain("Checking session…");
	});

	it("renders guest nav links and explicit expired-session hint", async () => {
		useAuthSessionMock.mockReturnValue({
			ready: true,
			session: null,
		});

		const navModule = await import("@/components/layout/AppNav");
		const html = renderToStaticMarkup(React.createElement(navModule.AppNav));

		expect(html).toContain("Login");
		expect(html).toContain("Register");
		expect(html).toContain("Session missing or expired.");
		expect(html).not.toContain("Challenges");
	});

	it("renders authenticated player navigation", async () => {
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

		const navModule = await import("@/components/layout/AppNav");
		const html = renderToStaticMarkup(React.createElement(navModule.AppNav));

		expect(html).toContain("Challenges");
		expect(html).toContain("Announcements");
		expect(html).toContain("Recruitment");
		expect(html).toContain("Scoreboard");
		expect(html).toContain("Logout");
		expect(html).not.toContain("/admin/users");
	});

	it("renders admin navigation entry for admin session", async () => {
		useAuthSessionMock.mockReturnValue({
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

		const navModule = await import("@/components/layout/AppNav");
		const html = renderToStaticMarkup(React.createElement(navModule.AppNav));

		expect(html).toContain("/admin/users");
		expect(html).toContain(">Admin<");
	});
});
