import { expect, test } from "@playwright/test";
import {
	gotoChallengeDetailByTitle,
	loginViaUi,
	PLAYER_EMAIL,
	PLAYER_PASSWORD,
} from "./helpers";

test.describe("player lifecycle journey", () => {
	test("login -> challenge detail -> start/stop/cooldown path", async ({ page }) => {
		await loginViaUi(page, {
			email: PLAYER_EMAIL,
			password: PLAYER_PASSWORD,
		});

		await gotoChallengeDetailByTitle(page, "Log Trail");
		await page.getByRole("heading", { name: "Instance" }).waitFor();

		const mismatchWarning = page.getByText(
			/Active instance belongs to another challenge \(.+\)\./,
		);
		if (await mismatchWarning.isVisible().catch(() => false)) {
			throw new Error(
				"Environment has an active instance on another challenge; reset lifecycle state before running e2e.",
			);
		}

		const stopButton = page.getByRole("button", { name: "Stop instance" });
		if (await stopButton.isVisible().catch(() => false)) {
			await stopButton.click();
			await expect(page.getByText(/Retry at:/)).toBeVisible({ timeout: 30_000 });
		}

		const cooldownRemaining = page.getByText(/Cooldown remaining:/);
		if (await cooldownRemaining.isVisible().catch(() => false)) {
			await expect(cooldownRemaining).toBeHidden({ timeout: 90_000 });
		}

		const startButton = page.getByRole("button", { name: "Start instance" });
		await expect(startButton).toBeVisible({ timeout: 30_000 });
		await expect(startButton).toBeEnabled();
		await startButton.click();

		await expect(
			page.getByText(/Instance status:\s*(starting|running)/),
		).toBeVisible({ timeout: 30_000 });
		await expect(page.getByText(/Action state:\s*(starting|running)/)).toBeVisible({
			timeout: 30_000,
		});

		await expect(stopButton).toBeVisible({ timeout: 30_000 });
		await stopButton.click();

		await expect(page.getByText(/Retry at:/)).toBeVisible({ timeout: 30_000 });
		await expect(page.getByText(/Cooldown remaining:/)).toBeVisible({
			timeout: 30_000,
		});
		await expect(
			page.getByText(/Start action:\s*disabled\s*—\s*Cooldown active until/),
		).toBeVisible();
	});
});
