import { afterEach, describe, expect, it, vi } from "vitest";
import {
	getMyInstance,
	resolveActiveInstanceConflictDetails,
	resolveMyInstanceCooldownUntil,
	startInstance,
	stopInstance,
} from "@/lib/api/instances";
import { HttpError } from "@/lib/http";
import type { ApiErrorResponse } from "@/lib/types";

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

describe("instances api client", () => {
	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it("gets current user instance from /instances/me", async () => {
		const fetchMock = vi.fn(async () =>
			mockJsonResponse({
				body: {
					instance: {
						id: "inst-1",
						userId: "u-1",
						challengeId: "ch-1",
						status: "running",
						expiresAt: "2026-02-16T11:00:00Z",
					},
				},
			}),
		);
		vi.stubGlobal("fetch", fetchMock);

		const response = await getMyInstance("player-token");
		expect(response.instance?.status).toBe("running");

		const call = fetchMock.mock.calls.at(0) as unknown[] | undefined;
		const init = (call?.[1] ?? {}) as RequestInit;
		const headers = init.headers as Headers;

		expect(String(call?.[0])).toContain("/api/v1/instances/me");
		expect(init.method).toBe("GET");
		expect(headers.get("Authorization")).toBe("Bearer player-token");
	});

	it("starts and stops instance with expected payload", async () => {
		const fetchMock = vi
			.fn()
			.mockImplementationOnce(async (_url: string, init: RequestInit) => {
				const payload = JSON.parse(String(init.body)) as {
					challengeId: string;
				};
				return mockJsonResponse({
					status: 201,
					body: {
						id: "inst-2",
						userId: "u-1",
						challengeId: payload.challengeId,
						status: "running",
					},
				});
			})
			.mockImplementationOnce(async (_url: string, init: RequestInit) => {
				const payload = JSON.parse(String(init.body)) as {
					instanceId?: string;
				};
				return mockJsonResponse({
					body: {
						id: payload.instanceId ?? "inst-2",
						userId: "u-1",
						challengeId: "ch-1",
						status: "stopped",
						cooldownUntil: "2026-02-16T10:02:00Z",
					},
				});
			});
		vi.stubGlobal("fetch", fetchMock);

		const started = await startInstance("player-token", {
			challengeId: "ch-1",
		});
		const stopped = await stopInstance("player-token", {
			instanceId: started.id,
		});

		expect(started.status).toBe("running");
		expect(stopped.status).toBe("stopped");
		expect(fetchMock).toHaveBeenCalledTimes(2);

		const startCall = fetchMock.mock.calls.at(0) as unknown[] | undefined;
		expect(String(startCall?.[0])).toContain("/api/v1/instances/start");
		expect(JSON.parse(String((startCall?.[1] as RequestInit)?.body))).toEqual({
			challengeId: "ch-1",
		});

		const stopCall = fetchMock.mock.calls.at(1) as unknown[] | undefined;
		expect(String(stopCall?.[0])).toContain("/api/v1/instances/stop");
		expect(JSON.parse(String((stopCall?.[1] as RequestInit)?.body))).toEqual({
			instanceId: "inst-2",
		});
	});

	it("resolves cooldown from additive me cooldown metadata", () => {
		const response = {
			instance: null,
			cooldown: {
				retryAt: "2026-02-16T10:02:00Z",
			},
		};

		expect(resolveMyInstanceCooldownUntil(response)).toBe(
			"2026-02-16T10:02:00Z",
		);
	});

	it("prefers instance cooldownUntil when active instance exists", () => {
		const response = {
			instance: {
				id: "inst-2",
				userId: "u-1",
				challengeId: "ch-1",
				status: "running" as const,
				cooldownUntil: "2026-02-16T10:03:00Z",
			},
			cooldown: {
				retryAt: "2026-02-16T10:02:00Z",
			},
		};

		expect(resolveMyInstanceCooldownUntil(response)).toBe(
			"2026-02-16T10:03:00Z",
		);
	});

	it("returns undefined when no cooldown metadata exists", () => {
		expect(resolveMyInstanceCooldownUntil({ instance: null })).toBeUndefined();
	});

	it("resolves active-instance conflict details when payload contains additive fields", () => {
		const details = {
			activeInstanceId: "inst-1",
			activeUserId: "u-1",
			activeChallengeId: "ch-1",
			activeStatus: "running",
			activeStartedAt: "2026-02-16T10:00:00Z",
			activeExpiresAt: "2026-02-16T11:00:00Z",
		} as const;

		const payload: ApiErrorResponse = {
			error: {
				code: "INSTANCE_ACTIVE_EXISTS",
				message: "An active instance already exists",
				details,
			},
		};

		const error = new HttpError(
			"An active instance already exists",
			409,
			payload,
		);
		expect(resolveActiveInstanceConflictDetails(error)).toEqual(details);
	});

	it("handles optional active-instance timestamp fields when omitted", () => {
		const details = {
			activeInstanceId: "inst-1",
			activeUserId: "u-1",
			activeChallengeId: "ch-1",
			activeStatus: "starting",
		} as const;

		const payload: ApiErrorResponse = {
			error: {
				code: "INSTANCE_ACTIVE_EXISTS",
				message: "An active instance already exists",
				details,
			},
		};

		const error = new HttpError(
			"An active instance already exists",
			409,
			payload,
		);
		const resolved = resolveActiveInstanceConflictDetails(error);

		expect(resolved).toEqual(details);
		expect(resolved?.activeStartedAt).toBeUndefined();
		expect(resolved?.activeExpiresAt).toBeUndefined();
	});

	it("returns undefined for non-matching code or invalid details shape", () => {
		const notConflict = new HttpError("Cooldown", 409, {
			error: {
				code: "INSTANCE_COOLDOWN_ACTIVE",
				message: "Cooldown",
				details: {
					retryAt: "2026-02-16T10:03:00Z",
				},
			},
		});
		const invalidConflict = new HttpError(
			"An active instance already exists",
			409,
			{
				error: {
					code: "INSTANCE_ACTIVE_EXISTS",
					message: "An active instance already exists",
					details: {
						activeInstanceId: "inst-1",
						activeUserId: "u-1",
						activeStatus: "running",
					},
				},
			},
		);

		expect(resolveActiveInstanceConflictDetails(notConflict)).toBeUndefined();
		expect(
			resolveActiveInstanceConflictDetails(invalidConflict),
		).toBeUndefined();
	});
});
