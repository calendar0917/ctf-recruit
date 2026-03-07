import { httpRequest } from "@/lib/http";
import type {
  CreateSubmissionRequest,
  SubmissionListResponse,
  SubmissionResponse,
} from "@/lib/types";

const SUBMISSION_BASE = "/api/v1/submissions";

export async function createSubmission(
  token: string,
  payload: CreateSubmissionRequest,
): Promise<SubmissionResponse> {
  return httpRequest<SubmissionResponse>(SUBMISSION_BASE, {
    method: "POST",
    token,
    body: payload,
  });
}

type ListSubmissionsQuery = {
  limit?: number;
  offset?: number;
};

function buildQuery(query: ListSubmissionsQuery = {}): string {
  const params = new URLSearchParams();
  if (query.limit !== undefined) {
    params.set("limit", String(query.limit));
  }
  if (query.offset !== undefined) {
    params.set("offset", String(query.offset));
  }
  const raw = params.toString();
  return raw ? `?${raw}` : "";
}

export async function listMySubmissions(
  token: string,
  query: ListSubmissionsQuery = {},
): Promise<SubmissionListResponse> {
  return httpRequest<SubmissionListResponse>(`${SUBMISSION_BASE}/me${buildQuery(query)}`, {
    method: "GET",
    token,
  });
}

export async function listMySubmissionsByChallenge(
  token: string,
  challengeId: string,
  query: ListSubmissionsQuery = {},
): Promise<SubmissionListResponse> {
  return httpRequest<SubmissionListResponse>(
    `${SUBMISSION_BASE}/challenge/${challengeId}${buildQuery(query)}`,
    {
      method: "GET",
      token,
    },
  );
}
