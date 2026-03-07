import { beforeEach, describe, expect, it, vi } from "vitest";
import { HttpError } from "@/lib/http";
import type {
	Challenge,
	ChallengeInstance,
	MyChallengeInstanceResponse,
	SubmissionListResponse,
	SubmissionResponse,
} from "@/lib/types";

type EffectState = {
	deps?: unknown[];
	cleanup?: (() => void) | undefined;
};

class HookHarness {
	private hookIndex = 0;

	private states: unknown[] = [];

	private refs: unknown[] = [];

	private callbacks: Array<{
		deps?: unknown[];
		fn: unknown;
	}> = [];

	private effects: EffectState[] = [];

	private pendingEffects: Array<{
		index: number;
		effect: () => (() => void) | undefined;
		deps?: unknown[];
	}> = [];

	beginRender() {
		this.hookIndex = 0;
		this.pendingEffects = [];
	}

	commitEffects() {
		for (const pending of this.pendingEffects) {
			const current = this.effects[pending.index];
			if (current?.cleanup) {
				current.cleanup();
			}

			const cleanup = pending.effect();
			this.effects[pending.index] = {
				deps: pending.deps,
				cleanup,
			};
		}
	}

	unmount() {
		for (const effect of this.effects) {
			effect?.cleanup?.();
		}
		this.effects = [];
	}

	useState<T>(initial: T | (() => T)): [T, (next: T | ((prev: T) => T)) => void] {
		const slot = this.hookIndex++;
		if (!(slot in this.states)) {
			this.states[slot] =
				typeof initial === "function" ? (initial as () => T)() : initial;
		}

		const setState = (next: T | ((prev: T) => T)) => {
			const prev = this.states[slot] as T;
			this.states[slot] =
				typeof next === "function" ? (next as (prev: T) => T)(prev) : next;
		};

		return [this.states[slot] as T, setState];
	}

	useRef<T>(initial: T): { current: T } {
		const slot = this.hookIndex++;
		if (!(slot in this.refs)) {
			this.refs[slot] = { current: initial };
		}

		return this.refs[slot] as { current: T };
	}

	useCallback<T extends (...args: never[]) => unknown>(
		fn: T,
		deps?: unknown[],
	): T {
		const slot = this.hookIndex++;
		const previous = this.callbacks[slot];
		const changed =
			!previous ||
			deps === undefined ||
			previous.deps === undefined ||
			deps.length !== previous.deps.length ||
			deps.some((value, index) => !Object.is(value, previous.deps?.[index]));

		if (!changed) {
			return previous.fn as T;
		}

		this.callbacks[slot] = {
			deps,
			fn,
		};

		return fn;
	}

	useEffect(effect: () => (() => void) | undefined, deps?: unknown[]) {
		const slot = this.hookIndex++;
		const previous = this.effects[slot];
		const changed =
			!previous ||
			deps === undefined ||
			previous.deps === undefined ||
			deps.length !== previous.deps.length ||
			deps.some((value, index) => !Object.is(value, previous.deps?.[index]));

		if (!changed) {
			return;
		}

		this.pendingEffects.push({
			index: slot,
			effect,
			deps,
		});
	}
}

let currentHarness: HookHarness | null = null;

const mocks = vi.hoisted(() => ({
	useParams: vi.fn(),
	useRequireAuth: vi.fn(),
	getChallenge: vi.fn(),
	getMyInstance: vi.fn(),
	listMySubmissionsByChallenge: vi.fn(),
	startInstance: vi.fn(),
	stopInstance: vi.fn(),
	resolveMyInstanceCooldownUntil: vi.fn(
		(response: Pick<MyChallengeInstanceResponse, "instance" | "cooldown">) =>
			response.instance?.cooldownUntil ?? response.cooldown?.retryAt,
	),
	setInterval: vi.fn(),
	clearInterval: vi.fn(),
}));

