"use client";

import React, { useEffect, useMemo, useState } from "react";
import { AdminSection } from "@/components/admin/AdminPrimitives";
import {
	ChallengeEditor,
	type ChallengeEditorValue,
} from "@/components/admin/ChallengeEditor";
import { ChallengeTable } from "@/components/admin/ChallengeTable";
import {
	createChallenge,
	deleteChallenge,
	listChallenges,
	updateChallenge,
} from "@/lib/api/challenges";
import { HttpError } from "@/lib/http";
import {
	buildCreateChallengePayload,
	buildUpdateChallengePayload,
	type Challenge,
	type ChallengeRuntimeFieldErrors,
	mapChallengeRuntimeFieldErrors,
} from "@/lib/types";
import { useRequireAuth } from "@/lib/use-auth";

void React;

export default function AdminChallengesPage() {
	const { session, ready } = useRequireAuth({ adminOnly: true });

	const [items, setItems] = useState<Challenge[]>([]);
	const [editing, setEditing] = useState<Challenge | null>(null);
	const [loadingList, setLoadingList] = useState(false);
	const [saving, setSaving] = useState(false);
	const [workingId, setWorkingId] = useState<string | undefined>();
	const [error, setError] = useState<string | undefined>();
	const [fieldErrors, setFieldErrors] = useState<
		ChallengeRuntimeFieldErrors | undefined
	>();

	const isAdmin = session?.user.role === "admin";

	useEffect(() => {
		if (!ready || !session || !isAdmin) {
			return;
		}

		const accessToken = session.accessToken;
		let cancelled = false;

		async function run() {
			setLoadingList(true);
			setError(undefined);
			try {
				const response = await listChallenges(accessToken, {
					limit: 100,
					offset: 0,
				});
				if (!cancelled) {
					setItems(response.items);
				}
			} catch (err) {
				if (cancelled) {
					return;
				}
				if (err instanceof HttpError) {
					setError(err.message);
				} else {
					setError("Failed to load admin challenge list.");
				}
			} finally {
				if (!cancelled) {
					setLoadingList(false);
				}
			}
		}

		void run();

		return () => {
			cancelled = true;
		};
	}, [isAdmin, ready, session]);

	async function refreshList(): Promise<void> {
		if (!session || !isAdmin) {
			return;
		}

		const accessToken = session.accessToken;
		const response = await listChallenges(accessToken, {
			limit: 100,
			offset: 0,
		});
		setItems(response.items);
	}

	async function handleSave(value: ChallengeEditorValue): Promise<void> {
		if (!session || !isAdmin) {
			return;
		}

		setSaving(true);
		setError(undefined);
		setFieldErrors(undefined);

		try {
			const accessToken = session.accessToken;
			if (value.id) {
				const payload = buildUpdateChallengePayload(value);
				await updateChallenge(accessToken, value.id, payload);
			} else {
				await createChallenge(accessToken, buildCreateChallengePayload(value));
			}

			await refreshList();
			setEditing(null);
			setFieldErrors(undefined);
		} catch (err) {
			if (err instanceof HttpError) {
				setError(err.message);
				const runtimeFieldErrors = mapChallengeRuntimeFieldErrors(err.message);
				if (Object.keys(runtimeFieldErrors).length > 0) {
					setFieldErrors(runtimeFieldErrors);
				}
			} else {
				setError("Failed to save challenge.");
				setFieldErrors(undefined);
			}
		} finally {
			setSaving(false);
		}
	}

	async function handleDelete(challenge: Challenge): Promise<void> {
		if (!session || !isAdmin) {
			return;
		}

		const accessToken = session.accessToken;
		setWorkingId(challenge.id);
		setError(undefined);
		setFieldErrors(undefined);
		try {
			await deleteChallenge(accessToken, challenge.id);
			await refreshList();
			if (editing?.id === challenge.id) {
				setEditing(null);
			}
		} catch (err) {
			if (err instanceof HttpError) {
				setError(err.message);
			} else {
				setError("Failed to delete challenge.");
			}
		} finally {
			setWorkingId(undefined);
		}
	}

	async function handleTogglePublish(challenge: Challenge): Promise<void> {
		if (!session || !isAdmin) {
			return;
		}

		const accessToken = session.accessToken;
		setWorkingId(challenge.id);
		setError(undefined);
		setFieldErrors(undefined);

		try {
			await updateChallenge(accessToken, challenge.id, {
				isPublished: !challenge.isPublished,
			});
			await refreshList();
		} catch (err) {
			if (err instanceof HttpError) {
				setError(err.message);
			} else {
				setError("Failed to update publish status.");
			}
		} finally {
			setWorkingId(undefined);
		}
	}

	const sorted = useMemo(
		() => [...items].sort((a, b) => a.title.localeCompare(b.title)),
		[items],
	);

	if (!ready) {
		return (
			<main className="page">
				<section className="card">
					<p>Loading session...</p>
				</section>
			</main>
		);
	}

	if (!session) {
		return (
			<main className="page">
				<section className="card">
					<p className="error-text">Unauthorized. Please login.</p>
				</section>
			</main>
		);
	}

	if (!isAdmin) {
		return (
			<main className="page">
				<section className="card">
					<h1>403</h1>
					<p className="error-text">Admin role required.</p>
				</section>
			</main>
		);
	}

	return (
		<main className="page page-content">
			<AdminSection>
				<h1>Admin: Challenge management</h1>
				<p>Create, update, delete, and publish challenges.</p>
			</AdminSection>

			<ChallengeEditor
				initial={editing}
				loading={saving}
				error={error}
				fieldErrors={fieldErrors}
				onSubmit={handleSave}
				onCancelEdit={() => {
					setEditing(null);
					setFieldErrors(undefined);
				}}
			/>

			<AdminSection>
				<h2>Existing challenges</h2>
				{loadingList ? <p>Loading challenges...</p> : null}
				{!loadingList ? (
					<ChallengeTable
						items={sorted}
						workingId={workingId}
						onEdit={(challenge) => setEditing(challenge)}
						onDelete={handleDelete}
						onTogglePublish={handleTogglePublish}
					/>
				) : null}
			</AdminSection>
		</main>
	);
}
