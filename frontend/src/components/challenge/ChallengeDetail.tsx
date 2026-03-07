"use client";

import React from "react";
import { SubmissionForm } from "@/components/submission/SubmissionForm";
import type {
	Challenge,
	ChallengeInstance,
	SubmissionResponse,
} from "@/lib/types";

void React;

type ChallengeDetailProps = {
	token: string;
	challenge: Challenge;
	latestSubmission: SubmissionResponse | null;
	submissionHistory: SubmissionResponse[];
	instance: ChallengeInstance | null;
	instanceAction: "starting" | "stopping" | null;
	instanceChallengeMismatch: boolean;
	cooldownUntil?: string;
	instanceError?: string;
	nowMs: number;
	refreshing: boolean;
	refreshError?: string;
	onSubmitted: (result: SubmissionResponse) => void;
	onStartInstance: () => Promise<void>;
	onStopInstance: () => Promise<void>;
	onManualRefresh: () => Promise<void>;
};

function parseTime(value?: string): number | null {
	if (!value) {
		return null;
	}
	const parsed = Date.parse(value);
	return Number.isNaN(parsed) ? null : parsed;
}

type LifecycleOperation = "start" | "stop";

type LifecycleHttpErrorLike = {
	message: string;
	status: number;
	code?: string;
	details?: unknown;
	retry?: {
		retryAt?: string;
	};
};

export function formatLifecycleError(
	err: LifecycleHttpErrorLike,
	operation: LifecycleOperation,
): { message: string; retryAt?: string } {
	const details =
		err.details && typeof err.details === "object"
			? (err.details as Record<string, unknown>)
			: undefined;
	const retryAt =
		typeof details?.retryAt === "string" ? details.retryAt : err.retry?.retryAt;

	const guidance: string[] = [];
	if (retryAt) {
		guidance.push(`Retry after ${retryAt}.`);
	}

	if (err.code === "INSTANCE_COOLDOWN_ACTIVE") {
		guidance.push(
			"Instance is cooling down. Wait until retry time, then start again.",
		);
	}

	if (
		err.code === "AUTH_MISSING_TOKEN" ||
		err.code === "AUTH_INVALID_TOKEN" ||
		err.status === 401
	) {
		guidance.push(
			"Authentication token is missing/invalid. Sign in again and retry.",
		);
	}

	if (
		err.code === "INSTANCE_RUNTIME_START_FAILED" ||
		err.code === "INSTANCE_RUNTIME_STOP_FAILED"
	) {
		guidance.push(
			"Runtime dependency failed. Verify runtime availability and retry. If this persists, contact admins.",
		);
	}

	if (!guidance.length) {
		guidance.push(
			operation === "start"
				? "Start request failed. Retry, then use manual refresh to reconcile state."
				: "Stop request failed. Retry, then use manual refresh to reconcile state.",
		);
	}

	const guidanceText = guidance.join(" ");
	const message = `${err.message} (${err.code ?? `HTTP_${err.status}`}). ${guidanceText}`;

	return {
		message,
		retryAt,
	};
}

