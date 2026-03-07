import { HttpError, httpRequest } from "@/lib/http";
import type {
  ChallengeInstance,
  InstanceActiveExistsDetails,
  MyChallengeInstanceResponse,
  StartChallengeInstanceRequest,
  StopChallengeInstanceRequest,
} from "@/lib/types";
import { isInstanceActiveExistsDetails } from "@/lib/types";

const INSTANCES_BASE = "/api/v1/instances";

export async function getMyInstance(token: string): Promise<MyChallengeInstanceResponse> {
  return httpRequest<MyChallengeInstanceResponse>(`${INSTANCES_BASE}/me`, {
    method: "GET",
    token,
  });
}

export function resolveMyInstanceCooldownUntil(
  response: Pick<MyChallengeInstanceResponse, "instance" | "cooldown">,
): string | undefined {
  if (response.instance?.cooldownUntil) {
    return response.instance.cooldownUntil;
  }

  if (response.cooldown?.retryAt) {
    return response.cooldown.retryAt;
  }

  return undefined;
}

export async function startInstance(
  token: string,
  payload: StartChallengeInstanceRequest,
): Promise<ChallengeInstance> {
  return httpRequest<ChallengeInstance>(`${INSTANCES_BASE}/start`, {
    method: "POST",
    token,
    body: payload,
  });
}

export function resolveActiveInstanceConflictDetails(
  error: unknown,
): InstanceActiveExistsDetails | undefined {
  if (!(error instanceof HttpError) || error.code !== "INSTANCE_ACTIVE_EXISTS") {
    return undefined;
  }

  if (!isInstanceActiveExistsDetails(error.details)) {
    return undefined;
  }

  return error.details;
}

export async function stopInstance(
  token: string,
  payload: StopChallengeInstanceRequest = {},
): Promise<ChallengeInstance> {
  return httpRequest<ChallengeInstance>(`${INSTANCES_BASE}/stop`, {
    method: "POST",
    token,
    body: payload,
  });
}
