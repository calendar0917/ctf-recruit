export type Role = "admin" | "player";

export type User = {
  id: string;
  email: string;
  displayName: string;
  role: Role;
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
  points: number;
  flag: string;
  isPublished: boolean;
};

export type UpdateChallengeRequest = Partial<CreateChallengeRequest>;

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

export type ApiErrorResponse = {
  error?: {
    code?: string;
    message?: string;
    details?: unknown;
  };
  requestId?: string;
};
