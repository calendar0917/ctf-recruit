import { httpRequest } from "@/lib/http";
import type {
  CreateRecruitmentSubmissionRequest,
  RecruitmentSubmission,
  RecruitmentSubmissionListResponse,
} from "@/lib/types";

const RECRUITMENT_BASE = "/api/v1/recruitments";

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

export async function createRecruitmentSubmission(
  token: string,
  payload: CreateRecruitmentSubmissionRequest,
): Promise<RecruitmentSubmission> {
  return httpRequest<RecruitmentSubmission>(RECRUITMENT_BASE, {
    method: "POST",
    token,
    body: payload,
  });
}

export async function listRecruitmentSubmissions(
  token: string,
  params: ListParams = {},
): Promise<RecruitmentSubmissionListResponse> {
  return httpRequest<RecruitmentSubmissionListResponse>(
    `${RECRUITMENT_BASE}${buildListQuery(params)}`,
    {
      token,
    },
  );
}

export async function getRecruitmentSubmission(
  token: string,
  id: string,
): Promise<RecruitmentSubmission> {
  return httpRequest<RecruitmentSubmission>(`${RECRUITMENT_BASE}/${id}`, {
    token,
  });
}
