import { httpRequest } from "@/lib/http";
import type {
  Announcement,
  AnnouncementListResponse,
  CreateAnnouncementRequest,
  UpdateAnnouncementRequest,
} from "@/lib/types";

const ANNOUNCEMENT_BASE = "/api/v1/announcements";

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

export async function listAnnouncements(
  token: string,
  params: ListParams = {},
): Promise<AnnouncementListResponse> {
  return httpRequest<AnnouncementListResponse>(
    `${ANNOUNCEMENT_BASE}${buildListQuery(params)}`,
    {
      token,
    },
  );
}

export async function getAnnouncement(token: string, id: string): Promise<Announcement> {
  return httpRequest<Announcement>(`${ANNOUNCEMENT_BASE}/${id}`, {
    token,
  });
}

export async function createAnnouncement(
  token: string,
  payload: CreateAnnouncementRequest,
): Promise<Announcement> {
  return httpRequest<Announcement>(ANNOUNCEMENT_BASE, {
    method: "POST",
    token,
    body: payload,
  });
}

export async function updateAnnouncement(
  token: string,
  id: string,
  payload: UpdateAnnouncementRequest,
): Promise<Announcement> {
  return httpRequest<Announcement>(`${ANNOUNCEMENT_BASE}/${id}`, {
    method: "PUT",
    token,
    body: payload,
  });
}

export async function deleteAnnouncement(token: string, id: string): Promise<void> {
  return httpRequest<void>(`${ANNOUNCEMENT_BASE}/${id}`, {
    method: "DELETE",
    token,
  });
}
