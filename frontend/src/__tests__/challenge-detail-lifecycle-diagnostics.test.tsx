import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import {
	ChallengeDetail,
	formatLifecycleError,
} from "@/components/challenge/ChallengeDetail";
import { HttpError } from "@/lib/http";
import type {
	ApiErrorResponse,
	Challenge,
	ChallengeInstance,
} from "@/lib/types";

void React;

const challenge: Challenge = {
	id: "11111111-1111-1111-1111-111111111111",
	title: "Dynamic Challenge",
	description: "Judge this challenge asynchronously",
	category: "web",
	difficulty: "medium",
	mode: "dynamic",
	points: 300,
	isPublished: true,
	createdAt: "2026-02-16T10:00:00Z",
	updatedAt: "2026-02-16T10:00:00Z",
};

function createHttpError(options: {
	status: number;
	code?: string;
	message: string;
	details?: Record<string, unknown>;
	retryAt?: string;
}): HttpError {
	const payload: ApiErrorResponse = {
		error: {
			code: options.code,
			message: options.message,
			details: options.details,
		},
	};

	return new HttpError(
		options.message,
		options.status,
		payload,
		options.retryAt
			? {
					retryAt: options.retryAt,
				}
			: undefined,
	);
}

function renderInstance(
	instance: ChallengeInstance | null,
	options?: {
		instanceChallengeMismatch?: boolean;
		cooldownUntil?: string;
		nowMs?: number;
		instanceAction?: "starting" | "stopping" | null;
	},
): string {
	return renderToStaticMarkup(
		<ChallengeDetail
			token="player-token"
			challenge={challenge}
			latestSubmission={null}
			submissionHistory={[]}
			instance={instance}
			instanceAction={options?.instanceAction ?? null}
			instanceChallengeMismatch={options?.instanceChallengeMismatch ?? false}
			cooldownUntil={options?.cooldownUntil}
			nowMs={options?.nowMs ?? Date.parse("2026-02-16T10:00:00Z")}
			refreshing={false}
			onSubmitted={() => undefined}
			onStartInstance={async () => undefined}
			onStopInstance={async () => undefined}
			onManualRefresh={async () => undefined}
		/>,
	);
}

describe("challenge detail lifecycle action-state transparency", () => {
	it("shows deterministic idle state with start enabled and stop disabled reason", () => {
		const html = renderInstance(null);

		expect(html).toContain("Action state: <strong>idle</strong>");
		expect(html).toContain("Start action: <strong>enabled</strong>");
		expect(html).toContain(
			"Stop action: <strong>disabled</strong> — No active instance to stop.",
		);
	});

	it("shows deterministic cooldown state and explicit start disabled reason", () => {
		const cooldownUntil = "2026-02-16T10:01:20Z";
		const html = renderInstance(null, {
			cooldownUntil,
			nowMs: Date.parse("2026-02-16T10:01:00Z"),
		});

		expect(html).toContain("Action state: <strong>cooldown</strong>");
		expect(html).toContain(
			`Start action: <strong>disabled</strong> — Cooldown active until ${cooldownUntil}.`,
		);
		expect(html).toContain("Retry at: 2026-02-16T10:01:20Z");
		expect(html).toContain("Cooldown remaining: 20s");
	});

	it("shows deterministic mismatch state and disables both actions with reason", () => {
		const mismatchChallengeId = "22222222-2222-2222-2222-222222222222";
		const html = renderInstance(
			{
				id: "inst-mismatch",
				userId: "u-1",
				challengeId: mismatchChallengeId,
				status: "running",
			},
			{ instanceChallengeMismatch: true },
		);

		expect(html).toContain("Action state: <strong>mismatch</strong>");
		expect(html).toContain(
			`Start action: <strong>disabled</strong> — Blocked by active instance mismatch (${mismatchChallengeId}).`,
		);
		expect(html).toContain(
			`Stop action: <strong>disabled</strong> — Blocked by active instance mismatch (${mismatchChallengeId}).`,
		);
	});

	it("shows deterministic transition state while starting", () => {
		const html = renderInstance(null, { instanceAction: "starting" });

		expect(html).toContain("Action state: <strong>starting</strong>");
		expect(html).toContain(
			"Start action: <strong>disabled</strong> — Start request is in progress.",
		);
		expect(html).toContain(
			"Stop action: <strong>disabled</strong> — Start request is in progress.",
		);
	});

	it("shows active instance state with deterministic start disabled reason", () => {
		const html = renderInstance({
			id: "inst-running",
			userId: "u-1",
			challengeId: challenge.id,
			status: "running",
		});

		expect(html).toContain("Action state: <strong>running</strong>");
		expect(html).toContain(
			"Start action: <strong>disabled</strong> — Instance already active. Stop it before starting a new one.",
		);
		expect(html).toContain("Stop action: <strong>enabled</strong>");
	});
});

