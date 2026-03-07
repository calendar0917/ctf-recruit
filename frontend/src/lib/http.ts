import { API_BASE_URL } from "@/lib/config";
import type { ApiErrorResponse, HttpRetryMetadata } from "@/lib/types";

type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE";

type RequestOptions = {
  method?: HttpMethod;
  token?: string;
  body?: unknown;
};

export class HttpError<TDetails = unknown> extends Error {
  status: number;
  code?: string;
  details?: TDetails;
  requestId?: string;
  retry?: HttpRetryMetadata;

  constructor(
    message: string,
    status: number,
    payload?: ApiErrorResponse<TDetails>,
    retry?: HttpRetryMetadata,
  ) {
    super(message);
    this.name = "HttpError";
    this.status = status;
    this.code = payload?.error?.code;
    this.details = payload?.error?.details;
    this.requestId = payload?.requestId;
    this.retry = retry;
  }
}

function parseRetryAt(value: unknown): { retryAt: string; retryAtMs: number } | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return undefined;
  }

  return {
    retryAt: new Date(parsed).toISOString(),
    retryAtMs: parsed,
  };
}

function parseRetryAfterHeader(value: string): HttpRetryMetadata | undefined {
  const trimmed = value.trim();
  if (!trimmed) {
    return undefined;
  }

  if (/^\d+$/.test(trimmed)) {
    const seconds = Number.parseInt(trimmed, 10);
    if (!Number.isFinite(seconds) || seconds < 0) {
      return undefined;
    }

    const nowMs = Date.now();
    const retryAtMs = nowMs + seconds * 1000;
    return {
      retryAfterSeconds: seconds,
      retryAtMs,
      retryAt: new Date(retryAtMs).toISOString(),
      source: "header.retry-after-seconds",
      headerValue: value,
    };
  }

  const parsedAt = parseRetryAt(trimmed);
  if (!parsedAt) {
    return undefined;
  }

  const deltaSeconds = Math.max(0, Math.ceil((parsedAt.retryAtMs - Date.now()) / 1000));
  return {
    retryAt: parsedAt.retryAt,
    retryAtMs: parsedAt.retryAtMs,
    retryAfterSeconds: deltaSeconds,
    source: "header.retry-after-date",
    headerValue: value,
  };
}

function normalizeRetryMetadata(
  payload: ApiErrorResponse | undefined,
  headers: Headers,
): HttpRetryMetadata | undefined {
  const detailsRecord =
    payload?.error?.details && typeof payload.error.details === "object"
      ? (payload.error.details as Record<string, unknown>)
      : undefined;

  const detailsRetryAt = parseRetryAt(detailsRecord?.retryAt);
  if (detailsRetryAt) {
    const retryAfterSeconds = Math.max(0, Math.ceil((detailsRetryAt.retryAtMs - Date.now()) / 1000));
    return {
      retryAt: detailsRetryAt.retryAt,
      retryAtMs: detailsRetryAt.retryAtMs,
      retryAfterSeconds,
      source: "details.retryAt",
    };
  }

  const retryAfterHeader = headers.get("Retry-After");
  if (!retryAfterHeader) {
    return undefined;
  }

  return parseRetryAfterHeader(retryAfterHeader);
}

function parseApiErrorPayload(payload: unknown): ApiErrorResponse {
  if (!payload || typeof payload !== "object") {
    return {};
  }

  const root = payload as Record<string, unknown>;
  const error =
    root.error && typeof root.error === "object"
      ? (root.error as Record<string, unknown>)
      : undefined;

  return {
    error: error
      ? {
          code: typeof error.code === "string" ? error.code : undefined,
          message: typeof error.message === "string" ? error.message : undefined,
          details: error.details,
        }
      : undefined,
    requestId: typeof root.requestId === "string" ? root.requestId : undefined,
  };
}

function resolveUrl(path: string): string {
  if (path.startsWith("http://") || path.startsWith("https://")) {
    return path;
  }

  if (!path.startsWith("/")) {
    return `${API_BASE_URL}/${path}`;
  }

  return `${API_BASE_URL}${path}`;
}

function parsePayload(rawText: string): unknown {
  if (!rawText) {
    return null;
  }

  try {
    return JSON.parse(rawText) as unknown;
  } catch {
    return null;
  }
}

export async function httpRequest<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const headers = new Headers({
    Accept: "application/json",
  });

  if (options.body !== undefined) {
    headers.set("Content-Type", "application/json");
  }

  if (options.token) {
    headers.set("Authorization", `Bearer ${options.token}`);
  }

  const response = await fetch(resolveUrl(path), {
    method: options.method ?? "GET",
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
    cache: "no-store",
  });

  const rawText = await response.text();
  const payload = parsePayload(rawText);

  if (!response.ok) {
    const typedPayload = parseApiErrorPayload(payload);
    const retry = normalizeRetryMetadata(typedPayload, response.headers);
    throw new HttpError(
      typedPayload.error?.message ?? `HTTP ${response.status}`,
      response.status,
      typedPayload,
      retry,
    );
  }

  if (response.status === 204 || payload === null) {
    return undefined as T;
  }

  return payload as T;
}
