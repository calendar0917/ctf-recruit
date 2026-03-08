export type ApiError = {
  error: string
  message: string
}

export type AuthUser = {
  id: number
  role: string
  username: string
  email: string
  display_name: string
  status: string
  last_login_at?: string | null
}

export type AuthResponse = {
  token: string
  expires_at: string
  user: AuthUser
}

export type PublicAnnouncement = {
  id: number
  title: string
  content: string
  pinned: boolean
  published_at?: string | null
}

export type PublicChallengeSummary = {
  id: string
  slug: string
  title: string
  category: string
  points: number
  dynamic: boolean
}

export type PublicAttachment = {
  id: number
  filename: string
  content_type: string
  size_bytes: number
}

export type PublicChallengeDetail = {
  id: number
  slug: string
  title: string
  category: string
  points: number
  difficulty: string
  description: string
  dynamic: boolean
  attachments: PublicAttachment[]
}

export type ScoreboardSolve = {
  challenge_id: number
  challenge_slug: string
  challenge_title: string
  category: string
  difficulty: string
  awarded_points: number
  solved_at: string
}

export type ScoreboardEntry = {
  rank: number
  user_id: number
  username: string
  display_name: string
  score: number
  last_solve_at?: string | null
  solves: ScoreboardSolve[]
}

export type RuntimeInstance = {
  challenge_id: string
  status: string
  access_url?: string
  host_port?: number
  renew_count: number
  started_at: string
  expires_at: string
  terminated_at?: string | null
}

export type SubmissionResult = {
  submission_id: number
  correct: boolean
  solved: boolean
  message: string
  awarded_points: number
  solved_at?: string | null
}

export type UserSubmission = {
  id: number
  challenge_id: number
  challenge_slug: string
  challenge_title: string
  category: string
  correct: boolean
  submitted_at: string
  source_ip: string
}

export type UserSolve = {
  id: number
  challenge_id: number
  challenge_slug: string
  challenge_title: string
  category: string
  submission_id: number
  awarded_points: number
  solved_at: string
}

export type AdminRuntimeConfig = {
  enabled: boolean
  image_name: string
  exposed_protocol: string
  container_port: number
  default_ttl_seconds: number
  max_renew_count: number
  memory_limit_mb: number
  cpu_limit_millicores: number
  env?: Record<string, string>
  command?: string[]
}

export type AdminAttachment = {
  id: number
  filename: string
  content_type: string
  size_bytes: number
}

export type AdminChallengeSummary = {
  id: number
  slug: string
  title: string
  category: string
  points: number
  visible: boolean
  dynamic_enabled: boolean
}

export type AdminChallengeDetail = {
  id: number
  slug: string
  title: string
  category: string
  description: string
  points: number
  difficulty: string
  flag_type: string
  flag_value: string
  visible: boolean
  dynamic_enabled: boolean
  sort_order: number
  attachments: AdminAttachment[]
  runtime_config: AdminRuntimeConfig
}

export type AdminChallengeInput = {
  slug: string
  title: string
  category_slug: string
  description: string
  points: number
  difficulty: string
  flag_type: string
  flag_value: string
  dynamic_enabled: boolean
  visible: boolean
  sort_order: number
  runtime_config?: AdminRuntimeConfig
}

export type AdminAnnouncement = {
  id: number
  title: string
  content: string
  pinned: boolean
  published: boolean
  published_at?: string | null
}

export type AdminSubmission = {
  id: number
  challenge_id: number
  challenge_slug: string
  username: string
  correct: boolean
  submitted_at: string
  source_ip: string
}

export type AdminInstance = {
  id: number
  challenge_id: number
  challenge_slug: string
  username: string
  status: string
  host_port: number
  expires_at: string
  terminated_at?: string | null
  container_id: string
}

export type AdminUser = {
  id: number
  role: string
  username: string
  email: string
  display_name: string
  status: string
  last_login_at?: string | null
  created_at: string
}

export type AdminAuditLog = {
  id: number
  actor_user_id?: number | null
  action: string
  resource_type: string
  resource_id: string
  details?: Record<string, unknown>
  created_at: string
}

