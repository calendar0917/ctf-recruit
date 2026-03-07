import { spawn } from "node:child_process";

const BASE_URL = "http://127.0.0.1:3001";
const API_BASE_URL = process.env.E2E_API_BASE_URL ?? "http://localhost:18080";
const NEXT_ARGS = ["exec", "next", "dev", "--hostname", "127.0.0.1", "--port", "3001"];
const PLAYWRIGHT_ARGS = ["exec", "playwright", "test", ...process.argv.slice(2)];

let serverProcess;

function runCommand(command, args, options = {}) {
	return spawn(command, args, {
		stdio: "inherit",
		env: process.env,
		shell: false,
		...options,
	});
}

function sleep(ms) {
	return new Promise((resolve) => {
		setTimeout(resolve, ms);
	});
}

async function waitForReady(timeoutMs = 120_000) {
	const deadline = Date.now() + timeoutMs;

	while (Date.now() < deadline) {
		if (serverProcess.exitCode !== null) {
			throw new Error("Next dev server exited before readiness.");
		}

		try {
			const response = await fetch(`${BASE_URL}/login`);
			if (response.status === 200) {
				return;
			}
		} catch {
			// Keep polling until timeout.
		}

		await sleep(1_000);
	}

	throw new Error(`Timed out waiting for ${BASE_URL}/login HTTP 200.`);
}

async function stopServer() {
	if (!serverProcess || serverProcess.exitCode !== null) {
		return;
	}

	await new Promise((resolve) => {
		serverProcess.once("exit", () => resolve());
		serverProcess.kill("SIGTERM");
		setTimeout(() => {
			if (serverProcess && serverProcess.exitCode === null) {
				serverProcess.kill("SIGKILL");
			}
		}, 5_000);
	});
}

async function main() {
	serverProcess = runCommand("pnpm", NEXT_ARGS, {
		env: {
			...process.env,
			NEXT_PUBLIC_API_BASE_URL: API_BASE_URL,
		},
	});

	const signalHandler = async () => {
		await stopServer();
		process.exit(1);
	};

	process.once("SIGINT", signalHandler);
	process.once("SIGTERM", signalHandler);

	try {
		await waitForReady();

		const playwright = runCommand("pnpm", PLAYWRIGHT_ARGS, {
			env: {
				...process.env,
				E2E_BASE_URL: BASE_URL,
			},
		});

		const exitCode = await new Promise((resolve, reject) => {
			playwright.once("error", reject);
			playwright.once("exit", (code, signal) => {
				if (signal) {
					reject(new Error(`Playwright exited due to signal: ${signal}`));
					return;
				}
				resolve(code ?? 1);
			});
		});

		process.exitCode = exitCode;
	} finally {
		await stopServer();
	}
}

main().catch(async (error) => {
	console.error(error instanceof Error ? error.message : String(error));
	await stopServer();
	process.exit(1);
});