vi.mock("react", async () => {
	const actual = await vi.importActual<typeof import("react")>("react");

	return {
		...actual,
		useState: <T,>(initial: T | (() => T)) => {
			if (!currentHarness) {
				throw new Error("HookHarness is not initialized.");
			}
			return currentHarness.useState(initial);
		},
		useRef: <T,>(initial: T) => {
			if (!currentHarness) {
				throw new Error("HookHarness is not initialized.");
			}
			return currentHarness.useRef(initial);
		},
		useCallback: <T extends (...args: never[]) => unknown>(
			fn: T,
			deps?: unknown[],
		) => {
			if (!currentHarness) {
				throw new Error("HookHarness is not initialized.");
			}
			return currentHarness.useCallback(fn, deps);
		},
		useEffect: (
			effect: () => (() => void) | undefined,
			deps?: unknown[],
		) => {
			if (!currentHarness) {
				throw new Error("HookHarness is not initialized.");
			}
			return currentHarness.useEffect(effect, deps);
		},
	};
});

vi.mock("next/navigation", () => ({
	useParams: () => mocks.useParams(),
}));

vi.mock("@/lib/use-auth", () => ({
	useRequireAuth: () => mocks.useRequireAuth(),
}));

vi.mock("@/lib/api/challenges", () => ({
	getChallenge: (...args: unknown[]) => mocks.getChallenge(...args),
}));

vi.mock("@/lib/api/instances", () => ({
	getMyInstance: (...args: unknown[]) => mocks.getMyInstance(...args),
	resolveMyInstanceCooldownUntil: (
		...args: [Pick<MyChallengeInstanceResponse, "instance" | "cooldown">]
	) => mocks.resolveMyInstanceCooldownUntil(...args),
	startInstance: (...args: unknown[]) => mocks.startInstance(...args),
	stopInstance: (...args: unknown[]) => mocks.stopInstance(...args),
}));

vi.mock("@/lib/api/submissions", () => ({
	listMySubmissionsByChallenge: (...args: unknown[]) =>
		mocks.listMySubmissionsByChallenge(...args),
}));

function createDeferred<T>() {
	let resolve: ((value: T) => void) | undefined;
	let reject: ((reason?: unknown) => void) | undefined;

	const promise = new Promise<T>((resolvePromise, rejectPromise) => {
		resolve = resolvePromise;
		reject = rejectPromise;
	});

	return {
		promise,
		resolve: (value: T) => resolve?.(value),
		reject: (reason?: unknown) => reject?.(reason),
	};
}

async function flushMicrotasks(rounds = 5): Promise<void> {
	for (let index = 0; index < rounds; index += 1) {
		await Promise.resolve();
	}
}

async function importPageModule() {
	return import("@/app/challenges/[id]/page");
}

function renderPage(
	pageModule: Awaited<ReturnType<typeof importPageModule>>,
	harness: HookHarness,
): unknown {
	harness.beginRender();
	const element = pageModule.default();
	harness.commitEffects();
	return element;
}

type ChallengeDetailPropsShape = {
	challenge: Challenge;
	instance: ChallengeInstance | null;
	cooldownUntil?: string;
	latestSubmission: SubmissionResponse | null;
	submissionHistory: SubmissionResponse[];
	instanceError?: string;
	onStartInstance: () => Promise<void>;
	onStopInstance: () => Promise<void>;
};

function extractChallengeDetailProps(element: unknown): ChallengeDetailPropsShape {
	const child = (element as { props?: { children?: unknown } })?.props?.children as {
		props?: ChallengeDetailPropsShape;
	};

	if (!child?.props?.challenge || !child?.props?.onStartInstance) {
		throw new Error("ChallengeDetail props are not available in current render.");
	}

	return child.props;
}

const baseChallenge: Challenge = {
	id: "challenge-1",
	title: "Log Trail",
	description: "Analyze runtime logs",
	category: "ops",
	difficulty: "medium",
	mode: "dynamic",
	points: 200,
	isPublished: true,
	createdAt: "2026-02-17T00:00:00Z",
	updatedAt: "2026-02-17T00:00:00Z",
};