async function request<T>(path: string, init?: RequestInit, token?: string): Promise<T> {
  const headers = new Headers(init?.headers)
  if (!headers.has('Content-Type') && init?.body && !(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  const response = await fetch(path, { ...init, headers })
  if (!response.ok) {
    let payload: ApiError | null = null
    try {
      payload = (await response.json()) as ApiError
    } catch {
      payload = null
    }
    const error = new Error(payload?.message ?? `HTTP ${response.status}`)
    ;(error as Error & { code?: string; status?: number }).code = payload?.error
    ;(error as Error & { code?: string; status?: number }).status = response.status
    throw error
  }
  return (await response.json()) as T
}

export const api = {
  register(input: { username: string; email: string; password: string; display_name: string }) {
    return request<AuthResponse>('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(input),
    })
  },
  login(identifier: string, password: string) {
    return request<AuthResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ identifier, password }),
    })
  },
  me(token: string) {
    return request<{ user: AuthUser }>('/api/v1/me', undefined, token)
  },
  announcements() {
    return request<{ items: PublicAnnouncement[] }>('/api/v1/announcements')
  },
  challenges() {
    return request<{ items: PublicChallengeSummary[] }>('/api/v1/challenges')
  },
  challenge(challengeID: string) {
    return request<{ challenge: PublicChallengeDetail }>(`/api/v1/challenges/${challengeID}`)
  },
  scoreboard() {
    return request<{ items: ScoreboardEntry[] }>('/api/v1/scoreboard')
  },
  submitFlag(token: string, challengeID: string, flag: string) {
    return request<SubmissionResult>(
      `/api/v1/challenges/${challengeID}/submissions`,
      {
        method: 'POST',
        body: JSON.stringify({ flag }),
      },
      token,
    )
  },
  getInstance(token: string, challengeID: string) {
    return request<RuntimeInstance>(`/api/v1/challenges/${challengeID}/instances/me`, undefined, token)
  },
  startInstance(token: string, challengeID: string) {
    return request<RuntimeInstance>(`/api/v1/challenges/${challengeID}/instances/me`, { method: 'POST' }, token)
  },
  renewInstance(token: string, challengeID: string) {
    return request<RuntimeInstance>(`/api/v1/challenges/${challengeID}/instances/me/renew`, { method: 'POST' }, token)
  },
  deleteInstance(token: string, challengeID: string) {
    return request<RuntimeInstance>(`/api/v1/challenges/${challengeID}/instances/me`, { method: 'DELETE' }, token)
  },
  mySubmissions(token: string) {
    return request<{ items: UserSubmission[] }>('/api/v1/me/submissions', undefined, token)
  },
  mySolves(token: string) {
    return request<{ items: UserSolve[] }>('/api/v1/me/solves', undefined, token)
  },
  adminChallenges(token: string) {
    return request<{ items: AdminChallengeSummary[] }>('/api/v1/admin/challenges', undefined, token)
  },
  adminChallenge(token: string, challengeID: number) {
    return request<{ challenge: AdminChallengeDetail }>(`/api/v1/admin/challenges/${challengeID}`, undefined, token)
  },
  createAdminChallenge(token: string, payload: AdminChallengeInput) {
    return request<{ challenge: AdminChallengeSummary }>(
      '/api/v1/admin/challenges',
      {
        method: 'POST',
        body: JSON.stringify(payload),
      },
      token,
    )
  },
  updateAdminChallenge(token: string, challengeID: number, payload: AdminChallengeInput) {
    return request<{ challenge: AdminChallengeSummary }>(
      `/api/v1/admin/challenges/${challengeID}`,
      {
        method: 'PATCH',
        body: JSON.stringify(payload),
      },
      token,
    )
  },
  uploadAdminAttachment(token: string, challengeID: number, file: File) {
    const form = new FormData()
    form.append('file', file)
    return request<{ attachment: AdminAttachment }>(
      `/api/v1/admin/challenges/${challengeID}/attachments`,
      {
        method: 'POST',
        body: form,
      },
      token,
    )
  },
  adminAnnouncements(token: string) {
    return request<{ items: AdminAnnouncement[] }>('/api/v1/admin/announcements', undefined, token)
  },
  createAdminAnnouncement(token: string, payload: { title: string; content: string; pinned: boolean; published: boolean }) {
    return request<{ announcement: AdminAnnouncement }>(
      '/api/v1/admin/announcements',
      {
        method: 'POST',
        body: JSON.stringify(payload),
      },
      token,
    )
  },
  deleteAdminAnnouncement(token: string, announcementID: number) {
    return request<{ announcement: AdminAnnouncement }>(`/api/v1/admin/announcements/${announcementID}`, { method: 'DELETE' }, token)
  },
  adminSubmissions(token: string) {
    return request<{ items: AdminSubmission[] }>('/api/v1/admin/submissions', undefined, token)
  },
  adminInstances(token: string) {
    return request<{ items: AdminInstance[] }>('/api/v1/admin/instances', undefined, token)
  },
  terminateAdminInstance(token: string, instanceID: number) {
    return request<{ instance: AdminInstance }>(`/api/v1/admin/instances/${instanceID}/terminate`, { method: 'POST' }, token)
  },
  adminUsers(token: string) {
    return request<{ items: AdminUser[] }>('/api/v1/admin/users', undefined, token)
  },
  updateAdminUser(token: string, userID: number, payload: { role: string; display_name: string; status: string }) {
    return request<{ user: AdminUser }>(
      `/api/v1/admin/users/${userID}`,
      {
        method: 'PATCH',
        body: JSON.stringify(payload),
      },
      token,
    )
  },
  adminAuditLogs(token: string) {
    return request<{ items: AdminAuditLog[] }>('/api/v1/admin/audit-logs', undefined, token)
  },
}
