import { describe, expect, it } from "vitest";
import {
	type AdminChallengeEditorPayload,
	buildCreateChallengePayload,
	buildUpdateChallengePayload,
	mapChallengeRuntimeFieldErrors,
} from "@/lib/types";

describe("admin challenge runtime payload mapping", () => {
	it("includes runtime fields for create payload when provided", () => {
		const value: AdminChallengeEditorPayload = {
			title: " Runtime Probe ",
			description: " Description ",
			category: " ops ",
			difficulty: "easy",
			mode: "dynamic",
			runtimeImage: " busybox:1.36 ",
			runtimeCommand: " httpd -f -p 8080 ",
			runtimeExposedPort: 8080,
			points: 150,
			flag: " CTF{flag} ",
			isPublished: true,
		};

		expect(buildCreateChallengePayload(value)).toEqual({
			title: "Runtime Probe",
			description: "Description",
			category: "ops",
			difficulty: "easy",
			mode: "dynamic",
			runtimeImage: "busybox:1.36",
			runtimeCommand: "httpd -f -p 8080",
			runtimeExposedPort: 8080,
			points: 150,
			flag: "CTF{flag}",
			isPublished: true,
		});
	});

	it("keeps runtime fields optional for static-compatible create payload", () => {
		const value: AdminChallengeEditorPayload = {
			title: "Static",
			description: "No runtime",
			category: "misc",
			difficulty: "medium",
			mode: "static",
			runtimeImage: "   ",
			runtimeCommand: "",
			runtimeExposedPort: undefined,
			points: 100,
			flag: "CTF{static}",
			isPublished: false,
		};

		expect(buildCreateChallengePayload(value)).toEqual({
			title: "Static",
			description: "No runtime",
			category: "misc",
			difficulty: "medium",
			mode: "static",
			points: 100,
			flag: "CTF{static}",
			isPublished: false,
		});
	});

	it("builds update payload without empty flag while preserving runtime fields", () => {
		const value: AdminChallengeEditorPayload = {
			id: "ch-1",
			title: " Updated ",
			description: " Updated description ",
			category: " web ",
			difficulty: "hard",
			mode: "dynamic",
			runtimeImage: " nginx:latest ",
			runtimeCommand: " nginx -g 'daemon off;' ",
			runtimeExposedPort: 80,
			points: 500,
			flag: "   ",
			isPublished: true,
		};

		expect(buildUpdateChallengePayload(value)).toEqual({
			title: "Updated",
			description: "Updated description",
			category: "web",
			difficulty: "hard",
			mode: "dynamic",
			runtimeImage: "nginx:latest",
			runtimeCommand: "nginx -g 'daemon off;'",
			runtimeExposedPort: 80,
			points: 500,
			isPublished: true,
		});
	});
});

describe("admin challenge runtime validation mapping", () => {
	it("maps runtimeExposedPort backend validation message to field error", () => {
		const message = "runtimeExposedPort must be greater than zero";
		expect(mapChallengeRuntimeFieldErrors(message)).toEqual({
			runtimeExposedPort: message,
		});
	});

	it("returns empty mapping for non-runtime message", () => {
		expect(mapChallengeRuntimeFieldErrors("Failed to save challenge.")).toEqual(
			{},
		);
	});
});
