import type { Locator, Page } from "@playwright/test";

export const PLAYER_EMAIL = "player@ctf.local";
export const PLAYER_PASSWORD = "PlayerPass123!";
export const ADMIN_EMAIL = "admin@ctf.local";
export const ADMIN_PASSWORD = "AdminPass123!";

export async function loginViaUi(
	page: Page,
	credentials: { email: string; password: string },
): Promise<void> {
	await page.goto("/login");
	await page.getByLabel("Email").fill(credentials.email);
	await page.getByLabel("Password").fill(credentials.password);
	await page.getByRole("button", { name: "Sign in" }).click();
	await page.waitForURL("**/challenges", { timeout: 30_000 });
	await page.getByRole("heading", { name: "Challenges" }).waitFor();
}

export async function gotoChallengeDetailByTitle(
	page: Page,
	challengeTitle: string,
): Promise<void> {
	const exactChallengeCard = page
		.locator("article")
		.filter({ has: page.getByRole("heading", { name: challengeTitle }) })
		.first();
	const fallbackChallengeCard = page.locator("article").first();

	const challengeCard =
		(await exactChallengeCard.isVisible().catch(() => false))
			? exactChallengeCard
			: fallbackChallengeCard;

	await challengeCard.waitFor({ state: "visible", timeout: 30_000 });
	await challengeCard.getByRole("link", { name: "View challenge" }).click();
	await page.waitForURL("**/challenges/*", { timeout: 30_000 });

	const detailReadyMarkers = [
		page.getByRole("heading", { name: "Instance" }),
		page.getByText(/Start action:\s*(enabled|disabled)/),
		page.getByRole("button", { name: "Start instance" }),
	];

	await Promise.race(
		detailReadyMarkers.map((locator) => locator.waitFor({ timeout: 30_000 })),
	);
}

export async function openChallengeEditor(
	page: Page,
	title: string,
): Promise<{ row: Locator; editButton: Locator }> {
	const row = page.locator("tbody tr").filter({ hasText: title }).first();
	await row.waitFor({ state: "visible", timeout: 30_000 });
	const editButton = row.getByRole("button", { name: "Edit" });
	await editButton.waitFor({ state: "visible" });
	await editButton.click();
	await page
		.getByRole("heading", {
			name: "Edit challenge",
		})
		.waitFor();

	return { row, editButton };
}
