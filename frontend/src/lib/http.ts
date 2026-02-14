import { API_BASE_URL } from "@/lib/config";
import type { ApiErrorResponse } from "@/lib/types";

type HttpMethod = "GET" | "POST" | "PUT" | "DELETE";

type RequestOptions = {
  method?: HttpMethod;
  token?: string;
  body?: unknown;
};

export class HttpError extends Error {
  status: number;
  code?: string;
  details?: unknown;
  requestId?: string;

  constructor(message: string, status: number, payload?: ApiErrorResponse) {
    super(message);
    this.name = "HttpError";
    this.status = status;
    this.code = payload?.error?.code;
    this.details = payload?.error?.details;
    this.requestId = payload?.requestId;
  }
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
    const typedPayload = (payload ?? {}) as ApiErrorResponse;
    throw new HttpError(
      typedPayload.error?.message ?? `HTTP ${response.status}`,
      response.status,
      typedPayload,
    );
  }

  if (response.status === 204 || payload === null) {
    return undefined as T;
  }

  return payload as T;
}
