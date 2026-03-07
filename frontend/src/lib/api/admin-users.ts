import { httpRequest } from "@/lib/http";
import type { AdminUpdateUserRequest, AdminUserListResponse, User } from "@/lib/types";

const ADMIN_USERS_BASE = "/api/v1/admin/users";

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

export async function listAdminUsers(token: string, params: ListParams = {}): Promise<AdminUserListResponse> {
  return httpRequest<AdminUserListResponse>(`${ADMIN_USERS_BASE}${buildListQuery(params)}`, {
    token,
  });
}

export async function updateAdminUser(token: string, id: string, payload: AdminUpdateUserRequest): Promise<User> {
  return httpRequest<User>(`${ADMIN_USERS_BASE}/${id}`, {
    method: "PATCH",
    token,
    body: payload,
  });
}