export function ChallengeDetail({
	token,
	challenge,
	latestSubmission,
	submissionHistory,
	instance,
	instanceAction,
	instanceChallengeMismatch,
	cooldownUntil,
	instanceError,
	nowMs,
	refreshing,
	refreshError,
	onSubmitted,
	onStartInstance,
	onStopInstance,
	onManualRefresh,
}: ChallengeDetailProps) {
	const instanceRunning =
		instance?.status === "starting" ||
		instance?.status === "running" ||
		instance?.status === "stopping";
	const cooldownTs = parseTime(cooldownUntil);
	const cooldownRemainingSeconds =
		cooldownTs && cooldownTs > nowMs
			? Math.ceil((cooldownTs - nowMs) / 1000)
			: 0;
	const cooldownActive = cooldownRemainingSeconds > 0;
	const startDisabled =
		instanceAction !== null ||
		instanceRunning ||
		cooldownActive ||
		instanceChallengeMismatch;
	const stopDisabled =
		instanceAction !== null || instanceChallengeMismatch || !instanceRunning;
	const lifecycleActionState =
		instanceAction ??
		(instanceChallengeMismatch
			? "mismatch"
			: cooldownActive
				? "cooldown"
				: (instance?.status ?? "idle"));

	const startDisabledReason = instanceAction
		? instanceAction === "starting"
			? "Start request is in progress."
			: "Stop request is in progress."
		: instanceChallengeMismatch && instance?.challengeId
			? `Blocked by active instance mismatch (${instance.challengeId}).`
			: instanceRunning
				? "Instance already active. Stop it before starting a new one."
				: cooldownActive
					? `Cooldown active until ${cooldownUntil}.`
					: undefined;

	const stopDisabledReason = instanceAction
		? instanceAction === "starting"
			? "Start request is in progress."
			: "Stop request is in progress."
		: instanceChallengeMismatch && instance?.challengeId
			? `Blocked by active instance mismatch (${instance.challengeId}).`
			: !instanceRunning
				? "No active instance to stop."
				: undefined;

	return (
		<div className="stack-lg">
			<section className="card">
				<div className="challenge-meta">
					<span>{challenge.category}</span>
					<span>{challenge.difficulty}</span>
					<span>{challenge.mode}</span>
					<span>{challenge.points} pts</span>
				</div>
				<h1>{challenge.title}</h1>
				<p>{challenge.description}</p>
			</section>

			<section className="card stack-sm">
				<h3>Instance</h3>
				<p>
					Instance status:{" "}
					<strong>
						{instance?.status ?? (cooldownActive ? "cooldown" : "idle")}
					</strong>
				</p>
				<p className="info-text">
					Action state: <strong>{lifecycleActionState}</strong>
				</p>
				<p className="info-text">
					Start action:{" "}
					<strong>{startDisabled ? "disabled" : "enabled"}</strong>
					{startDisabledReason ? ` — ${startDisabledReason}` : ""}
				</p>
				<p className="info-text">
					Stop action: <strong>{stopDisabled ? "disabled" : "enabled"}</strong>
					{stopDisabledReason ? ` — ${stopDisabledReason}` : ""}
				</p>

				{instance?.expiresAt ? (
					<p className="info-text">Expires at: {instance.expiresAt}</p>
				) : null}
				{instance?.accessInfo ? (
					<p className="info-text">
						Access:{" "}
						{instance.accessInfo.connectionString ??
							`${instance.accessInfo.host}:${instance.accessInfo.port}`}
					</p>
				) : null}

				{cooldownActive ? (
					<>
						<p className="info-text">Retry at: {cooldownUntil}</p>
						<p className="info-text">
							Cooldown remaining: {cooldownRemainingSeconds}s
						</p>
					</>
				) : null}

				{instanceError ? <p className="error-text">{instanceError}</p> : null}

				{instanceChallengeMismatch && instance?.challengeId ? (
					<p className="error-text">
						Active instance belongs to another challenge ({instance.challengeId}
						). Open{" "}
						<a href={`/challenges/${instance.challengeId}`}>
							/challenges/{instance.challengeId}
						</a>{" "}
						to manage it. Start and stop actions are disabled on this page.
					</p>
				) : null}

				<div className="inline-actions">
					{instanceRunning ? (
						<button
							type="button"
							className="button danger"
							onClick={() => {
								void onStopInstance();
							}}
							disabled={stopDisabled}
						>
							{instanceAction === "stopping" ? "Stopping..." : "Stop instance"}
						</button>
					) : (
						<button
							type="button"
							className="button"
							onClick={() => {
								void onStartInstance();
							}}
							disabled={startDisabled}
						>
							{instanceAction === "starting" ? "Starting..." : "Start instance"}
						</button>
					)}
				</div>
			</section>

			<SubmissionForm
				token={token}
				challengeId={challenge.id}
				onSubmitted={onSubmitted}
			/>

			{latestSubmission ? (
				<section className="card">
					<h3>Latest submission result</h3>
					<p>
						Status: <strong>{latestSubmission.status}</strong>
					</p>
					<p>Awarded points: {latestSubmission.awardedPoints}</p>
					{latestSubmission.judgeJobId ? (
						<p>Judge job: {latestSubmission.judgeJobId}</p>
					) : null}

					{latestSubmission.status === "pending" ? (
						<p className="info-text">
							Dynamic judging is pending. Click refresh to fetch latest
							challenge submission status.
						</p>
					) : null}

					{refreshError ? <p className="error-text">{refreshError}</p> : null}

					<button
						type="button"
						className="button secondary"
						onClick={() => {
							void onManualRefresh();
						}}
						disabled={refreshing}
					>
						{refreshing ? "Refreshing..." : "Manual refresh"}
					</button>
				</section>
			) : null}

			<section className="card">
				<h3>Submission history</h3>
				{submissionHistory.length === 0 ? (
					<p className="empty-text">No submissions yet.</p>
				) : (
					<ul className="score-list">
						{submissionHistory.map((item) => (
							<li key={item.id}>
								{item.createdAt} — <strong>{item.status}</strong> —{" "}
								{item.awardedPoints} pts
							</li>
						))}
					</ul>
				)}
			</section>
		</div>
	);
}
