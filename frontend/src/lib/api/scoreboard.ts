import { httpRequest } from "@/lib/http";
import type { ScoreboardResponse } from "@/lib/types";

const SCOREBOARD_BASE = "/api/v1/scoreboard";

type ListScoreboardParams = {
  limit?: number;
  offset?: number;
};

function buildQuery(params: ListScoreboardParams): string {
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

export async function listScoreboard(
  token: string,
  params: ListScoreboardParams = {},
): Promise<ScoreboardResponse> {
  return httpRequest<ScoreboardResponse>(`${SCOREBOARD_BASE}${buildQuery(params)}`, {
    token,
  });
}
