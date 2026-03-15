import type {
  AdminAnnouncement,
  AdminAuditLog,
  AdminChallengeAuthor,
  AdminChallengeDetail,
  AdminChallengeInput,
  AdminChallengeSummary,
  AdminContestInput,
  AdminInstance,
  AdminSubmission,
  AdminUser,
  ContestResponse,
} from '../../api'

export type AdminData = {
  contest: ContestResponse
  challenges: AdminChallengeSummary[]
  challengeDetail: AdminChallengeDetail | null
  challengeAuthors: AdminChallengeAuthor[]
  announcements: AdminAnnouncement[]
  submissions: AdminSubmission[]
  instances: AdminInstance[]
  users: AdminUser[]
  auditLogs: AdminAuditLog[]
}

export type {
  AdminAnnouncement,
  AdminAuditLog,
  AdminChallengeAuthor,
  AdminChallengeDetail,
  AdminChallengeInput,
  AdminChallengeSummary,
  AdminContestInput,
  AdminInstance,
  AdminSubmission,
  AdminUser,
}

