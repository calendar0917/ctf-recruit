import { httpRequest } from "@/lib/http";
import type { CreateSubmissionRequest, SubmissionResponse } from "@/lib/types";

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
