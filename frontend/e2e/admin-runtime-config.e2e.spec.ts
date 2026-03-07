import { expect, test } from "@playwright/test";
import {
	ADMIN_EMAIL,
	ADMIN_PASSWORD,
	gotoChallengeDetailByTitle,
	loginViaUi,
	openChallengeEditor,
	PLAYER_EMAIL,
	PLAYER_PASSWORD,
} from "./helpers";

test.describe("admin runtime config journey", () => {
	test("admin edits runtime fields and player can observe challenge detail", async ({
		browser,
		page,
	}) => {
		await loginViaUi(page, {
			email: ADMIN_EMAIL,
			password: ADMIN_PASSWORD,
		});

		await page.goto("/admin/challenges");
		await page
			.getByRole("heading", { name: "Admin: Challenge management" })
			.waitFor();

		await openChallengeEditor(page, "Log Trail");

		await page.getByLabel("Runtime image (optional)").fill("busybox:1.36");
		await page
			.getByLabel("Runtime command (optional)")
			.fill("httpd -f -p 8080");
		await page.getByLabel("Runtime exposed port (optional)").fill("8080");

		await page.getByRole("button", { name: "Update challenge" }).click();

		await expect(
			page.getByRole("heading", { name: "Create challenge" }),
		).toBeVisible({ timeout: 30_000 });

		const playerContext = await browser.newContext();
		const playerPage = await playerContext.newPage();
		await loginViaUi(playerPage, {
			email: PLAYER_EMAIL,
			password: PLAYER_PASSWORD,
		});

		await gotoChallengeDetailByTitle(playerPage, "Log Trail");
		await expect(
			playerPage.getByText(/Start action:\s*(enabled|disabled)/),
		).toBeVisible({ timeout: 30_000 });

		const mismatchWarning = playerPage.getByText(
			/Active instance belongs to another challenge \(.+\)\./,
		);
		if (await mismatchWarning.isVisible().catch(() => false)) {
			throw new Error(
				"Environment has active instance mismatch for player; clear active instance before running admin-runtime e2e.",
			);
		}

		await playerContext.close();
	});
});
