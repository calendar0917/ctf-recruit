"use client";

import { useParams } from "next/navigation";
import React, { useCallback, useEffect, useRef, useState } from "react";
import {
	ChallengeDetail,
	formatLifecycleError,
} from "@/components/challenge/ChallengeDetail";
import { getChallenge } from "@/lib/api/challenges";
import {
	getMyInstance,
	resolveMyInstanceCooldownUntil,
	startInstance,
	stopInstance,
} from "@/lib/api/instances";
import { listMySubmissionsByChallenge } from "@/lib/api/submissions";
import { HttpError } from "@/lib/http";
import type {
	Challenge,
	ChallengeInstance,
	MyChallengeInstanceResponse,
	SubmissionListResponse,
	SubmissionResponse,
} from "@/lib/types";
import { useRequireAuth } from "@/lib/use-auth";

void React;

type ContextRefreshReason = "initial" | "manual" | "background";

type ChallengeContextPayload = {
	challengeResp: Challenge;
	instanceResp: MyChallengeInstanceResponse;
	submissionResp: SubmissionListResponse;
};

async function loadChallengeContext(
	accessToken: string,
	challengeIdValue: string,
): Promise<ChallengeContextPayload> {
	const [challengeResp, instanceResp, submissionResp] = await Promise.all([
		getChallenge(accessToken, challengeIdValue),
		getMyInstance(accessToken),
		listMySubmissionsByChallenge(accessToken, challengeIdValue, {
			limit: 20,
			offset: 0,
		}),
	]);

	return { challengeResp, instanceResp, submissionResp };
}

function parseTime(value?: string): number | null {
	if (!value) {
		return null;
	}

	const parsed = Date.parse(value);
	return Number.isNaN(parsed) ? null : parsed;
}

function usePollingInterval(
	enabled: boolean,
	intervalMs: number,
	onTick: () => void,
): void {
	useEffect(() => {
		if (!enabled) {
			return;
		}

		const timer = window.setInterval(onTick, intervalMs);

		return () => {
			window.clearInterval(timer);
		};
	}, [enabled, intervalMs, onTick]);
}

