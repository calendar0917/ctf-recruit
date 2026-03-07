export type Role = "admin" | "player";

export type User = {
  id: string;
  email: string;
  displayName: string;
  role: Role;
  isDisabled?: boolean;
};

export type RegisterRequest = {
  email: string;
  password: string;
  displayName: string;
};

export type LoginRequest = {
  email: string;
  password: string;
};

export type LoginResponse = {
  accessToken: string;
  tokenType: string;
  user: User;
};

export type ChallengeDifficulty = "easy" | "medium" | "hard";
export type ChallengeMode = "static" | "dynamic";

export type Challenge = {
  id: string;
  title: string;
  description: string;
  category: string;
  difficulty: ChallengeDifficulty;
  mode: ChallengeMode;
  runtimeImage?: string;
  runtimeCommand?: string;
  runtimeExposedPort?: number;
  points: number;
  isPublished: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ChallengeListResponse = {
  items: Challenge[];
  limit: number;
  offset: number;
};

export type CreateChallengeRequest = {
  title: string;
  description: string;
  category: string;
  difficulty: ChallengeDifficulty;
  mode: ChallengeMode;
  runtimeImage?: string;
  runtimeCommand?: string;
  runtimeExposedPort?: number;
  points: number;
  flag: string;
  isPublished: boolean;
};

export type UpdateChallengeRequest = Partial<CreateChallengeRequest>;

export type AdminChallengeEditorPayload = CreateChallengeRequest & { id?: string };

export type ChallengeRuntimeField =
  | "runtimeImage"
  | "runtimeCommand"
  | "runtimeExposedPort";

export type ChallengeRuntimeFieldErrors = Partial<Record<ChallengeRuntimeField, string>>;

function normalizeOptionalChallengeText(value: string | undefined): string | undefined {
  if (typeof value !== "string") {
    return undefined;
  }

  const trimmed = value.trim();
  return trimmed ? trimmed : undefined;
}

function applyRuntimeFields(
  payload: Partial<CreateChallengeRequest>,
  value: AdminChallengeEditorPayload,
): void {
  const runtimeImage = normalizeOptionalChallengeText(value.runtimeImage);
  const runtimeCommand = normalizeOptionalChallengeText(value.runtimeCommand);

  if (runtimeImage !== undefined) {
    payload.runtimeImage = runtimeImage;
  }
  if (runtimeCommand !== undefined) {
    payload.runtimeCommand = runtimeCommand;
  }
  if (typeof value.runtimeExposedPort === "number") {
    payload.runtimeExposedPort = value.runtimeExposedPort;
  }
}

export function buildCreateChallengePayload(
  value: AdminChallengeEditorPayload,
): CreateChallengeRequest {
  const payload: CreateChallengeRequest = {
    title: value.title.trim(),
    description: value.description.trim(),
    category: value.category.trim(),
    difficulty: value.difficulty,
    mode: value.mode,
    points: value.points,
    flag: value.flag.trim(),
    isPublished: value.isPublished,
  };

  applyRuntimeFields(payload, value);
  return payload;
}

export function buildUpdateChallengePayload(
  value: AdminChallengeEditorPayload,
): UpdateChallengeRequest {
  const payload: UpdateChallengeRequest = {
    title: value.title.trim(),
    description: value.description.trim(),
    category: value.category.trim(),
    difficulty: value.difficulty,
    mode: value.mode,
    points: value.points,
    isPublished: value.isPublished,
  };

  const trimmedFlag = normalizeOptionalChallengeText(value.flag);
  if (trimmedFlag !== undefined) {
    payload.flag = trimmedFlag;
  }

  applyRuntimeFields(payload, value);
  return payload;
}

export function mapChallengeRuntimeFieldErrors(message: string): ChallengeRuntimeFieldErrors {
  const normalized = message.toLowerCase();
  const errors: ChallengeRuntimeFieldErrors = {};

  if (normalized.includes("runtimeexposedport")) {
    errors.runtimeExposedPort = message;
  }
  if (normalized.includes("runtimeimage")) {
    errors.runtimeImage = message;
  }
  if (normalized.includes("runtimecommand")) {
    errors.runtimeCommand = message;
  }

  return errors;
}

export type Announcement = {
  id: string;
  title: string;
  content: string;
  isPublished: boolean;
  publishedAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type AnnouncementListResponse = {
  items: Announcement[];
  limit: number;
  offset: number;
};

export type CreateAnnouncementRequest = {
  title: string;
  content: string;
  isPublished: boolean;
  publishedAt?: string;
};

export type UpdateAnnouncementRequest = Partial<CreateAnnouncementRequest>;

export type RecruitmentSubmission = {
  id: string;
  userId: string;
  name: string;
  school: string;
  grade: string;
  direction: string;
  contact: string;
  bio: string;
  createdAt: string;
  updatedAt: string;
};

export type RecruitmentSubmissionListResponse = {
  items: RecruitmentSubmission[];
  limit: number;
  offset: number;
};

export type CreateRecruitmentSubmissionRequest = {
  name: string;
  school: string;
  grade: string;
  direction: string;
  contact: string;
  bio: string;
};

export type SubmissionStatus = "correct" | "wrong" | "pending" | "failed";

export type CreateSubmissionRequest = {
  challengeId: string;
  challenge_id?: string;
  flag: string;
};

export type SubmissionResponse = {
  id: string;
  challengeId: string;
  status: SubmissionStatus;
  awardedPoints: number;
  judgeJobId?: string;
  createdAt: string;
};

export type SubmissionListResponse = {
  items: SubmissionResponse[];
  limit: number;
  offset: number;
};

export type ScoreboardItem = {
  rank: number;
  userId: string;
  displayName: string;
  totalPoints: number;
  solvedCount: number;
};

export type ScoreboardResponse = {
  items: ScoreboardItem[];
  limit: number;
  offset: number;
};

export type AuthSession = LoginResponse;

export type AdminUserListResponse = {
  items: User[];
};

export type AdminUpdateUserRequest = {
  role?: Role;
  isDisabled?: boolean;
};

export type ApiErrorResponse<TDetails = unknown> = {
  error?: {
    code?: string;
    message?: string;
    details?: TDetails;
  };
  requestId?: string;
};

export type HttpRetrySource =
  | "details.retryAt"
  | "header.retry-after-seconds"
  | "header.retry-after-date";

export type HttpRetryMetadata = {
  retryAt?: string;
  retryAtMs?: number;
  retryAfterSeconds?: number;
  source?: HttpRetrySource;
  headerValue?: string;
};

export type ChallengeInstanceStatus =
  | "starting"
  | "running"
  | "stopping"
  | "stopped"
  | "expired"
  | "failed"
  | "cooldown";

export type ChallengeInstanceAccessInfo = {
  host: string;
  port: number;
  connectionString?: string;
};

export type ChallengeInstance = {
  id: string;
  userId: string;
  challengeId: string;
  status: ChallengeInstanceStatus;
  containerId?: string;
  accessInfo?: ChallengeInstanceAccessInfo;
  startedAt?: string;
  expiresAt?: string;
  cooldownUntil?: string;
};

export type ChallengeInstanceCooldown = {
  retryAt: string;
};

export type InstanceActiveExistsDetails = {
  activeInstanceId: string;
  activeUserId: string;
  activeChallengeId: string;
  activeStatus: ChallengeInstanceStatus;
  activeStartedAt?: string;
  activeExpiresAt?: string;
};

const challengeInstanceStatusSet: ReadonlySet<string> = new Set([
  "starting",
  "running",
  "stopping",
  "stopped",
  "expired",
  "failed",
  "cooldown",
]);

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isOptionalString(value: unknown): value is string | undefined {
  return value === undefined || typeof value === "string";
}

function isChallengeInstanceStatus(value: unknown): value is ChallengeInstanceStatus {
  return typeof value === "string" && challengeInstanceStatusSet.has(value);
}

export function isInstanceActiveExistsDetails(
  value: unknown,
): value is InstanceActiveExistsDetails {
  if (!isRecord(value)) {
    return false;
  }

  return (
    typeof value.activeInstanceId === "string" &&
    typeof value.activeUserId === "string" &&
    typeof value.activeChallengeId === "string" &&
    isChallengeInstanceStatus(value.activeStatus) &&
    isOptionalString(value.activeStartedAt) &&
    isOptionalString(value.activeExpiresAt)
  );
}

export type MyChallengeInstanceResponse = {
  instance: ChallengeInstance | null;
  cooldown?: ChallengeInstanceCooldown;
};

export type StartChallengeInstanceRequest = {
  challengeId: string;
};

export type StopChallengeInstanceRequest = {
  instanceId?: string;
};
