import { httpRequest } from "@/lib/http";
import type {
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  User,
} from "@/lib/types";

const AUTH_BASE = "/api/v1/auth";

export async function register(payload: RegisterRequest): Promise<User> {
  return httpRequest<User>(`${AUTH_BASE}/register`, {
    method: "POST",
    body: payload,
  });
}

export async function login(payload: LoginRequest): Promise<LoginResponse> {
  return httpRequest<LoginResponse>(`${AUTH_BASE}/login`, {
    method: "POST",
    body: payload,
  });
}

export async function me(token: string): Promise<User> {
  return httpRequest<User>(`${AUTH_BASE}/me`, {
    token,
  });
}
