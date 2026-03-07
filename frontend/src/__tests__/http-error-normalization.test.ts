import { afterEach, describe, expect, it, vi } from "vitest";
import { HttpError, httpRequest } from "@/lib/http";
import {
	type InstanceActiveExistsDetails,
	isInstanceActiveExistsDetails,
} from "@/lib/types";

describe("http error normalization", () => {
	afterEach(() => {
		vi.unstubAllGlobals();
		vi.useRealTimers();
	});

	it("normalizes retry metadata from details.retryAt on 409 cooldown", async () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date("2026-02-17T10:00:00Z"));

		const retryAt = "2026-02-17T10:01:00Z";
		const fetchMock = vi.fn(
			async () =>
				new Response(
					JSON.stringify({
						error: {
							code: "INSTANCE_COOLDOWN_ACTIVE",
							message: "Instance is cooling down",
							details: {
								retryAt,
							},
						},
						requestId: "req-123",
					}),
					{
						status: 409,
						headers: {
							"Content-Type": "application/json",
						},
					},
				),
		);
		vi.stubGlobal("fetch", fetchMock);

		const call = httpRequest("/api/v1/instances/start", {
			method: "POST",
			token: "player-token",
			body: { challengeId: "ch-1" },
		});

		await expect(call).rejects.toBeInstanceOf(HttpError);

		try {
			await call;
		} catch (err) {
			expect(err).toBeInstanceOf(HttpError);
			const httpErr = err as HttpError;
			expect(httpErr.status).toBe(409);
			expect(httpErr.code).toBe("INSTANCE_COOLDOWN_ACTIVE");
			expect(httpErr.details).toEqual({ retryAt });
			expect(httpErr.requestId).toBe("req-123");
			expect(httpErr.retry).toEqual({
				retryAt: "2026-02-17T10:01:00.000Z",
				retryAtMs: Date.parse(retryAt),
				retryAfterSeconds: 60,
				source: "details.retryAt",
			});
		}
	});

	it("parses Retry-After seconds header when details.retryAt is absent", async () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date("2026-02-17T10:00:00Z"));

		const fetchMock = vi.fn(
			async () =>
				new Response(
					JSON.stringify({
						error: {
							code: "TOO_MANY_REQUESTS",
							message: "Slow down",
						},
					}),
					{
						status: 429,
						headers: {
							"Content-Type": "application/json",
							"Retry-After": "120",
						},
					},
				),
		);
		vi.stubGlobal("fetch", fetchMock);

		const call = httpRequest("/api/v1/submissions", {
			method: "POST",
			body: { challengeId: "ch-1", flag: "flag{a}" },
		});

		await expect(call).rejects.toBeInstanceOf(HttpError);

		try {
			await call;
		} catch (err) {
			const httpErr = err as HttpError;
			expect(httpErr.retry).toEqual({
				retryAfterSeconds: 120,
				retryAtMs: Date.parse("2026-02-17T10:02:00.000Z"),
				retryAt: "2026-02-17T10:02:00.000Z",
				source: "header.retry-after-seconds",
				headerValue: "120",
			});
		}
	});

	it("parses Retry-After HTTP-date header", async () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date("2026-02-17T10:00:00Z"));

		const retryAfterDate = "Tue, 17 Feb 2026 10:05:00 GMT";
		const fetchMock = vi.fn(
			async () =>
				new Response(
					JSON.stringify({
						error: {
							code: "RATE_LIMITED",
							message: "Try later",
						},
					}),
					{
						status: 429,
						headers: {
							"Content-Type": "application/json",
							"Retry-After": retryAfterDate,
						},
					},
				),
		);
		vi.stubGlobal("fetch", fetchMock);

		const call = httpRequest("/api/v1/challenges");

		await expect(call).rejects.toBeInstanceOf(HttpError);

		try {
			await call;
		} catch (err) {
			const httpErr = err as HttpError;
			expect(httpErr.retry).toEqual({
				retryAt: "2026-02-17T10:05:00.000Z",
				retryAtMs: Date.parse("2026-02-17T10:05:00.000Z"),
				retryAfterSeconds: 300,
				source: "header.retry-after-date",
				headerValue: retryAfterDate,
			});
		}
	});

	it("handles invalid Retry-After and non-JSON body safely", async () => {
		const fetchMock = vi.fn(
			async () =>
				new Response("Gateway timeout", {
					status: 504,
					headers: {
						"Content-Type": "text/plain",
						"Retry-After": "not-a-valid-value",
					},
				}),
		);
		vi.stubGlobal("fetch", fetchMock);

		const call = httpRequest("/api/v1/instances/me", {
			method: "GET",
			token: "player-token",
		});

		await expect(call).rejects.toBeInstanceOf(HttpError);

		try {
			await call;
		} catch (err) {
			const httpErr = err as HttpError;
			expect(httpErr.status).toBe(504);
			expect(httpErr.message).toBe("HTTP 504");
			expect(httpErr.code).toBeUndefined();
			expect(httpErr.details).toBeUndefined();
			expect(httpErr.requestId).toBeUndefined();
			expect(httpErr.retry).toBeUndefined();
		}
	});

	it("preserves additive active-instance conflict details from error payload", async () => {
		const fetchMock = vi.fn(
			async () =>
				new Response(
					JSON.stringify({
						error: {
							code: "INSTANCE_ACTIVE_EXISTS",
							message: "An active instance already exists",
							details: {
								activeInstanceId: "inst-1",
								activeUserId: "u-1",
								activeChallengeId: "ch-1",
								activeStatus: "running",
								activeStartedAt: "2026-02-17T10:00:00Z",
								activeExpiresAt: "2026-02-17T11:00:00Z",
							},
						},
					}),
					{
						status: 409,
						headers: {
							"Content-Type": "application/json",
						},
					},
				),
		);
		vi.stubGlobal("fetch", fetchMock);

		const call = httpRequest("/api/v1/instances/start", {
			method: "POST",
			token: "player-token",
			body: { challengeId: "ch-2" },
		});

		await expect(call).rejects.toBeInstanceOf(HttpError);

		try {
			await call;
		} catch (err) {
			const httpErr = err as HttpError;
			expect(httpErr.code).toBe("INSTANCE_ACTIVE_EXISTS");
			expect(httpErr.details).toBeDefined();
			expect(isInstanceActiveExistsDetails(httpErr.details)).toBe(true);

			if (isInstanceActiveExistsDetails(httpErr.details)) {
				const details: InstanceActiveExistsDetails = httpErr.details;
				expect(details.activeInstanceId).toBe("inst-1");
				expect(details.activeUserId).toBe("u-1");
				expect(details.activeChallengeId).toBe("ch-1");
				expect(details.activeStatus).toBe("running");
				expect(details.activeStartedAt).toBe("2026-02-17T10:00:00Z");
				expect(details.activeExpiresAt).toBe("2026-02-17T11:00:00Z");
			}
		}
	});

	it("keeps active-instance optional timestamp fields safely optional", async () => {
		const fetchMock = vi.fn(
			async () =>
				new Response(
					JSON.stringify({
						error: {
							code: "INSTANCE_ACTIVE_EXISTS",
							message: "An active instance already exists",
							details: {
								activeInstanceId: "inst-1",
								activeUserId: "u-1",
								activeChallengeId: "ch-1",
								activeStatus: "starting",
							},
						},
					}),
					{
						status: 409,
						headers: {
							"Content-Type": "application/json",
						},
					},
				),
		);
		vi.stubGlobal("fetch", fetchMock);

		const call = httpRequest("/api/v1/instances/start", {
			method: "POST",
			token: "player-token",
			body: { challengeId: "ch-2" },
		});

		await expect(call).rejects.toBeInstanceOf(HttpError);

		try {
			await call;
		} catch (err) {
			const httpErr = err as HttpError;
			expect(httpErr.code).toBe("INSTANCE_ACTIVE_EXISTS");
			expect(isInstanceActiveExistsDetails(httpErr.details)).toBe(true);

			if (isInstanceActiveExistsDetails(httpErr.details)) {
				expect(httpErr.details.activeStartedAt).toBeUndefined();
				expect(httpErr.details.activeExpiresAt).toBeUndefined();
			}
		}
	});
});