export default function ChallengeDetailPage() {
	const params = useParams<{ id: string }>();
	const challengeId = params?.id;

	const { session, ready, authorized } = useRequireAuth();

	const [challenge, setChallenge] = useState<Challenge | null>(null);
	const [instance, setInstance] = useState<ChallengeInstance | null>(null);
	const [cooldownUntil, setCooldownUntil] = useState<string | undefined>();
	const [instanceAction, setInstanceAction] = useState<
		"starting" | "stopping" | null
	>(null);
	const [instanceError, setInstanceError] = useState<string | undefined>();
	const [nowMs, setNowMs] = useState(() => Date.now());
	const [submissionHistory, setSubmissionHistory] = useState<
		SubmissionResponse[]
	>([]);
	const [latestSubmission, setLatestSubmission] =
		useState<SubmissionResponse | null>(null);
	const [loading, setLoading] = useState(false);
	const [refreshState, setRefreshState] = useState<
		"idle" | "manual" | "background"
	>("idle");
	const [error, setError] = useState<string | undefined>();
	const [refreshError, setRefreshError] = useState<string | undefined>();
	const refreshRequestIdRef = useRef(0);
	const mountedRef = useRef(true);

	const accessToken = session?.accessToken;
	const cooldownTs = parseTime(cooldownUntil);
	const cooldownActive = typeof cooldownTs === "number" && cooldownTs > nowMs;
	const instanceChallengeMismatch = Boolean(
		instance && challengeId && instance.challengeId !== challengeId,
	);

	useEffect(() => {
		mountedRef.current = true;

		return () => {
			mountedRef.current = false;
		};
	}, []);

	const applyChallengeContext = useCallback(
		(context: ChallengeContextPayload) => {
			setChallenge(context.challengeResp);
			setInstance(context.instanceResp.instance);
			setCooldownUntil(resolveMyInstanceCooldownUntil(context.instanceResp));
			setSubmissionHistory(context.submissionResp.items);
			setLatestSubmission(context.submissionResp.items[0] ?? null);
		},
		[],
	);

	const refreshChallengeContext = useCallback(
		async (reason: ContextRefreshReason): Promise<void> => {
			if (!accessToken || !challengeId) {
				return;
			}

			const requestId = ++refreshRequestIdRef.current;
			const isInitial = reason === "initial";

			if (isInitial) {
				setLoading(true);
				setError(undefined);
			} else {
				setRefreshError(undefined);
				setRefreshState(reason === "manual" ? "manual" : "background");
			}

			try {
				const context = await loadChallengeContext(accessToken, challengeId);
				if (!mountedRef.current || requestId !== refreshRequestIdRef.current) {
					return;
				}

				applyChallengeContext(context);
			} catch (err) {
				if (!mountedRef.current || requestId !== refreshRequestIdRef.current) {
					return;
				}

				if (isInitial) {
					if (err instanceof HttpError) {
						setError(err.message);
					} else {
						setError("Failed to load challenge.");
					}
					return;
				}

				if (err instanceof HttpError) {
					setRefreshError(err.message);
				} else {
					setRefreshError("Refresh failed.");
				}
			} finally {
				if (mountedRef.current && requestId === refreshRequestIdRef.current) {
					if (isInitial) {
						setLoading(false);
					} else {
						setRefreshState("idle");
					}
				}
			}
		},
		[accessToken, applyChallengeContext, challengeId],
	);

	const reconcileInstanceState = useCallback(async (): Promise<void> => {
		if (!accessToken) {
			return;
		}

		try {
			const current = await getMyInstance(accessToken);
			setInstance(current.instance);
			setCooldownUntil(
				(previous) => resolveMyInstanceCooldownUntil(current) ?? previous,
			);
		} catch (err) {
			if (err instanceof HttpError) {
				setInstanceError((previous) =>
					previous
						? `${previous} Reconciliation check failed: ${err.message} (${err.code ?? `HTTP_${err.status}`}).`
						: `Reconciliation check failed: ${err.message} (${err.code ?? `HTTP_${err.status}`}).`,
				);
				return;
			}

			setInstanceError((previous) =>
				previous
					? `${previous} Reconciliation check failed due to an unexpected error.`
					: "Reconciliation check failed due to an unexpected error.",
			);
		}
	}, [accessToken]);

	const handleManualRefresh = useCallback(async (): Promise<void> => {
		await refreshChallengeContext("manual");
	}, [refreshChallengeContext]);

	useEffect(() => {
		if (!ready || !authorized || !accessToken || !challengeId) {
			return;
		}

		void refreshChallengeContext("initial");
	}, [accessToken, authorized, challengeId, ready, refreshChallengeContext]);

	async function handleStartInstance(): Promise<void> {
		if (!accessToken || !challengeId) {
			return;
		}

		if (instance && instance.challengeId !== challengeId) {
			setInstanceError(
				`Active instance belongs to another challenge (${instance.challengeId}). Open /challenges/${instance.challengeId} to manage it.`,
			);
			return;
		}

		setInstanceAction("starting");
		setInstanceError(undefined);

		try {
			const started = await startInstance(accessToken, { challengeId });
			setInstance(started);
			setCooldownUntil(started.cooldownUntil);
		} catch (err) {
			if (err instanceof HttpError) {
				const formatted = formatLifecycleError(err, "start");
				if (formatted.retryAt) {
					setCooldownUntil(formatted.retryAt);
				}
				setInstanceError(formatted.message);
			} else {
				setInstanceError("Failed to start instance.");
			}

			await reconcileInstanceState();
		} finally {
			setInstanceAction(null);
		}
	}

	async function handleStopInstance(): Promise<void> {
		if (!accessToken || !challengeId) {
			return;
		}

		if (instance && instance.challengeId !== challengeId) {
			setInstanceError(
				`Active instance belongs to another challenge (${instance.challengeId}). Open /challenges/${instance.challengeId} to manage it.`,
			);
			return;
		}

		if (!instance?.id) {
			setInstanceError(
				"Cannot stop instance because the current instance ID is unavailable.",
			);
			return;
		}

		const instanceId = instance.id;

		setInstanceAction("stopping");
		setInstanceError(undefined);

		try {
			const stopped = await stopInstance(accessToken, { instanceId });
			setInstance(stopped);
			setCooldownUntil(stopped.cooldownUntil);
		} catch (err) {
			if (err instanceof HttpError) {
				const formatted = formatLifecycleError(err, "stop");
				if (formatted.retryAt) {
					setCooldownUntil(formatted.retryAt);
				}
				setInstanceError(formatted.message);
			} else {
				setInstanceError("Failed to stop instance.");
			}

			await reconcileInstanceState();
		} finally {
			setInstanceAction(null);
		}
	}

	const pollChallengeContext = useCallback(() => {
		void refreshChallengeContext("background");
	}, [refreshChallengeContext]);

	const tickNow = useCallback(() => {
		setNowMs(Date.now());
	}, []);

	usePollingInterval(
		Boolean(
			accessToken && challengeId && latestSubmission?.status === "pending",
		),
		5000,
		pollChallengeContext,
	);

	usePollingInterval(
		Boolean(
			accessToken &&
				challengeId &&
				(instance?.status === "starting" || instance?.status === "stopping"),
		),
		3000,
		pollChallengeContext,
	);

	usePollingInterval(cooldownActive, 1000, tickNow);

	if (!ready || !authorized || !session) {
		return (
			<main className="page">
				<section className="card">
					<p>Loading session...</p>
				</section>
			</main>
		);
	}

	if (!challengeId) {
		return (
			<main className="page">
				<section className="card">
					<p className="error-text">Challenge ID is missing.</p>
				</section>
			</main>
		);
	}

	if (loading) {
		return (
			<main className="page">
				<section className="card">
					<p>Loading challenge...</p>
				</section>
			</main>
		);
	}

	if (error || !challenge) {
		return (
			<main className="page">
				<section className="card">
					<p className="error-text">{error ?? "Challenge not found."}</p>
				</section>
			</main>
		);
	}

	return (
		<main className="page page-content">
			<ChallengeDetail
				token={session.accessToken}
				challenge={challenge}
				latestSubmission={latestSubmission}
				submissionHistory={submissionHistory}
				instance={instance}
				instanceAction={instanceAction}
				instanceChallengeMismatch={instanceChallengeMismatch}
				cooldownUntil={cooldownUntil}
				instanceError={instanceError}
				nowMs={nowMs}
				refreshing={refreshState === "manual"}
				refreshError={refreshError}
				onSubmitted={(result) => {
					setLatestSubmission(result);
					setSubmissionHistory((prev) => [result, ...prev]);
				}}
				onStartInstance={handleStartInstance}
				onStopInstance={handleStopInstance}
				onManualRefresh={handleManualRefresh}
			/>
		</main>
	);
}
