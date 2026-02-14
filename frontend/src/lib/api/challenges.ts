import { httpRequest } from "@/lib/http";
import type {
  Challenge,
  ChallengeListResponse,
  CreateChallengeRequest,
  UpdateChallengeRequest,
} from "@/lib/types";

const CHALLENGE_BASE = "/api/v1/challenges";

type ListParams = {
  limit?: number;
  offset?: number;
};

function buildListQuery(params: ListParams): string {
  const query = new URLSearchParams();

  if (params.limit !== undefined) {
    query.set("limit", String(params.limit));
  }
  if (params.offset !== undefined) {
    query.set("offset", String(params.offset));
  }

  const encoded = query.toString();
  return encoded ? `?${encoded}` : "";
}

export async function listChallenges(
  token: string,
  params: ListParams = {},
): Promise<ChallengeListResponse> {
  return httpRequest<ChallengeListResponse>(
    `${CHALLENGE_BASE}${buildListQuery(params)}`,
    {
      token,
    },
  );
}

export async function getChallenge(token: string, id: string): Promise<Challenge> {
  return httpRequest<Challenge>(`${CHALLENGE_BASE}/${id}`, {
    token,
  });
}

export async function createChallenge(
  token: string,
  payload: CreateChallengeRequest,
): Promise<Challenge> {
  return httpRequest<Challenge>(CHALLENGE_BASE, {
    method: "POST",
    token,
    body: payload,
  });
}

export async function updateChallenge(
  token: string,
  id: string,
  payload: UpdateChallengeRequest,
): Promise<Challenge> {
  return httpRequest<Challenge>(`${CHALLENGE_BASE}/${id}`, {
    method: "PUT",
    token,
    body: payload,
  });
}

export async function deleteChallenge(token: string, id: string): Promise<void> {
  return httpRequest<void>(`${CHALLENGE_BASE}/${id}`, {
    method: "DELETE",
    token,
  });
}
