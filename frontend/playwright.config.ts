import { defineConfig, devices } from "@playwright/test";

const FRONTEND_BASE_URL =
	process.env.E2E_BASE_URL ?? process.env.PLAYWRIGHT_BASE_URL ?? "http://127.0.0.1:3001";

export default defineConfig({
	testDir: "./e2e",
	fullyParallel: false,
	forbidOnly: Boolean(process.env.CI),
	retries: process.env.CI ? 2 : 0,
	workers: 1,
	reporter: [["list"], ["html", { open: "never" }]],
	use: {
		baseURL: FRONTEND_BASE_URL,
		trace: "retain-on-failure",
		video: "on",
		screenshot: "only-on-failure",
		actionTimeout: 10_000,
		navigationTimeout: 20_000,
	},
	projects: [
		{
			name: "chromium",
			use: { ...devices["Desktop Chrome"] },
		},
	],
});