const pendingSubmission: SubmissionResponse = {
	id: "sub-pending",
	challengeId: "challenge-1",
	status: "pending",
	awardedPoints: 0,
	createdAt: "2026-02-17T00:00:00Z",
};

beforeEach(() => {
	vi.clearAllMocks();

	mocks.useParams.mockReturnValue({ id: "challenge-1" });
	mocks.useRequireAuth.mockReturnValue({
		ready: true,
		authorized: true,
		session: {
			accessToken: "player-token",
			tokenType: "Bearer",
			user: {
				id: "u-player",
				email: "player@example.com",
				displayName: "Player",
				role: "player",
			},
		},
	});

	mocks.setInterval.mockImplementation((_cb: () => void, _ms: number) => 101);
	mocks.clearInterval.mockImplementation((_timer: number) => undefined);

	Object.defineProperty(globalThis, "window", {
		configurable: true,
		writable: true,
		value: {
			setInterval: mocks.setInterval,
			clearInterval: mocks.clearInterval,
		},
	});
});

describe("challenge detail page orchestration", () => {
	it("loads challenge + me + submissions in parallel and applies hydrated state", async () => {
		const challengeDeferred = createDeferred<Challenge>();
		const instanceDeferred = createDeferred<MyChallengeInstanceResponse>();
		const submissionsDeferred = createDeferred<SubmissionListResponse>();

		mocks.getChallenge.mockReturnValue(challengeDeferred.promise);
		mocks.getMyInstance.mockReturnValue(instanceDeferred.promise);
		mocks.listMySubmissionsByChallenge.mockReturnValue(submissionsDeferred.promise);

		const harness = new HookHarness();
		currentHarness = harness;
		const pageModule = await importPageModule();

		renderPage(pageModule, harness);

		expect(mocks.getChallenge).toHaveBeenCalledWith("player-token", "challenge-1");
		expect(mocks.getMyInstance).toHaveBeenCalledWith("player-token");
		expect(mocks.listMySubmissionsByChallenge).toHaveBeenCalledWith(
			"player-token",
			"challenge-1",
			{ limit: 20, offset: 0 },
		);

		challengeDeferred.resolve(baseChallenge);
		instanceDeferred.resolve({
			instance: {
				id: "inst-1",
				userId: "u-player",
				challengeId: "challenge-1",
				status: "running",
			},
		});
		submissionsDeferred.resolve({
			items: [pendingSubmission],
			limit: 20,
			offset: 0,
		});

		await flushMicrotasks();
		const hydrated = renderPage(pageModule, harness);
		const props = extractChallengeDetailProps(hydrated);

		expect(props.challenge.id).toBe("challenge-1");
		expect(props.instance?.id).toBe("inst-1");
		expect(props.latestSubmission?.id).toBe("sub-pending");
		expect(props.submissionHistory).toHaveLength(1);
	});

	it("applies retryAt cooldown and runs reconcile after start conflict", async () => {
		const retryAt = "2026-02-17T00:02:00.000Z";

		mocks.getChallenge.mockResolvedValue(baseChallenge);
		mocks.getMyInstance
			.mockResolvedValueOnce({ instance: null })
			.mockResolvedValueOnce({ instance: null, cooldown: { retryAt } });
		mocks.listMySubmissionsByChallenge.mockResolvedValue({
			items: [],
			limit: 20,
			offset: 0,
		});
		mocks.startInstance.mockRejectedValue(
			new HttpError("Instance is cooling down", 409, {
				error: {
					code: "INSTANCE_COOLDOWN_ACTIVE",
					message: "Instance is cooling down",
					details: { retryAt },
				},
			}),
		);

		const harness = new HookHarness();
		currentHarness = harness;
		const pageModule = await importPageModule();

		renderPage(pageModule, harness);
		await flushMicrotasks();
		const loaded = renderPage(pageModule, harness);
		const beforeStart = extractChallengeDetailProps(loaded);

		await beforeStart.onStartInstance();
		await flushMicrotasks();
		const afterStart = renderPage(pageModule, harness);
		const props = extractChallengeDetailProps(afterStart);

		expect(mocks.startInstance).toHaveBeenCalledWith("player-token", {
			challengeId: "challenge-1",
		});
		expect(mocks.getMyInstance).toHaveBeenCalledTimes(2);
		expect(props.cooldownUntil).toBe(retryAt);
		expect(props.instanceError).toContain("INSTANCE_COOLDOWN_ACTIVE");
		expect(props.instanceError).toContain(`Retry after ${retryAt}.`);
	});

	it("runs reconcile on stop failure and surfaces reconciliation diagnostics", async () => {
		const runtimeFailure = new HttpError("Runtime unavailable", 500, {
			error: {
				code: "INSTANCE_RUNTIME_STOP_FAILED",
				message: "Runtime unavailable",
			},
		});
		const reconcileFailure = new HttpError("state probe failed", 503, {
			error: {
				code: "STATE_PROBE_FAILED",
				message: "state probe failed",
			},
		});

		mocks.getChallenge.mockResolvedValue(baseChallenge);
		mocks.getMyInstance
			.mockResolvedValueOnce({
				instance: {
					id: "inst-running",
					userId: "u-player",
					challengeId: "challenge-1",
					status: "running",
				},
			})
			.mockRejectedValueOnce(reconcileFailure);
		mocks.listMySubmissionsByChallenge.mockResolvedValue({
			items: [],
			limit: 20,
			offset: 0,
		});
		mocks.stopInstance.mockRejectedValue(runtimeFailure);

		const harness = new HookHarness();
		currentHarness = harness;
		const pageModule = await importPageModule();

		renderPage(pageModule, harness);
		await flushMicrotasks();
		const loaded = renderPage(pageModule, harness);
		const beforeStop = extractChallengeDetailProps(loaded);

		await beforeStop.onStopInstance();
		await flushMicrotasks();
		const afterStop = renderPage(pageModule, harness);
		const props = extractChallengeDetailProps(afterStop);

		expect(mocks.stopInstance).toHaveBeenCalledWith("player-token", {
			instanceId: "inst-running",
		});
		expect(mocks.getMyInstance).toHaveBeenCalledTimes(2);
		expect(props.instanceError).toContain("INSTANCE_RUNTIME_STOP_FAILED");
		expect(props.instanceError).toContain("Reconciliation check failed");
		expect(props.instanceError).toContain("STATE_PROBE_FAILED");
	});

	it("activates polling intervals for pending/transition/cooldown and clears them on unmount", async () => {
		vi.spyOn(Date, "now").mockReturnValue(1_700_000_000_000);

		let timerId = 200;
		mocks.setInterval.mockImplementation((_cb: () => void, _ms: number) => {
			timerId += 1;
			return timerId;
		});

		mocks.getChallenge.mockResolvedValue(baseChallenge);
		mocks.getMyInstance.mockResolvedValue({
			instance: {
				id: "inst-starting",
				userId: "u-player",
				challengeId: "challenge-1",
				status: "starting",
				cooldownUntil: "2023-11-14T22:13:40.000Z",
			},
		});
		mocks.listMySubmissionsByChallenge.mockResolvedValue({
			items: [pendingSubmission],
			limit: 20,
			offset: 0,
		});

		const harness = new HookHarness();
		currentHarness = harness;
		const pageModule = await importPageModule();

		renderPage(pageModule, harness);
		await flushMicrotasks();
		renderPage(pageModule, harness);

		const intervalMs = mocks.setInterval.mock.calls.map((call) => call[1]);
		expect(intervalMs).toContain(5000);
		expect(intervalMs).toContain(3000);
		expect(intervalMs).toContain(1000);

		harness.unmount();
		expect(mocks.clearInterval).toHaveBeenCalled();
	});
});
