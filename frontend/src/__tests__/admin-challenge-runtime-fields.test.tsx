import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { ChallengeEditor } from "@/components/admin/ChallengeEditor";
import type { Challenge } from "@/lib/types";

void React;

vi.mock("@/components/admin/AdminPrimitives", () => ({
	AdminEditorShell: ({
		title,
		error,
		children,
	}: {
		title: React.ReactNode;
		error?: string;
		children: React.ReactNode;
	}) =>
		React.createElement(
			"section",
			undefined,
			React.createElement("h2", undefined, title),
			error
				? React.createElement("p", { className: "error-text" }, error)
				: null,
			children,
		),
	AdminActionGroup: ({ children }: { children: React.ReactNode }) =>
		React.createElement("div", { className: "row-actions" }, children),
}));

const baseChallenge: Challenge = {
	id: "11111111-1111-1111-1111-111111111111",
	title: "Runtime Challenge",
	description: "challenge with runtime",
	category: "ops",
	difficulty: "easy",
	mode: "dynamic",
	runtimeImage: "busybox:1.36",
	runtimeCommand: "httpd -f -p 8080",
	runtimeExposedPort: 8080,
	points: 100,
	isPublished: false,
	createdAt: "2026-02-17T00:00:00Z",
	updatedAt: "2026-02-17T00:00:00Z",
};

function renderEditor(options: {
	initial?: Challenge | null;
	runtimePortError?: string;
}): string {
	return renderToStaticMarkup(
		<ChallengeEditor
			initial={options.initial}
			loading={false}
			error={undefined}
			fieldErrors={
				options.runtimePortError
					? {
							runtimeExposedPort: options.runtimePortError,
						}
					: undefined
			}
			onSubmit={async () => undefined}
			onCancelEdit={() => undefined}
		/>,
	);
}

describe("admin challenge editor runtime fields", () => {
	it("shows runtime field labels in create mode", () => {
		const html = renderEditor({ initial: null });

		expect(html).toContain("Runtime image (optional)");
		expect(html).toContain("Runtime command (optional)");
		expect(html).toContain("Runtime exposed port (optional)");
	});

	it("prefills runtime values in edit mode", () => {
		const html = renderEditor({ initial: baseChallenge });

		expect(html).toContain('value="busybox:1.36"');
		expect(html).toContain('value="httpd -f -p 8080"');
		expect(html).toContain('value="8080"');
	});

	it("surfaces runtimeExposedPort backend validation message", () => {
		const message = "runtimeExposedPort must be greater than zero";
		const html = renderEditor({ initial: null, runtimePortError: message });

		expect(html).toContain(message);
	});
});