describe("challenge detail lifecycle diagnostics messaging", () => {
	it("keeps backend cooldown message and adds retry guidance", () => {
		const err = createHttpError({
			status: 409,
			code: "INSTANCE_COOLDOWN_ACTIVE",
			message: "Instance is cooling down",
			details: {
				retryAt: "2026-02-17T10:01:00Z",
			},
		});

		const formatted = formatLifecycleError(err, "start");

		expect(formatted.retryAt).toBe("2026-02-17T10:01:00Z");
		expect(formatted.message).toContain("Instance is cooling down");
		expect(formatted.message).toContain("(INSTANCE_COOLDOWN_ACTIVE)");
		expect(formatted.message).toContain("Retry after 2026-02-17T10:01:00Z.");
		expect(formatted.message).toContain(
			"Instance is cooling down. Wait until retry time, then start again.",
		);
	});

	it("adds auth guidance for missing/invalid token errors", () => {
		const err = createHttpError({
			status: 401,
			code: "AUTH_MISSING_TOKEN",
			message: "Authorization header is required",
		});

		const formatted = formatLifecycleError(err, "start");

		expect(formatted.message).toContain("Authorization header is required");
		expect(formatted.message).toContain("(AUTH_MISSING_TOKEN)");
		expect(formatted.message).toContain(
			"Authentication token is missing/invalid. Sign in again and retry.",
		);
	});

	it("adds runtime guidance for runtime start failures", () => {
		const err = createHttpError({
			status: 500,
			code: "INSTANCE_RUNTIME_START_FAILED",
			message: "Runtime unavailable",
		});

		const formatted = formatLifecycleError(err, "start");

		expect(formatted.message).toContain("Runtime unavailable");
		expect(formatted.message).toContain("(INSTANCE_RUNTIME_START_FAILED)");
		expect(formatted.message).toContain(
			"Runtime dependency failed. Verify runtime availability and retry. If this persists, contact admins.",
		);
	});

	it("falls back to retry/manual-refresh guidance while preserving original message", () => {
		const err = createHttpError({
			status: 502,
			message: "Upstream dependency timeout",
		});

		const formatted = formatLifecycleError(err, "stop");

		expect(formatted.message).toContain("Upstream dependency timeout");
		expect(formatted.message).toContain("(HTTP_502)");
		expect(formatted.message).toContain(
			"Stop request failed. Retry, then use manual refresh to reconcile state.",
		);
	});

	it("uses retry metadata when details.retryAt is absent", () => {
		const err = createHttpError({
			status: 409,
			code: "INSTANCE_COOLDOWN_ACTIVE",
			message: "Cooldown active",
			retryAt: "2026-02-17T10:03:00.000Z",
		});

		const formatted = formatLifecycleError(err, "start");

		expect(formatted.retryAt).toBe("2026-02-17T10:03:00.000Z");
		expect(formatted.message).toContain(
			"Retry after 2026-02-17T10:03:00.000Z.",
		);
	});
});
