import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { ChallengeDetail } from "@/components/challenge/ChallengeDetail";
import type { Challenge, ChallengeInstance, SubmissionResponse } from "@/lib/types";

void React;

const challenge: Challenge = {
  id: "11111111-1111-1111-1111-111111111111",
  title: "Dynamic Challenge",
  description: "Judge this challenge asynchronously",
  category: "web",
  difficulty: "medium",
  mode: "dynamic",
  points: 300,
  isPublished: true,
  createdAt: "2026-02-16T10:00:00Z",
  updatedAt: "2026-02-16T10:00:00Z",
};

type RenderOptions = {
  latestSubmission: SubmissionResponse | null;
  history: SubmissionResponse[];
  instance?: ChallengeInstance | null;
  instanceChallengeMismatch?: boolean;
  cooldownUntil?: string;
  nowMs?: number;
  instanceAction?: "starting" | "stopping" | null;
};

function render(options: RenderOptions): string {
  const {
    latestSubmission,
    history,
    instance = null,
    instanceChallengeMismatch = false,
    cooldownUntil,
    nowMs = Date.parse("2026-02-16T10:00:00Z"),
    instanceAction = null,
  } = options;

  return renderToStaticMarkup(
    <ChallengeDetail
      token="player-token"
      challenge={challenge}
      latestSubmission={latestSubmission}
      submissionHistory={history}
      instance={instance}
      instanceAction={instanceAction}
      instanceChallengeMismatch={instanceChallengeMismatch}
      cooldownUntil={cooldownUntil}
      nowMs={nowMs}
      refreshing={false}
      onSubmitted={() => undefined}
      onStartInstance={async () => undefined}
      onStopInstance={async () => undefined}
      onManualRefresh={async () => undefined}
    />,
  );
}

describe("challenge detail submission status", () => {
  it("renders pending status and resolves to correct", () => {
    const pending: SubmissionResponse = {
      id: "sub-pending",
      challengeId: challenge.id,
      status: "pending",
      awardedPoints: 0,
      judgeJobId: "job-1",
      createdAt: "2026-02-16T10:01:00Z",
    };

    const resolved: SubmissionResponse = {
      id: "sub-pending",
      challengeId: challenge.id,
      status: "correct",
      awardedPoints: 300,
      createdAt: "2026-02-16T10:02:00Z",
    };

    const pendingHtml = render({ latestSubmission: pending, history: [pending] });
    expect(pendingHtml).toContain("Dynamic judging is pending");
    expect(pendingHtml).toContain("Status: <strong>pending</strong>");

    const resolvedHtml = render({ latestSubmission: resolved, history: [resolved, pending] });
    expect(resolvedHtml).toContain("Status: <strong>correct</strong>");
    expect(resolvedHtml).not.toContain("Dynamic judging is pending");
  });

  it("shows Stop only when instance is running", () => {
    const runningHtml = render({
      latestSubmission: null,
      history: [],
      instance: {
        id: "inst-1",
        userId: "u-1",
        challengeId: challenge.id,
        status: "running",
      },
    });

    expect(runningHtml).toContain("Stop instance");
    expect(runningHtml).not.toContain("Start instance");
    expect(runningHtml).toContain("Action state: <strong>running</strong>");
    expect(runningHtml).toContain(
      "Start action: <strong>disabled</strong> — Instance already active. Stop it before starting a new one.",
    );
    expect(runningHtml).toContain("Stop action: <strong>enabled</strong>");
  });

  it("shows cooldown retry info and disables Start", () => {
    const cooldownUntil = "2026-02-16T10:01:20Z";
    const cooldownHtml = render({
      latestSubmission: null,
      history: [],
      instance: null,
      cooldownUntil,
      nowMs: Date.parse("2026-02-16T10:01:00Z"),
    });

    expect(cooldownHtml).toContain("Instance status: <strong>cooldown</strong>");
    expect(cooldownHtml).toContain("Action state: <strong>cooldown</strong>");
    expect(cooldownHtml).toContain(`Retry at: ${cooldownUntil}`);
    expect(cooldownHtml).toContain("Cooldown remaining: 20s");
    expect(cooldownHtml).toContain(
      `Start action: <strong>disabled</strong> — Cooldown active until ${cooldownUntil}.`,
    );
    expect(cooldownHtml).toContain("Start instance");
    expect(cooldownHtml).toContain("disabled");
  });

  it("shows deterministic mismatch message and disables unsafe actions", () => {
    const mismatchChallengeId = "22222222-2222-2222-2222-222222222222";
    const mismatchHtml = render({
      latestSubmission: null,
      history: [],
      instance: {
        id: "inst-mismatch",
        userId: "u-1",
        challengeId: mismatchChallengeId,
        status: "running",
      },
      instanceChallengeMismatch: true,
    });

    expect(mismatchHtml).toContain(
      `Active instance belongs to another challenge (${mismatchChallengeId}). Open`,
    );
    expect(mismatchHtml).toContain(`/challenges/${mismatchChallengeId}`);
    expect(mismatchHtml).toContain("Start and stop actions are disabled on this page.");
    expect(mismatchHtml).toContain("Action state: <strong>mismatch</strong>");
    expect(mismatchHtml).toContain(
      `Start action: <strong>disabled</strong> — Blocked by active instance mismatch (${mismatchChallengeId}).`,
    );
    expect(mismatchHtml).toContain(
      `Stop action: <strong>disabled</strong> — Blocked by active instance mismatch (${mismatchChallengeId}).`,
    );
    expect(mismatchHtml).toContain("Stop instance");
    expect(mismatchHtml).toContain("disabled");
  });

  it("keeps actions available when active instance matches the challenge", () => {
    const matchingHtml = render({
      latestSubmission: null,
      history: [],
      instance: {
        id: "inst-match",
        userId: "u-1",
        challengeId: challenge.id,
        status: "running",
      },
      instanceChallengeMismatch: false,
    });

    expect(matchingHtml).toContain("Stop instance");
    expect(matchingHtml).not.toContain("Active instance belongs to another challenge");
  });
});
