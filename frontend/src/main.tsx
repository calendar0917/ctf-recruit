import React, { useCallback, useEffect, useMemo, useState } from 'react'
import ReactDOM from 'react-dom/client'
import {
  api,
  type AdminAnnouncement,
  type AdminAuditLog,
  type AdminChallengeDetail,
  type AdminContestInput,
  type ContestInfo,
  type ContestPhase,
  type AdminChallengeAuthor,
  type AdminChallengeInput,
  type AdminChallengeSummary,
  type AdminInstance,
  type AdminSubmission,
  type AdminUser,
  type AuthUser,
  type PublicAnnouncement,
  type PublicChallengeDetail,
  type PublicChallengeSummary,
  type RuntimeInstance,
  type ScoreboardEntry,
  type UserSolve,
  type UserSubmission,
} from './api'
import './styles.css'

const studioMarkUrl = new URL('./assets/yulin-long.svg', import.meta.url).href

type View = 'briefing' | 'board' | 'runtime' | 'scoreboard' | 'admin'
type AuthMode = 'login' | 'register'
type AdminSection = 'contest' | 'challenges' | 'announcements' | 'traffic' | 'users' | 'audit'
type NoticeTone = 'neutral' | 'success' | 'danger'

type Notice = {
  tone: NoticeTone
  text: string
}

type BoardDifficultyFilter = 'all' | 'easy' | 'normal' | 'hard'
type AdminChallengeStatusFilter = 'all' | 'draft' | 'review' | 'ready' | 'published'

type AdminChallengeDraft = {
  slug: string
  title: string
  category_slug: string
  description: string
  points: string
  difficulty: string
  flag_type: string
  flag_value: string
  dynamic_enabled: boolean
  status: string
  visible: boolean
  sort_order: string
  runtime_enabled: boolean
  image_name: string
  exposed_protocol: string
  container_port: string
  default_ttl_seconds: string
  max_renew_count: string
  memory_limit_mb: string
  cpu_limit_millicores: string
  max_active_instances: string
  user_cooldown_seconds: string
  env_text: string
  command_text: string
}

type AnnouncementDraft = {
  title: string
  content: string
  pinned: boolean
  published: boolean
}

type AppError = Error & {
  code?: string
  status?: number
}

const TOKEN_STORAGE_KEY = 'ctf.frontend.token'

const views: Array<{ id: View; label: string }> = [
  { id: 'briefing', label: '总览' },
  { id: 'board', label: '题目' },
  { id: 'runtime', label: '实例' },
  { id: 'scoreboard', label: '排行榜' },
  { id: 'admin', label: '管理' },
]

const categoryOptions = ['web', 'pwn', 'misc', 'crypto', 'reverse']
const difficultyOptions = ['easy', 'normal', 'hard']
const flagTypeOptions = ['static', 'case_insensitive', 'regex']
const boardDifficultyOptions: BoardDifficultyFilter[] = ['all', 'easy', 'normal', 'hard']
const userRoleOptions = ['player', 'author', 'ops', 'admin']
const userStatusOptions = ['active', 'disabled']
const challengeStatusOptions = ['draft', 'review', 'ready', 'published']

function describeError(error: unknown, fallback: string): string {
  const typed = error as AppError | undefined
  return typed?.message?.trim() ? typed.message : fallback
}

function isUnauthorized(error: unknown): boolean {
  const typed = error as AppError | undefined
  return typed?.status === 401
}

function readErrorCode(error: unknown): string {
  const typed = error as AppError | undefined
  return typed?.code ?? ''
}

function formatDateTime(value?: string | null): string {
  if (!value) {
    return '未记录'
  }
  try {
    return new Intl.DateTimeFormat('zh-CN', {
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    }).format(new Date(value))
  } catch {
    return value
  }
}

function formatBytes(value: number): string {
  if (value < 1024) {
    return `${value} B`
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`
  }
  return `${(value / (1024 * 1024)).toFixed(1)} MB`
}

function formatChallengeStatus(value?: string | null): string {
  if (!value) {
    return '草稿'
  }
  if (value === 'draft') {
    return '草稿'
  }
  if (value === 'review') {
    return '待审核'
  }
  if (value === 'ready') {
    return '待发布'
  }
  if (value === 'published') {
    return '已发布'
  }
  return value
}

function challengeStatusBadgeClass(value?: string | null): string {
  if (value === 'published') {
    return 'badge badge-solid'
  }
  if (value === 'ready') {
    return 'badge badge-accent'
  }
  return 'badge'
}

function formatDifficultyLabel(value?: string | null): string {
  if (!value) {
    return '未标注'
  }
  if (value === 'easy') {
    return '简单'
  }
  if (value === 'normal') {
    return '标准'
  }
  if (value === 'hard') {
    return '困难'
  }
  return value
}

function getDifficultyRank(value?: string | null): number {
  if (value === 'easy') {
    return 0
  }
  if (value === 'normal') {
    return 1
  }
  if (value === 'hard') {
    return 2
  }
  return 3
}

function formatRemaining(expiresAt?: string | null): string {
  if (!expiresAt) {
    return '未设置'
  }
  const remainingMs = new Date(expiresAt).getTime() - Date.now()
  if (Number.isNaN(remainingMs)) {
    return '未知'
  }
  if (remainingMs <= 0) {
    return '已过期'
  }
  const minutes = Math.floor(remainingMs / 60000)
  if (minutes < 1) {
    return '少于 1 分钟'
  }
  if (minutes < 60) {
    return `${minutes} 分钟`
  }
  const hours = Math.floor(minutes / 60)
  const restMinutes = minutes % 60
  return restMinutes > 0 ? `${hours} 小时 ${restMinutes} 分钟` : `${hours} 小时`
}

function formatEnvText(env?: Record<string, string>): string {
  if (!env) {
    return ''
  }
  return Object.entries(env)
    .map(([key, value]) => `${key}=${value}`)
    .join('\n')
}

function formatCommandText(command?: string[]): string {
  return (command ?? []).join('\n')
}

function summarizeText(value?: string | null, maxLength = 180): string {
  if (!value) {
    return ''
  }
  const normalized = value.replace(/\s+/g, ' ').trim()
  if (normalized.length <= maxLength) {
    return normalized
  }
  return `${normalized.slice(0, maxLength).trim()}...`
}

function parseEnvText(input: string): Record<string, string> {
  const result: Record<string, string> = {}
  for (const line of input.split('\n')) {
    const trimmed = line.trim()
    if (!trimmed) {
      continue
    }
    const separator = trimmed.indexOf('=')
    if (separator <= 0) {
      continue
    }
    const key = trimmed.slice(0, separator).trim()
    const value = trimmed.slice(separator + 1).trim()
    if (!key) {
      continue
    }
    result[key] = value
  }
  return result
}

function parseCommandText(input: string): string[] {
  return input
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
}

function parseInteger(input: string, fallback = 0): number {
  const parsed = Number.parseInt(input, 10)
  return Number.isFinite(parsed) ? parsed : fallback
}

function createBlankChallengeDraft(): AdminChallengeDraft {
  return {
    slug: '',
    title: '',
    category_slug: 'web',
    description: '',
    points: '100',
    difficulty: 'normal',
    flag_type: 'static',
    flag_value: '',
    dynamic_enabled: false,
    status: 'draft',
    visible: false,
    sort_order: '10',
    runtime_enabled: false,
    image_name: '',
    exposed_protocol: 'http',
    container_port: '80',
    default_ttl_seconds: '1800',
    max_renew_count: '1',
    memory_limit_mb: '256',
    cpu_limit_millicores: '500',
    max_active_instances: '0',
    user_cooldown_seconds: '0',
    env_text: '',
    command_text: '',
  }
}

function challengeDraftFromDetail(detail: AdminChallengeDetail): AdminChallengeDraft {
  return {
    slug: detail.slug,
    title: detail.title,
    category_slug: detail.category,
    description: detail.description,
    points: String(detail.points),
    difficulty: detail.difficulty,
    flag_type: detail.flag_type,
    flag_value: detail.flag_value,
    dynamic_enabled: detail.dynamic_enabled,
    status: detail.status,
    visible: detail.visible,
    sort_order: String(detail.sort_order),
    runtime_enabled: detail.runtime_config.enabled || detail.dynamic_enabled,
    image_name: detail.runtime_config.image_name ?? '',
    exposed_protocol: detail.runtime_config.exposed_protocol || 'http',
    container_port: String(detail.runtime_config.container_port || 80),
    default_ttl_seconds: String(detail.runtime_config.default_ttl_seconds || 1800),
    max_renew_count: String(detail.runtime_config.max_renew_count || 0),
    memory_limit_mb: String(detail.runtime_config.memory_limit_mb || 256),
    cpu_limit_millicores: String(detail.runtime_config.cpu_limit_millicores || 500),
    max_active_instances: String(detail.runtime_config.max_active_instances || 0),
    user_cooldown_seconds: String(detail.runtime_config.user_cooldown_seconds || 0),
    env_text: formatEnvText(detail.runtime_config.env),
    command_text: formatCommandText(detail.runtime_config.command),
  }
}

function buildChallengePayload(draft: AdminChallengeDraft): AdminChallengeInput {
  const hasRuntimeConfig =
    draft.dynamic_enabled ||
    draft.runtime_enabled ||
    draft.image_name.trim() !== '' ||
    draft.env_text.trim() !== '' ||
    draft.command_text.trim() !== ''

  return {
    slug: draft.slug.trim(),
    title: draft.title.trim(),
    category_slug: draft.category_slug,
    description: draft.description.trim(),
    points: parseInteger(draft.points, 100),
    difficulty: draft.difficulty,
    flag_type: draft.flag_type,
    flag_value: draft.flag_value.trim(),
    dynamic_enabled: draft.dynamic_enabled,
    status: draft.status,
    visible: draft.status === 'published',
    sort_order: parseInteger(draft.sort_order, 10),
    runtime_config: hasRuntimeConfig
      ? {
          enabled: draft.runtime_enabled,
          image_name: draft.image_name.trim(),
          exposed_protocol: draft.exposed_protocol.trim() || 'http',
          container_port: parseInteger(draft.container_port, 80),
          default_ttl_seconds: parseInteger(draft.default_ttl_seconds, 1800),
          max_renew_count: parseInteger(draft.max_renew_count, 0),
          memory_limit_mb: parseInteger(draft.memory_limit_mb, 256),
          cpu_limit_millicores: parseInteger(draft.cpu_limit_millicores, 500),
          max_active_instances: parseInteger(draft.max_active_instances, 0),
          user_cooldown_seconds: parseInteger(draft.user_cooldown_seconds, 0),
          env: parseEnvText(draft.env_text),
          command: parseCommandText(draft.command_text),
        }
      : undefined,
  }
}

function Panel(props: {
  eyebrow?: string
  title: string
  subtitle?: string
  actions?: React.ReactNode
  className?: string
  children: React.ReactNode
}): React.JSX.Element {
  return (
    <section className={`panel ${props.className ?? ''}`.trim()}>
      <div className="panel-head">
        <div>
          {props.eyebrow ? <p className="eyebrow">{props.eyebrow}</p> : null}
          <h2>{props.title}</h2>
          {props.subtitle ? <p className="panel-subtitle">{props.subtitle}</p> : null}
        </div>
        {props.actions ? <div className="panel-actions">{props.actions}</div> : null}
      </div>
      {props.children}
    </section>
  )
}

function formatContestStatus(value?: string | null): string {
  if (value === 'upcoming') return '未开始'
  if (value === 'running') return '进行中'
  if (value === 'frozen') return '冻结中'
  if (value === 'ended') return '已结束'
  return '准备中'
}

function formatCapabilityState(value?: boolean): string {
  return value ? '开放' : '关闭'
}

function buildContestNotice(phase: ContestPhase | null): Notice | null {
  if (!phase) return null
  if (phase.status === 'running') return { tone: 'success', text: phase.message }
  if (phase.status === 'draft' || phase.status === 'upcoming') return { tone: 'neutral', text: phase.message }
  return { tone: 'danger', text: phase.message }
}

function NoticeBanner({ notice }: { notice: Notice | null }): React.JSX.Element | null {
  if (!notice) {
    return null
  }
  return (
    <div aria-live="polite" className={`notice notice-${notice.tone}`}>
      {notice.text}
    </div>
  )
}

function App(): React.JSX.Element {
  const [view, setView] = useState<View>('briefing')
  const [authMode, setAuthMode] = useState<AuthMode>('login')
  const [token, setToken] = useState<string>(() => window.localStorage.getItem(TOKEN_STORAGE_KEY) ?? '')
  const [sessionLoading, setSessionLoading] = useState(Boolean(token))
  const [authBusy, setAuthBusy] = useState(false)
  const [authNotice, setAuthNotice] = useState<Notice | null>(null)
  const [authUser, setAuthUser] = useState<AuthUser | null>(null)
  const [loginForm, setLoginForm] = useState({ identifier: '', password: '' })
  const [registerForm, setRegisterForm] = useState({
    username: '',
    email: '',
    display_name: '',
    password: '',
  })

  const [contestInfo, setContestInfo] = useState<ContestInfo | null>(null)
  const [contestPhase, setContestPhase] = useState<ContestPhase | null>(null)
  const [announcements, setAnnouncements] = useState<PublicAnnouncement[]>([])
  const [challengeList, setChallengeList] = useState<PublicChallengeSummary[]>([])
  const [scoreboard, setScoreboard] = useState<ScoreboardEntry[]>([])
  const [expandedRanks, setExpandedRanks] = useState<Record<number, boolean>>({})
  const [publicLoading, setPublicLoading] = useState(true)
  const [publicNotice, setPublicNotice] = useState<Notice | null>(null)

  const [challengeSearch, setChallengeSearch] = useState('')
  const [boardDifficultyFilter, setBoardDifficultyFilter] = useState<BoardDifficultyFilter>('all')
  const [collapsedCategories, setCollapsedCategories] = useState<Record<string, boolean>>({})
  const [selectedChallengeId, setSelectedChallengeId] = useState('')
  const [challengeDetail, setChallengeDetail] = useState<PublicChallengeDetail | null>(null)
  const [challengeDetailLoading, setChallengeDetailLoading] = useState(false)
  const [challengeDetailNotice, setChallengeDetailNotice] = useState<Notice | null>(null)

  const [mySubmissions, setMySubmissions] = useState<UserSubmission[]>([])
  const [mySolves, setMySolves] = useState<UserSolve[]>([])
  const [historyLoading, setHistoryLoading] = useState(false)
  const [historyNotice, setHistoryNotice] = useState<Notice | null>(null)

  const [flagInput, setFlagInput] = useState('')
  const [submitBusy, setSubmitBusy] = useState(false)
  const [submitNotice, setSubmitNotice] = useState<Notice | null>(null)

  const [runtimeInstance, setRuntimeInstance] = useState<RuntimeInstance | null>(null)
  const [runtimeLoading, setRuntimeLoading] = useState(false)
  const [runtimeNotice, setRuntimeNotice] = useState<Notice | null>(null)

  const [adminSection, setAdminSection] = useState<AdminSection>('contest')
  const [adminContestNotice, setAdminContestNotice] = useState<Notice | null>(null)
  const [adminContestLoading, setAdminContestLoading] = useState(false)
  const [adminContestSubmitting, setAdminContestSubmitting] = useState(false)
  const [adminContestDraft, setAdminContestDraft] = useState<AdminContestInput>({ status: 'draft', starts_at: '', ends_at: '' })
  const [adminChallenges, setAdminChallenges] = useState<AdminChallengeSummary[]>([])
  const [adminChallengesLoading, setAdminChallengesLoading] = useState(false)
  const [adminChallengesNotice, setAdminChallengesNotice] = useState<Notice | null>(null)
  const [adminChallengeStatusFilter, setAdminChallengeStatusFilter] = useState<AdminChallengeStatusFilter>('all')
  const [selectedAdminChallenge, setSelectedAdminChallenge] = useState<number | 'new' | null>(null)
  const [adminChallengeDetail, setAdminChallengeDetail] = useState<AdminChallengeDetail | null>(null)
  const [adminChallengeDraft, setAdminChallengeDraft] = useState<AdminChallengeDraft>(createBlankChallengeDraft())
  const [adminChallengeDetailLoading, setAdminChallengeDetailLoading] = useState(false)
  const [adminChallengeNotice, setAdminChallengeNotice] = useState<Notice | null>(null)
  const [adminChallengeSubmitting, setAdminChallengeSubmitting] = useState(false)
  const [adminChallengeAuthorIDs, setAdminChallengeAuthorIDs] = useState<number[]>([])
  const [adminChallengeAuthorSubmitting, setAdminChallengeAuthorSubmitting] = useState(false)
  const [attachmentFile, setAttachmentFile] = useState<File | null>(null)
  const [attachmentUploading, setAttachmentUploading] = useState(false)

  const [adminAnnouncements, setAdminAnnouncements] = useState<AdminAnnouncement[]>([])
  const [adminAnnouncementsLoading, setAdminAnnouncementsLoading] = useState(false)
  const [adminAnnouncementsNotice, setAdminAnnouncementsNotice] = useState<Notice | null>(null)
  const [announcementDraft, setAnnouncementDraft] = useState<AnnouncementDraft>({
    title: '',
    content: '',
    pinned: false,
    published: false,
  })
  const [announcementSubmitting, setAnnouncementSubmitting] = useState(false)
  const [deletingAnnouncementId, setDeletingAnnouncementId] = useState<number | null>(null)

  const [adminSubmissions, setAdminSubmissions] = useState<AdminSubmission[]>([])
  const [adminSubmissionsLoading, setAdminSubmissionsLoading] = useState(false)
  const [adminSubmissionsNotice, setAdminSubmissionsNotice] = useState<Notice | null>(null)

  const [adminInstances, setAdminInstances] = useState<AdminInstance[]>([])
  const [adminInstancesLoading, setAdminInstancesLoading] = useState(false)
  const [adminInstancesNotice, setAdminInstancesNotice] = useState<Notice | null>(null)
  const [terminatingInstanceId, setTerminatingInstanceId] = useState<number | null>(null)

  const [adminUsers, setAdminUsers] = useState<AdminUser[]>([])
  const [adminUsersLoading, setAdminUsersLoading] = useState(false)
  const [adminUsersNotice, setAdminUsersNotice] = useState<Notice | null>(null)
  const [selectedAdminUserId, setSelectedAdminUserId] = useState<number | null>(null)
  const [adminUserDraft, setAdminUserDraft] = useState({ role: 'player', display_name: '', status: 'active' })
  const [adminUserSubmitting, setAdminUserSubmitting] = useState(false)

  const [adminAuditLogs, setAdminAuditLogs] = useState<AdminAuditLog[]>([])
  const [adminAuditLoading, setAdminAuditLoading] = useState(false)
  const [adminAuditNotice, setAdminAuditNotice] = useState<Notice | null>(null)

  const canAccessAdmin = authUser?.role === 'admin' || authUser?.role === 'ops' || authUser?.role === 'author'
  const canWriteChallenges = authUser?.role === 'admin' || authUser?.role === 'author'
  const canManageChallengeAuthors = authUser?.role === 'admin'
  const canWriteAnnouncements = authUser?.role === 'admin'
  const canUploadAttachments = authUser?.role === 'admin' || authUser?.role === 'ops' || authUser?.role === 'author'
  const canManageUsers = authUser?.role === 'admin'
  const canTerminateInstances = authUser?.role === 'admin' || authUser?.role === 'ops'

  const clearSession = useCallback((message?: string) => {
    setToken('')
    setAuthUser(null)
    setMySubmissions([])
    setMySolves([])
    setRuntimeInstance(null)
    if (message) {
      setAuthNotice({ tone: 'danger', text: message })
    }
  }, [])

  const guardedError = useCallback(
    (error: unknown, fallback: string): string => {
      if (isUnauthorized(error)) {
        clearSession('登录态已失效，请重新登录。')
        return '登录态已失效，请重新登录。'
      }
      return describeError(error, fallback)
    },
    [clearSession],
  )

  const loadContestState = useCallback(async () => {
    const response = await api.contest()
    setContestInfo(response.contest)
    setContestPhase(response.phase)
  }, [])

  const loadScoreboard = useCallback(async () => {
    const response = await api.scoreboard()
    setScoreboard(response.items)
  }, [])

  const loadPublicData = useCallback(async () => {
    setPublicLoading(true)
    setPublicNotice(null)
    try {
      const contestResponse = await api.contest()
      setContestInfo(contestResponse.contest)
      setContestPhase(contestResponse.phase)

      const tasks: Promise<unknown>[] = []
      if (contestResponse.phase.announcement_visible) {
        tasks.push(api.announcements().then((response) => setAnnouncements(response.items)))
      } else {
        setAnnouncements([])
      }
      if (contestResponse.phase.challenge_list_visible) {
        tasks.push(api.challenges().then((response) => setChallengeList(response.items)))
      } else {
        setChallengeList([])
        setChallengeDetail(null)
      }
      if (contestResponse.phase.scoreboard_visible) {
        tasks.push(api.scoreboard().then((response) => setScoreboard(response.items)))
      } else {
        setScoreboard([])
      }
      await Promise.all(tasks)
    } catch (error) {
      setPublicNotice({ tone: 'danger', text: describeError(error, '公开数据加载失败。') })
    } finally {
      setPublicLoading(false)
    }
  }, [])

  const loadAdminContest = useCallback(async () => {
    if (!token) {
      return
    }
    setAdminContestLoading(true)
    setAdminContestNotice(null)
    try {
      const response = await api.adminContest(token)
      setContestInfo(response.contest)
      setContestPhase(response.phase)
      setAdminContestDraft({
        status: response.contest.status,
        starts_at: response.contest.starts_at ?? '',
        ends_at: response.contest.ends_at ?? '',
      })
    } catch (error) {
      setAdminContestNotice({ tone: 'danger', text: guardedError(error, '比赛状态加载失败。') })
    } finally {
      setAdminContestLoading(false)
    }
  }, [guardedError, token])

  const loadPersonalData = useCallback(async () => {
    if (!token) {
      setMySubmissions([])
      setMySolves([])
      setHistoryNotice(null)
      return
    }
    setHistoryLoading(true)
    setHistoryNotice(null)
    try {
      const [submissionResponse, solveResponse] = await Promise.all([api.mySubmissions(token), api.mySolves(token)])
      setMySubmissions(submissionResponse.items)
      setMySolves(solveResponse.items)
    } catch (error) {
      setHistoryNotice({ tone: 'danger', text: guardedError(error, '个人历史加载失败。') })
    } finally {
      setHistoryLoading(false)
    }
  }, [guardedError, token])

  const loadAdminChallenges = useCallback(async () => {
    if (!token) {
      return
    }
    setAdminChallengesLoading(true)
    setAdminChallengesNotice(null)
    try {
      const response = await api.adminChallenges(token)
      setAdminChallenges(response.items)
      setSelectedAdminChallenge((current) => {
        if (current === 'new') {
          return current
        }
        if (typeof current === 'number' && response.items.some((item) => item.id === current)) {
          return current
        }
        if (response.items.length > 0) {
          return response.items[0].id
        }
        return canWriteChallenges ? 'new' : null
      })
    } catch (error) {
      setAdminChallengesNotice({ tone: 'danger', text: guardedError(error, '题目列表加载失败。') })
    } finally {
      setAdminChallengesLoading(false)
    }
  }, [canWriteChallenges, guardedError, token])

  const loadAdminChallengeDetail = useCallback(
    async (challengeId: number) => {
      if (!token) {
        return
      }
      setAdminChallengeDetailLoading(true)
      setAdminChallengeNotice(null)
      try {
        const response = await api.adminChallenge(token, challengeId)
        setAdminChallengeDetail(response.challenge)
        setAdminChallengeDraft(challengeDraftFromDetail(response.challenge))
        setAdminChallengeAuthorIDs(response.challenge.authors.map((item) => item.user_id))
      } catch (error) {
        setAdminChallengeDetail(null)
        setAdminChallengeNotice({ tone: 'danger', text: guardedError(error, '题目详情加载失败。') })
      } finally {
        setAdminChallengeDetailLoading(false)
      }
    },
    [guardedError, token],
  )

  const loadAdminAnnouncements = useCallback(async () => {
    if (!token) {
      return
    }
    setAdminAnnouncementsLoading(true)
    setAdminAnnouncementsNotice(null)
    try {
      const response = await api.adminAnnouncements(token)
      setAdminAnnouncements(response.items)
    } catch (error) {
      setAdminAnnouncementsNotice({ tone: 'danger', text: guardedError(error, '公告加载失败。') })
    } finally {
      setAdminAnnouncementsLoading(false)
    }
  }, [guardedError, token])

  const loadAdminSubmissions = useCallback(async () => {
    if (!token) {
      return
    }
    setAdminSubmissionsLoading(true)
    setAdminSubmissionsNotice(null)
    try {
      const response = await api.adminSubmissions(token)
      setAdminSubmissions(response.items)
    } catch (error) {
      setAdminSubmissionsNotice({ tone: 'danger', text: guardedError(error, '提交记录加载失败。') })
    } finally {
      setAdminSubmissionsLoading(false)
    }
  }, [guardedError, token])

  const loadAdminInstances = useCallback(async () => {
    if (!token) {
      return
    }
    setAdminInstancesLoading(true)
    setAdminInstancesNotice(null)
    try {
      const response = await api.adminInstances(token)
      setAdminInstances(response.items)
    } catch (error) {
      setAdminInstancesNotice({ tone: 'danger', text: guardedError(error, '实例记录加载失败。') })
    } finally {
      setAdminInstancesLoading(false)
    }
  }, [guardedError, token])

  const loadAdminUsers = useCallback(async () => {
    if (!token) {
      return
    }
    setAdminUsersLoading(true)
    setAdminUsersNotice(null)
    try {
      const response = await api.adminUsers(token)
      setAdminUsers(response.items)
      setSelectedAdminUserId((current) => {
        if (current && response.items.some((item) => item.id === current)) {
          return current
        }
        return response.items[0]?.id ?? null
      })
    } catch (error) {
      setAdminUsersNotice({ tone: 'danger', text: guardedError(error, '用户列表加载失败。') })
    } finally {
      setAdminUsersLoading(false)
    }
  }, [guardedError, token])

  const loadAdminAuditLogs = useCallback(async () => {
    if (!token) {
      return
    }
    setAdminAuditLoading(true)
    setAdminAuditNotice(null)
    try {
      const response = await api.adminAuditLogs(token)
      setAdminAuditLogs(response.items)
    } catch (error) {
      setAdminAuditNotice({ tone: 'danger', text: guardedError(error, '审计日志加载失败。') })
    } finally {
      setAdminAuditLoading(false)
    }
  }, [guardedError, token])

  useEffect(() => {
    window.localStorage.setItem(TOKEN_STORAGE_KEY, token)
    if (!token) {
      window.localStorage.removeItem(TOKEN_STORAGE_KEY)
    }
  }, [token])

  useEffect(() => {
    void loadPublicData()
  }, [loadPublicData])

  useEffect(() => {
    if (!token) {
      setSessionLoading(false)
      setAuthUser(null)
      return
    }
    let active = true
    setSessionLoading(true)
    void api
      .me(token)
      .then((response) => {
        if (!active) {
          return
        }
        setAuthUser(response.user)
      })
      .catch((error) => {
        if (!active) {
          return
        }
        clearSession(describeError(error, '登录态已失效，请重新登录。'))
      })
      .finally(() => {
        if (active) {
          setSessionLoading(false)
        }
      })
    return () => {
      active = false
    }
  }, [clearSession, token])

  useEffect(() => {
    if (!authUser || !token) {
      setMySubmissions([])
      setMySolves([])
      return
    }
    void loadPersonalData()
  }, [authUser, loadPersonalData, token])

  useEffect(() => {
    if (challengeList.length === 0) {
      return
    }
    setCollapsedCategories((current) => {
      const next = { ...current }
      for (const item of challengeList) {
        if (next[item.category] === undefined) {
          next[item.category] = false
        }
      }
      return next
    })
  }, [challengeList])

  useEffect(() => {
    if (challengeList.length === 0) {
      return
    }
    if (!selectedChallengeId || !challengeList.some((item) => item.id === selectedChallengeId)) {
      setSelectedChallengeId(challengeList[0].id)
    }
  }, [challengeList, selectedChallengeId])

  useEffect(() => {
    if (!selectedChallengeId) {
      setChallengeDetail(null)
      return
    }
    let active = true
    setChallengeDetailLoading(true)
    setChallengeDetailNotice(null)
    void api
      .challenge(selectedChallengeId)
      .then((response) => {
        if (!active) {
          return
        }
        setChallengeDetail(response.challenge)
      })
      .catch((error) => {
        if (!active) {
          return
        }
        setChallengeDetail(null)
        setChallengeDetailNotice({ tone: 'danger', text: describeError(error, '题目详情加载失败。') })
      })
      .finally(() => {
        if (active) {
          setChallengeDetailLoading(false)
        }
      })
    return () => {
      active = false
    }
  }, [selectedChallengeId])

  const selectedChallengeSummary = useMemo(
    () => challengeList.find((item) => item.id === selectedChallengeId) ?? null,
    [challengeList, selectedChallengeId],
  )

  useEffect(() => {
    if (!selectedChallengeSummary) {
      setRuntimeInstance(null)
      setRuntimeNotice(null)
      return
    }
    if (!token) {
      setRuntimeInstance(null)
      setRuntimeNotice({ tone: 'neutral', text: '登录后可以拉起或续期动态实例。' })
      return
    }
    if (!selectedChallengeSummary.dynamic) {
      setRuntimeInstance(null)
      setRuntimeNotice({ tone: 'neutral', text: '当前题目不需要动态实例。' })
      return
    }

    let active = true
    setRuntimeLoading(true)
    setRuntimeNotice(null)
    void api
      .getInstance(token, selectedChallengeSummary.id)
      .then((instance) => {
        if (!active) {
          return
        }
        setRuntimeInstance(instance)
      })
      .catch((error) => {
        if (!active) {
          return
        }
        const code = readErrorCode(error)
        setRuntimeInstance(null)
        if (code === 'instance_not_found') {
          setRuntimeNotice({ tone: 'neutral', text: '当前尚未创建实例，可以直接启动。' })
          return
        }
        if (code === 'runtime_config_missing') {
          setRuntimeNotice({ tone: 'danger', text: '题目已标记为动态题，但运行配置还未补齐。' })
          return
        }
        if (code === 'challenge_not_dynamic') {
          setRuntimeNotice({ tone: 'neutral', text: '当前题目不需要动态实例。' })
          return
        }
        setRuntimeNotice({ tone: 'danger', text: guardedError(error, '实例状态加载失败。') })
      })
      .finally(() => {
        if (active) {
          setRuntimeLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [guardedError, selectedChallengeSummary, token])

  const availableAdminSections = useMemo(() => {
    const sections: Array<{ id: AdminSection; label: string; note: string }> = []
    if (!canAccessAdmin) {
      return sections
    }
    if (authUser?.role === 'admin' || authUser?.role === 'ops') {
      sections.push({ id: 'contest', label: '比赛', note: 'Lifecycle' })
    }
    sections.push({ id: 'challenges', label: '题目', note: 'Catalog' })
    if (authUser?.role === 'admin' || authUser?.role === 'ops') {
      sections.push({ id: 'announcements', label: '公告', note: 'Broadcast' })
      sections.push({ id: 'traffic', label: '流量', note: 'Ops Feed' })
    }
    if (canManageUsers) {
      sections.push({ id: 'users', label: '用户', note: 'Identity' })
    }
    sections.push({ id: 'audit', label: '审计', note: 'Audit Trail' })
    return sections
  }, [authUser?.role, canAccessAdmin, canManageUsers])

  useEffect(() => {
    if (view === 'admin' && !canAccessAdmin) {
      setView('briefing')
    }
  }, [canAccessAdmin, view])

  useEffect(() => {
    if (!canAccessAdmin) {
      return
    }
    if (availableAdminSections.length === 0) {
      return
    }
    if (!availableAdminSections.some((item) => item.id === adminSection)) {
      setAdminSection(availableAdminSections[0].id)
    }
  }, [adminSection, availableAdminSections, canAccessAdmin])

  useEffect(() => {
    if (!(canAccessAdmin && token && view === 'admin')) {
      return
    }
    if (adminSection === 'contest') {
      void loadAdminContest()
    }
    if (adminSection === 'challenges') {
      void loadAdminChallenges()
      if (canManageChallengeAuthors) {
        void loadAdminUsers()
      }
    }
    if (adminSection === 'announcements') {
      void loadAdminAnnouncements()
    }
    if (adminSection === 'traffic') {
      void loadAdminSubmissions()
      void loadAdminInstances()
    }
    if (adminSection === 'users' && canManageUsers) {
      void loadAdminUsers()
    }
    if (adminSection === 'audit') {
      void loadAdminAuditLogs()
    }
  }, [
    adminSection,
    canAccessAdmin,
    canManageChallengeAuthors,
    canManageUsers,
    loadAdminAnnouncements,
    loadAdminContest,
    loadAdminAuditLogs,
    loadAdminChallenges,
    loadAdminInstances,
    loadAdminSubmissions,
    loadAdminUsers,
    token,
    view,
  ])

  useEffect(() => {
    if (!(canAccessAdmin && token && view === 'admin' && adminSection === 'challenges')) {
      return
    }
    if (selectedAdminChallenge === 'new') {
      setAdminChallengeDetail(null)
      setAdminChallengeDraft(createBlankChallengeDraft())
      setAdminChallengeAuthorIDs([])
      return
    }
    if (typeof selectedAdminChallenge !== 'number') {
      return
    }
    void loadAdminChallengeDetail(selectedAdminChallenge)
  }, [adminSection, canAccessAdmin, loadAdminChallengeDetail, selectedAdminChallenge, token, view])

  const filteredAdminChallenges = useMemo(() => {
    return adminChallenges.filter((item) => adminChallengeStatusFilter === 'all' || item.status === adminChallengeStatusFilter)
  }, [adminChallengeStatusFilter, adminChallenges])

  useEffect(() => {
    if (!(canAccessAdmin && view === 'admin' && adminSection === 'challenges')) {
      return
    }
    if (selectedAdminChallenge === 'new') {
      return
    }
    if (typeof selectedAdminChallenge === 'number' && filteredAdminChallenges.some((item) => item.id === selectedAdminChallenge)) {
      return
    }
    if (filteredAdminChallenges.length > 0) {
      setSelectedAdminChallenge(filteredAdminChallenges[0].id)
      return
    }
    setSelectedAdminChallenge(canWriteChallenges ? 'new' : null)
  }, [adminSection, canAccessAdmin, canWriteChallenges, filteredAdminChallenges, selectedAdminChallenge, view])

  const selectedAdminUser = useMemo(
    () => adminUsers.find((item) => item.id === selectedAdminUserId) ?? null,
    [adminUsers, selectedAdminUserId],
  )

  useEffect(() => {
    if (!selectedAdminUser) {
      return
    }
    setAdminUserDraft({
      role: selectedAdminUser.role,
      display_name: selectedAdminUser.display_name,
      status: selectedAdminUser.status,
    })
  }, [selectedAdminUser])

  const solvedChallengeIds = useMemo(() => new Set(mySolves.map((item) => String(item.challenge_id))), [mySolves])

  const filteredChallengeGroups = useMemo(() => {
    const groups = new Map<string, PublicChallengeSummary[]>()
    const needle = challengeSearch.trim().toLowerCase()
    const visibleItems = challengeList.filter((item) => {
      if (
        needle &&
        !item.title.toLowerCase().includes(needle) &&
        !item.category.toLowerCase().includes(needle) &&
        !item.slug.toLowerCase().includes(needle)
      ) {
        return false
      }
      if (boardDifficultyFilter !== 'all' && item.difficulty !== boardDifficultyFilter) {
        return false
      }
      return true
    })

    const sortedItems = [...visibleItems].sort((left, right) =>
      left.points - right.points || getDifficultyRank(left.difficulty) - getDifficultyRank(right.difficulty) || left.title.localeCompare(right.title),
    )

    for (const item of sortedItems) {
      const current = groups.get(item.category) ?? []
      current.push(item)
      groups.set(item.category, current)
    }
    return Array.from(groups.entries()).map(([category, items]) => ({ category, items }))
  }, [boardDifficultyFilter, challengeList, challengeSearch])

  const totalScore = useMemo(() => mySolves.reduce((sum, item) => sum + item.awarded_points, 0), [mySolves])

  const recentSolvedChallengeIds = useMemo(() => mySolves.slice(0, 4).map((item) => String(item.challenge_id)), [mySolves])

  const selectedChallengeSolve = useMemo(
    () => mySolves.find((item) => String(item.challenge_id) === selectedChallengeId) ?? null,
    [mySolves, selectedChallengeId],
  )

  const dynamicChallenges = useMemo(() => challengeList.filter((item) => item.dynamic), [challengeList])
  const visibleBoardChallengeCount = filteredChallengeGroups.reduce((count, group) => count + group.items.length, 0)

  async function handleLoginSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setAuthBusy(true)
    setAuthNotice(null)
    try {
      const response = await api.login(loginForm.identifier.trim(), loginForm.password)
      setToken(response.token)
      setAuthUser(response.user)
      setAuthMode('login')
      setAuthNotice({ tone: 'success', text: `已登录为 ${response.user.display_name || response.user.username}` })
      setView('board')
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'login_rate_limited') {
        setAuthNotice({ tone: 'danger', text: '登录尝试过于频繁，请稍后再试。' })
      } else {
        setAuthNotice({ tone: 'danger', text: describeError(error, '登录失败。') })
      }
    } finally {
      setAuthBusy(false)
    }
  }

  async function handleRegisterSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setAuthBusy(true)
    setAuthNotice(null)
    try {
      const response = await api.register({
        username: registerForm.username.trim(),
        email: registerForm.email.trim(),
        display_name: registerForm.display_name.trim(),
        password: registerForm.password,
      })
      setToken(response.token)
      setAuthUser(response.user)
      setAuthNotice({ tone: 'success', text: '注册成功，已自动登录。' })
      setView('board')
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'register_rate_limited') {
        setAuthNotice({ tone: 'danger', text: '注册尝试过于频繁，请稍后再试。' })
      } else {
        setAuthNotice({ tone: 'danger', text: describeError(error, '注册失败。') })
      }
    } finally {
      setAuthBusy(false)
    }
  }

  function handleLogout() {
    clearSession('已退出当前账号。')
    setView('board')
  }

  function focusFlagInput() {
    const field = document.getElementById('flag-input') as HTMLInputElement | null
    field?.focus()
    field?.select()
  }

  async function handleSubmitFlag(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token || !selectedChallengeSummary) {
      setSubmitNotice({ tone: 'neutral', text: '需要先登录才能提交 Flag。' })
      return
    }
    if (!flagInput.trim()) {
      setSubmitNotice({ tone: 'neutral', text: '请输入待提交的 Flag。' })
      return
    }
    setSubmitBusy(true)
    setSubmitNotice(null)
    try {
      const result = await api.submitFlag(token, selectedChallengeSummary.id, flagInput.trim())
      setFlagInput('')
      setSubmitNotice({
        tone: result.correct ? 'success' : 'danger',
        text: result.correct
          ? result.solved
            ? `判题通过，获得 ${result.awarded_points} 分。`
            : '判题通过，但该题已解出。'
          : result.message || 'Flag 不正确。',
      })
      await Promise.all([loadPersonalData(), loadScoreboard()])
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'submission_rate_limited') {
        setSubmitNotice({ tone: 'danger', text: '提交过于频繁，请稍后再试。' })
      } else {
        setSubmitNotice({ tone: 'danger', text: guardedError(error, '提交失败。') })
      }
    } finally {
      setSubmitBusy(false)
    }
  }

	async function handleRuntimeAction(action: 'start' | 'renew' | 'delete') {
		if (!token || !selectedChallengeSummary) {
			setRuntimeNotice({ tone: 'neutral', text: '需要先登录才能操作动态实例。' })
			return
		}
		setRuntimeLoading(true)
		setRuntimeNotice(null)
		try {
			if (action === 'start') {
				const instance = await api.startInstance(token, selectedChallengeSummary.id)
				setRuntimeInstance(instance)
				setRuntimeNotice({ tone: 'success', text: '实例已就绪，可以直接访问。' })
			}
			if (action === 'renew') {
				const instance = await api.renewInstance(token, selectedChallengeSummary.id)
				setRuntimeInstance(instance)
				setRuntimeNotice({ tone: 'success', text: '实例已续期。' })
			}
			if (action === 'delete') {
				await api.deleteInstance(token, selectedChallengeSummary.id)
				setRuntimeInstance(null)
				setRuntimeNotice({ tone: 'success', text: '实例已回收。' })
			}
		} catch (error) {
			const code = readErrorCode(error)
			if (code === 'instance_not_found') {
				setRuntimeInstance(null)
				setRuntimeNotice({ tone: 'neutral', text: '当前没有活动实例。' })
			} else if (code === 'instance_capacity_reached') {
				setRuntimeNotice({ tone: 'danger', text: '当前题目实例配额已满，请稍后再试。' })
			} else if (code === 'instance_cooldown_active') {
				setRuntimeNotice({ tone: 'danger', text: '刚刚创建过实例，冷却中，请稍后再试。' })
			} else if (code === 'instance_port_exhausted') {
				setRuntimeNotice({ tone: 'danger', text: '当前实例端口资源已耗尽，请稍后再试或联系管理员。' })
			} else if (code === 'instance_renew_limit_reached') {
				setRuntimeNotice({ tone: 'danger', text: '实例已达到最大续期次数。' })
			} else if (code === 'runtime_config_missing') {
				setRuntimeNotice({ tone: 'danger', text: '运行配置不完整，无法拉起实例。' })
			} else {
				setRuntimeNotice({ tone: 'danger', text: guardedError(error, '实例操作失败。') })
			}
		} finally {
			setRuntimeLoading(false)
		}
	}

  async function handleSaveAdminChallenge(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token || !canWriteChallenges) {
      setAdminChallengeNotice({ tone: 'neutral', text: '当前账号没有题目写权限。' })
      return
    }
    setAdminChallengeSubmitting(true)
    setAdminChallengeNotice(null)
    try {
      const payload = buildChallengePayload(adminChallengeDraft)
      if (selectedAdminChallenge === 'new') {
        const response = await api.createAdminChallenge(token, payload)
        setSelectedAdminChallenge(response.challenge.id)
        setAdminChallengeNotice({ tone: 'success', text: '题目已创建。' })
        await loadAdminChallenges()
        await loadAdminChallengeDetail(response.challenge.id)
      } else if (typeof selectedAdminChallenge === 'number') {
        await api.updateAdminChallenge(token, selectedAdminChallenge, payload)
        setAdminChallengeNotice({ tone: 'success', text: '题目配置已更新。' })
        await loadAdminChallenges()
        await loadAdminChallengeDetail(selectedAdminChallenge)
      }
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'admin_rate_limited') {
        setAdminChallengeNotice({ tone: 'danger', text: '后台写操作过于频繁，请稍后再试。' })
      } else {
        setAdminChallengeNotice({ tone: 'danger', text: guardedError(error, '题目保存失败。') })
      }
    } finally {
      setAdminChallengeSubmitting(false)
    }
  }

  async function handleSaveChallengeAuthors() {
    if (!token || !canManageChallengeAuthors || typeof selectedAdminChallenge !== 'number') {
      setAdminChallengeNotice({ tone: 'neutral', text: '当前账号没有负责人管理权限，或尚未选中题目。' })
      return
    }
    setAdminChallengeAuthorSubmitting(true)
    setAdminChallengeNotice(null)
    try {
      const response = await api.updateAdminChallengeAuthors(token, selectedAdminChallenge, adminChallengeAuthorIDs)
      setAdminChallengeDetail((current) => (current ? { ...current, authors: response.items } : current))
      setAdminChallengeNotice({ tone: 'success', text: '题目负责人已更新。' })
      await loadAdminChallengeDetail(selectedAdminChallenge)
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'admin_rate_limited') {
        setAdminChallengeNotice({ tone: 'danger', text: '后台写操作过于频繁，请稍后再试。' })
      } else {
        setAdminChallengeNotice({ tone: 'danger', text: guardedError(error, '题目负责人更新失败。') })
      }
    } finally {
      setAdminChallengeAuthorSubmitting(false)
    }
  }

  async function handleUploadAttachment() {
    if (!token || !attachmentFile || typeof selectedAdminChallenge !== 'number') {
      setAdminChallengeNotice({ tone: 'neutral', text: '需要先选择题目和附件文件。' })
      return
    }
    if (!canUploadAttachments) {
      setAdminChallengeNotice({ tone: 'neutral', text: '当前账号没有附件上传权限。' })
      return
    }
    setAttachmentUploading(true)
    setAdminChallengeNotice(null)
    try {
      await api.uploadAdminAttachment(token, selectedAdminChallenge, attachmentFile)
      setAttachmentFile(null)
      setAdminChallengeNotice({ tone: 'success', text: '附件上传完成。' })
      await loadAdminChallengeDetail(selectedAdminChallenge)
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'admin_rate_limited') {
        setAdminChallengeNotice({ tone: 'danger', text: '后台写操作过于频繁，请稍后再试。' })
      } else {
        setAdminChallengeNotice({ tone: 'danger', text: guardedError(error, '附件上传失败。') })
      }
    } finally {
      setAttachmentUploading(false)
    }
  }

  async function handleCreateAnnouncement(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token || !canWriteAnnouncements) {
      setAdminAnnouncementsNotice({ tone: 'neutral', text: '当前账号没有公告发布权限。' })
      return
    }
    setAnnouncementSubmitting(true)
    setAdminAnnouncementsNotice(null)
    try {
      await api.createAdminAnnouncement(token, announcementDraft)
      setAnnouncementDraft({ title: '', content: '', pinned: false, published: false })
      setAdminAnnouncementsNotice({ tone: 'success', text: '公告已写入。' })
      await Promise.all([loadAdminAnnouncements(), loadPublicData()])
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'admin_rate_limited') {
        setAdminAnnouncementsNotice({ tone: 'danger', text: '后台写操作过于频繁，请稍后再试。' })
      } else {
        setAdminAnnouncementsNotice({ tone: 'danger', text: guardedError(error, '公告创建失败。') })
      }
    } finally {
      setAnnouncementSubmitting(false)
    }
  }

  async function handleDeleteAnnouncement(announcementId: number) {
    if (!token || !canWriteAnnouncements) {
      setAdminAnnouncementsNotice({ tone: 'neutral', text: '当前账号没有公告删除权限。' })
      return
    }
    setDeletingAnnouncementId(announcementId)
    setAdminAnnouncementsNotice(null)
    try {
      await api.deleteAdminAnnouncement(token, announcementId)
      setAdminAnnouncementsNotice({ tone: 'success', text: `公告 #${announcementId} 已移除。` })
      await Promise.all([loadAdminAnnouncements(), loadPublicData()])
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'admin_rate_limited') {
        setAdminAnnouncementsNotice({ tone: 'danger', text: '后台写操作过于频繁，请稍后再试。' })
      } else {
        setAdminAnnouncementsNotice({ tone: 'danger', text: guardedError(error, '公告删除失败。') })
      }
    } finally {
      setDeletingAnnouncementId(null)
    }
  }

  async function handleTerminateInstance(instanceId: number) {
    if (!token || !canTerminateInstances) {
      setAdminInstancesNotice({ tone: 'neutral', text: '当前账号没有实例终止权限。' })
      return
    }
    setTerminatingInstanceId(instanceId)
    setAdminInstancesNotice(null)
    try {
      await api.terminateAdminInstance(token, instanceId)
      setAdminInstancesNotice({ tone: 'success', text: `实例 #${instanceId} 已终止。` })
      await loadAdminInstances()
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'admin_rate_limited') {
        setAdminInstancesNotice({ tone: 'danger', text: '后台写操作过于频繁，请稍后再试。' })
      } else {
        setAdminInstancesNotice({ tone: 'danger', text: guardedError(error, '实例终止失败。') })
      }
    } finally {
      setTerminatingInstanceId(null)
    }
  }

  async function handleSaveUser(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token || !canManageUsers || !selectedAdminUserId) {
      setAdminUsersNotice({ tone: 'neutral', text: '请选择用户并确认当前账号具备权限。' })
      return
    }
    setAdminUserSubmitting(true)
    setAdminUsersNotice(null)
    try {
      await api.updateAdminUser(token, selectedAdminUserId, adminUserDraft)
      setAdminUsersNotice({ tone: 'success', text: '用户资料已更新。' })
      await loadAdminUsers()
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'admin_rate_limited') {
        setAdminUsersNotice({ tone: 'danger', text: '后台写操作过于频繁，请稍后再试。' })
      } else {
        setAdminUsersNotice({ tone: 'danger', text: guardedError(error, '用户更新失败。') })
      }
    } finally {
      setAdminUserSubmitting(false)
    }
  }

  const latestAnnouncements = announcements.slice(0, 2)
  const contestNotice = buildContestNotice(contestPhase)
  const contestStatusLabel = formatContestStatus(contestInfo?.status)
  const contestWindowLabel = contestInfo
    ? `${formatDateTime(contestInfo.starts_at)} - ${formatDateTime(contestInfo.ends_at)}`
    : '比赛时间待配置'
  const contestCapabilityCards = [
    { label: '题库', value: formatCapabilityState(contestPhase?.challenge_list_visible), note: '公开题目' },
    { label: '提交', value: formatCapabilityState(contestPhase?.submission_allowed), note: 'Flag 判题' },
    { label: '实例', value: formatCapabilityState(contestPhase?.runtime_allowed), note: '动态环境' },
    { label: '排行', value: formatCapabilityState(contestPhase?.scoreboard_visible), note: '公开榜单' },
  ]
  const personalScoreboardEntry = authUser ? scoreboard.find((item) => item.user_id === authUser.id) ?? null : null
  const personalRankLabel = personalScoreboardEntry ? `第 ${personalScoreboardEntry.rank} 名` : authUser ? '尚未上榜' : '登录后追踪排名'

  async function handleSaveContest(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token || !canAccessAdmin || authUser?.role !== 'admin') {
      setAdminContestNotice({ tone: 'neutral', text: '当前账号没有比赛配置写权限。' })
      return
    }
    setAdminContestSubmitting(true)
    setAdminContestNotice(null)
    try {
      await api.updateAdminContest(token, adminContestDraft)
      setAdminContestNotice({ tone: 'success', text: '比赛状态已更新。' })
      await Promise.all([loadAdminContest(), loadPublicData()])
    } catch (error) {
      const code = readErrorCode(error)
      if (code === 'admin_rate_limited') {
        setAdminContestNotice({ tone: 'danger', text: '后台写操作过于频繁，请稍后再试。' })
      } else {
        setAdminContestNotice({ tone: 'danger', text: guardedError(error, '比赛状态更新失败。') })
      }
    } finally {
      setAdminContestSubmitting(false)
    }
  }

  function renderAdminContest(): React.JSX.Element {
    const canWriteContest = authUser?.role === 'admin'
    return (
      <div className="two-column-layout admin-column-layout">
        <Panel eyebrow="Contest State" title={contestInfo?.title ?? '比赛状态'} subtitle="第一版生命周期控制采用单场比赛手动切换。">
          <NoticeBanner notice={adminContestNotice ?? contestNotice} />
          {adminContestLoading ? <div className="empty-state">正在读取比赛状态…</div> : null}
          {!adminContestLoading && contestInfo ? (
            <div className="detail-list compact-list">
              <div className="detail-row">
                <span>当前阶段</span>
                <strong>{formatContestStatus(contestInfo.status)}</strong>
              </div>
              <div className="detail-row">
                <span>开始时间</span>
                <strong>{formatDateTime(contestInfo.starts_at)}</strong>
              </div>
              <div className="detail-row">
                <span>结束时间</span>
                <strong>{formatDateTime(contestInfo.ends_at)}</strong>
              </div>
              <div className="detail-row">
                <span>公开说明</span>
                <strong>{contestPhase?.message ?? '未配置'}</strong>
              </div>
            </div>
          ) : null}
        </Panel>

        <Panel eyebrow="Lifecycle Editor" title="阶段编辑" subtitle="用于控制公开内容、提交和动态实例的开放窗口。">
          {!canWriteContest ? <div className="empty-state">当前账号只有只读权限。</div> : null}
          {canWriteContest ? (
            <form className="form-grid" onSubmit={handleSaveContest}>
              <label className="field">
                <span>状态</span>
                <select value={adminContestDraft.status} onChange={(event) => setAdminContestDraft((current) => ({ ...current, status: event.target.value }))}>
                  <option value="draft">draft</option>
                  <option value="upcoming">upcoming</option>
                  <option value="running">running</option>
                  <option value="frozen">frozen</option>
                  <option value="ended">ended</option>
                </select>
              </label>
              <label className="field">
                <span>开始时间 RFC3339</span>
                <input value={adminContestDraft.starts_at} onChange={(event) => setAdminContestDraft((current) => ({ ...current, starts_at: event.target.value }))} placeholder="2026-03-09T09:00:00Z" />
              </label>
              <label className="field">
                <span>结束时间 RFC3339</span>
                <input value={adminContestDraft.ends_at} onChange={(event) => setAdminContestDraft((current) => ({ ...current, ends_at: event.target.value }))} placeholder="2026-03-09T15:00:00Z" />
              </label>
              <div className="form-footer wide-field">
                <button className="primary-button" disabled={adminContestSubmitting} type="submit">
                  {adminContestSubmitting ? '保存中…' : '保存比赛状态'}
                </button>
              </div>
            </form>
          ) : null}
        </Panel>
      </div>
    )
  }

  function renderBriefing(): React.JSX.Element {
    return (
      <div className="view-stack briefing-view player-focused-view">
        <NoticeBanner notice={contestNotice ?? publicNotice} />

        <section className="player-hero panel panel-hero page-enter page-enter-1">
          <div className="player-hero-stack">
            <div className="player-hero-copy">
              <p className="eyebrow">Solve First</p>
              <h2>{contestInfo?.title ?? '单场招新赛平台'}</h2>
              <p className="panel-subtitle">
                {contestInfo?.description?.trim() || '把视线收束到做题本身：选题、读题、提交，动态题再按需拉起实例。'}
              </p>
            </div>

            <div className="hero-stat-row compact-stat-row" aria-label="Contest summary">
              <div className="hero-stat-card hero-stat-card-strong">
                <span>比赛阶段</span>
                <strong>{contestStatusLabel}</strong>
                <small>{contestPhase?.message ?? '等待后台配置比赛状态说明。'}</small>
              </div>
              <div className="hero-stat-card">
                <span>公开题目</span>
                <strong>{challengeList.length}</strong>
                <small>{dynamicChallenges.length} 道支持动态实例</small>
              </div>
              <div className="hero-stat-card">
                <span>{authUser ? '我的排名' : '当前入口'}</span>
                <strong>{authUser ? personalRankLabel : '游客'}</strong>
                <small>{authUser ? `累计 ${totalScore} 分` : '登录后提交 Flag 并记录进度'}</small>
              </div>
            </div>

            <div className="inline-actions wrap-actions">
              <button className="primary-button" onClick={() => setView('board')} type="button">
                {authUser ? '继续做题' : '进入题目区'}
              </button>
              <button className="ghost-button" onClick={() => setView('scoreboard')} type="button">
                查看排行榜
              </button>
              {canAccessAdmin ? (
                <button className="ghost-button" onClick={() => setView('admin')} type="button">
                  进入管理区
                </button>
              ) : null}
            </div>
          </div>
        </section>

        <div className="player-entry-grid page-enter page-enter-2">
          <Panel
            eyebrow={authUser ? '继续比赛' : '进入比赛'}
            title={authUser ? `${authUser.display_name || authUser.username} 的做题入口` : '登录或注册后开始做题'}
            subtitle={authUser ? '保留最常用的信息和动作，不在首页堆叠过多入口。' : '完成登录后即可提交 Flag、记录解题和使用动态实例。'}
          >
            <NoticeBanner notice={authNotice} />
            {sessionLoading ? <div className="empty-state">正在恢复登录态…</div> : null}
            {!sessionLoading && !authUser ? (
              <div className="auth-layout">
                <div className="tab-strip compact-strip">
                  <button
                    className={authMode === 'login' ? 'tab-pill active' : 'tab-pill'}
                    onClick={() => setAuthMode('login')}
                    type="button"
                  >
                    登录
                  </button>
                  <button
                    className={authMode === 'register' ? 'tab-pill active' : 'tab-pill'}
                    onClick={() => setAuthMode('register')}
                    type="button"
                  >
                    注册
                  </button>
                </div>

                {authMode === 'login' ? (
                  <form className="form-grid single-column" onSubmit={handleLoginSubmit}>
                    <label className="field">
                      <span>邮箱或用户名</span>
                      <input
                        onChange={(event) => setLoginForm((current) => ({ ...current, identifier: event.target.value }))}
                        placeholder="输入邮箱或用户名"
                        value={loginForm.identifier}
                      />
                    </label>
                    <label className="field">
                      <span>密码</span>
                      <input
                        onChange={(event) => setLoginForm((current) => ({ ...current, password: event.target.value }))}
                        placeholder="输入登录密码"
                        type="password"
                        value={loginForm.password}
                      />
                    </label>
                    <button className="primary-button" disabled={authBusy} type="submit">
                      {authBusy ? '登录中…' : '进入比赛'}
                    </button>
                  </form>
                ) : (
                  <form className="form-grid single-column" onSubmit={handleRegisterSubmit}>
                    <label className="field">
                      <span>用户名</span>
                      <input
                        onChange={(event) => setRegisterForm((current) => ({ ...current, username: event.target.value }))}
                        value={registerForm.username}
                      />
                    </label>
                    <label className="field">
                      <span>邮箱</span>
                      <input
                        onChange={(event) => setRegisterForm((current) => ({ ...current, email: event.target.value }))}
                        type="email"
                        value={registerForm.email}
                      />
                    </label>
                    <label className="field">
                      <span>显示名</span>
                      <input
                        onChange={(event) => setRegisterForm((current) => ({ ...current, display_name: event.target.value }))}
                        value={registerForm.display_name}
                      />
                    </label>
                    <label className="field">
                      <span>密码</span>
                      <input
                        onChange={(event) => setRegisterForm((current) => ({ ...current, password: event.target.value }))}
                        type="password"
                        value={registerForm.password}
                      />
                    </label>
                    <button className="primary-button" disabled={authBusy} type="submit">
                      {authBusy ? '注册中…' : '注册并进入'}
                    </button>
                  </form>
                )}
              </div>
            ) : null}

            {!sessionLoading && authUser ? (
              <div className="player-summary-stack">
                <div className="mini-grid player-mini-grid">
                  <div className="summary-card">
                    <span>累计得分</span>
                    <strong>{totalScore}</strong>
                  </div>
                  <div className="summary-card">
                    <span>解题数</span>
                    <strong>{mySolves.length}</strong>
                  </div>
                  <div className="summary-card">
                    <span>我的排名</span>
                    <strong>{personalRankLabel}</strong>
                  </div>
                  <div className="summary-card">
                    <span>最后登录</span>
                    <strong>{formatDateTime(authUser.last_login_at)}</strong>
                  </div>
                </div>

                <div className="detail-list compact-list">
                  <div className="detail-row">
                    <span>比赛时间</span>
                    <strong>{contestWindowLabel}</strong>
                  </div>
                  <div className="detail-row">
                    <span>当前身份</span>
                    <strong>{authUser.role}</strong>
                  </div>
                </div>

                <div className="inline-actions wrap-actions">
                  <button className="primary-button" onClick={() => setView('board')} type="button">
                    去做题
                  </button>
                  <button className="ghost-button" onClick={() => setView('scoreboard')} type="button">
                    看排行榜
                  </button>
                </div>
              </div>
            ) : null}
          </Panel>

          <Panel eyebrow="赛务信息" title={contestStatusLabel} subtitle={contestPhase?.message ?? '只保留和参赛直接相关的赛务信息。'} className="player-side-panel">
            <div className="detail-list compact-list">
              <div className="detail-row">
                <span>赛程</span>
                <strong>{contestWindowLabel}</strong>
              </div>
              <div className="detail-row">
                <span>公告数</span>
                <strong>{announcements.length}</strong>
              </div>
            </div>

            <div className="capability-grid compact-capability-grid" aria-label="Contest capabilities">
              {contestCapabilityCards.map((item) => (
                <div className="capability-card" key={item.label}>
                  <span>{item.label}</span>
                  <strong className="capability-value">{item.value}</strong>
                  <small>{item.note}</small>
                </div>
              ))}
            </div>

            <div className="subpanel compact-subpanel announcement-subpanel">
              <h3>最新公告</h3>
              {publicLoading ? <div className="empty-state small">正在读取公告…</div> : null}
              {!publicLoading && latestAnnouncements.length === 0 ? <div className="empty-state small">当前还没有已发布公告。</div> : null}
              {!publicLoading && latestAnnouncements.length > 0 ? (
                <div className="compact-list announcement-compact-list">
                  {latestAnnouncements.map((item) => (
                    <div className="row-card announcement-row" key={item.id}>
                      <strong>{item.title}</strong>
                      <span>{formatDateTime(item.published_at)}</span>
                      <p>{summarizeText(item.content, 72) || '暂无摘要。'}</p>
                    </div>
                  ))}
                </div>
              ) : null}
            </div>

            {canAccessAdmin ? (
              <div className="admin-launch-card compact-admin-launch">
                <div>
                  <span>后台权限已开放</span>
                  <strong>题目、公告和运维入口仍保留在管理区。</strong>
                </div>
                <button className="ghost-button" onClick={() => setView('admin')} type="button">
                  打开后台
                </button>
              </div>
            ) : null}
          </Panel>
        </div>
      </div>
    )
  }

  function renderBoard(): React.JSX.Element {
    const showBoardRuntimePanel = Boolean(selectedChallengeSummary?.dynamic)

    return (
      <div className="view-stack board-view player-focused-board">
        <NoticeBanner notice={contestNotice ?? publicNotice} />

        <section className="board-summary panel panel-hero board-hero-compact page-enter page-enter-1">
          <div className="board-summary-grid">
            <div>
              <p className="eyebrow">Solve Mode</p>
              <h2>{selectedChallengeSummary?.title ?? '集中做题'}</h2>
              <p className="panel-subtitle">
                {selectedChallengeSummary
                  ? `${selectedChallengeSummary.category} · ${selectedChallengeSummary.points} pts · ${formatDifficultyLabel(selectedChallengeSummary.difficulty)}${selectedChallengeSummary.dynamic ? ' · 支持动态实例' : ''}`
                  : '保持题目列表、题面和提交区三段式结构，把主要注意力收束到当前题目。'}
              </p>
            </div>
            <div className="board-summary-metrics compact-stat-row">
              <div className="summary-card">
                <span>公开题目</span>
                <strong>{visibleBoardChallengeCount}</strong>
              </div>
              <div className="summary-card">
                <span>已解题</span>
                <strong>{mySolves.length}</strong>
              </div>
              <div className="summary-card">
                <span>当前排名</span>
                <strong>{personalRankLabel}</strong>
              </div>
            </div>
          </div>
        </section>

        <div className="board-shell player-board-shell page-enter page-enter-2">
          <Panel eyebrow="题目列表" title="先选题，再进入题面" subtitle="仅保留检索和难度筛选。" className="rail-panel challenge-rail-panel player-side-panel">
            <div className="board-list-toolbar">
              <label className="field compact-field board-search-field">
                <span>检索</span>
                <input
                  onChange={(event) => setChallengeSearch(event.target.value)}
                  placeholder="welcome / web / crypto"
                  value={challengeSearch}
                />
              </label>

              <label className="field compact-field board-select-field">
                <span>难度</span>
                <select onChange={(event) => setBoardDifficultyFilter(event.target.value as BoardDifficultyFilter)} value={boardDifficultyFilter}>
                  {boardDifficultyOptions.map((option) => (
                    <option key={option} value={option}>
                      {option === 'all' ? '全部难度' : formatDifficultyLabel(option)}
                    </option>
                  ))}
                </select>
              </label>

              <div className="board-filter-meta">
                <span>{visibleBoardChallengeCount} 题</span>
                <span>{boardDifficultyFilter === 'all' ? '全部难度' : formatDifficultyLabel(boardDifficultyFilter)}</span>
              </div>

              {authUser && recentSolvedChallengeIds.length > 0 ? (
                <div className="recent-progress-strip">
                  <span>最近解出</span>
                  <div className="badge-row wrap-actions">
                    {recentSolvedChallengeIds.map((challengeId) => {
                      const challenge = challengeList.find((item) => item.id === challengeId)
                      if (!challenge) {
                        return null
                      }
                      return (
                        <button className="badge recent-progress-badge" key={challenge.id} onClick={() => setSelectedChallengeId(challenge.id)} type="button">
                          {challenge.title}
                        </button>
                      )
                    })}
                  </div>
                </div>
              ) : null}
            </div>

            <div className="accordion-stack board-accordion-stack">
              {filteredChallengeGroups.map((group) => (
                <section className="accordion-block" key={group.category}>
                  <button
                    className="accordion-head"
                    onClick={() =>
                      setCollapsedCategories((current) => ({
                        ...current,
                        [group.category]: !current[group.category],
                      }))
                    }
                    type="button"
                  >
                    <div>
                      <strong>{group.category}</strong>
                      <span>{group.items.length} 题</span>
                    </div>
                    <small className={collapsedCategories[group.category] ? 'accordion-toggle collapsed' : 'accordion-toggle'}>
                      {collapsedCategories[group.category] ? '展开' : '收起'}
                    </small>
                  </button>
                  <div className={collapsedCategories[group.category] ? 'accordion-body collapsed' : 'accordion-body'}>
                    <div className="challenge-card-list">
                      {group.items.map((item) => (
                        <button
                          className={selectedChallengeId === item.id ? `challenge-card challenge-card-player difficulty-${item.difficulty} active` : `challenge-card challenge-card-player difficulty-${item.difficulty}`}
                          key={item.id}
                          onClick={() => setSelectedChallengeId(item.id)}
                          type="button"
                        >
                          <div className="challenge-card-head">
                            <strong>{item.title}</strong>
                            <span>{item.points} pts</span>
                          </div>
                          <div className="challenge-card-subline">
                            <span>{item.category}</span>
                            <small>{selectedChallengeId === item.id ? '当前题目' : solvedChallengeIds.has(item.id) ? '已完成' : '可开始'}</small>
                          </div>
                          <div className="badge-row board-badge-row">
                            <span className={`badge difficulty-pill difficulty-${item.difficulty}`}>{formatDifficultyLabel(item.difficulty)}</span>
                            {item.dynamic ? <span className="badge badge-accent">动态题</span> : null}
                            {solvedChallengeIds.has(item.id) ? <span className="badge badge-solid">已解出</span> : null}
                          </div>
                        </button>
                      ))}
                    </div>
                  </div>
                </section>
              ))}
              {filteredChallengeGroups.length === 0 ? <div className="empty-state">没有匹配的题目。</div> : null}
            </div>
          </Panel>

          <div className="board-main-column player-board-column">
            <Panel
              key={selectedChallengeId || 'empty'}
              eyebrow="题面"
              className={selectedChallengeId ? `challenge-detail-panel challenge-workspace-panel challenge-shift challenge-${selectedChallengeId}` : 'challenge-detail-panel challenge-workspace-panel'}
              title={challengeDetail?.title ?? selectedChallengeSummary?.title ?? '选择题目'}
              subtitle={
                challengeDetail
                  ? `${challengeDetail.category} · ${challengeDetail.points} pts · ${formatDifficultyLabel(challengeDetail.difficulty)}`
                  : selectedChallengeSummary
                    ? `${selectedChallengeSummary.category} · ${selectedChallengeSummary.points} pts · ${formatDifficultyLabel(selectedChallengeSummary.difficulty)}`
                    : '从左侧题目列表中选择一题，右侧只保留题面与提交。'
              }
              actions={
                showBoardRuntimePanel ? (
                  <button className="ghost-button" onClick={() => setView('runtime')} type="button">
                    实例控制
                  </button>
                ) : undefined
              }
            >
              <NoticeBanner notice={challengeDetailNotice} />
              {challengeDetailLoading ? <div className="empty-state">正在读取题面…</div> : null}
              {!challengeDetailLoading && !challengeDetail ? <div className="empty-state">从左侧选择一题进入题目详情。</div> : null}
              {!challengeDetailLoading && challengeDetail ? (
                <div className="detail-stack board-main-stack">
                  <div className="challenge-meta-grid player-meta-grid">
                    <div className="meta-chip">
                      <span>分类</span>
                      <strong>{challengeDetail.category}</strong>
                    </div>
                    <div className="meta-chip">
                      <span>分值</span>
                      <strong>{challengeDetail.points} pts</strong>
                    </div>
                    <div className="meta-chip">
                      <span>难度</span>
                      <strong>{formatDifficultyLabel(challengeDetail.difficulty)}</strong>
                    </div>
                    <div className="meta-chip">
                      <span>状态</span>
                      <strong>{selectedChallengeSolve ? '已解出' : selectedChallengeSummary ? '待提交' : '先选题'}</strong>
                    </div>
                  </div>

                  <article className="statement-card statement-card-feature player-statement-card">
                    <div className="statement-topline">
                      <div>
                        <span className="section-tag">题面</span>
                        <h3>任务说明</h3>
                      </div>
                      <div className="badge-row wrap-actions">
                        {challengeDetail.dynamic ? <span className="badge badge-accent">动态题</span> : null}
                        {selectedChallengeSolve ? <span className="badge badge-solid">已解出</span> : null}
                      </div>
                    </div>
                    <p className="statement-text">{challengeDetail.description}</p>
                  </article>

                  {challengeDetail.attachments.length > 0 ? (
                    <div className="subpanel compact-subpanel mission-panel">
                      <h3>附件</h3>
                      <div className="attachment-list">
                        {challengeDetail.attachments.map((attachment) => (
                          <a
                            className="attachment-row"
                            href={`/api/v1/challenges/${challengeDetail.id}/attachments/${attachment.id}`}
                            key={attachment.id}
                            rel="noreferrer"
                            target="_blank"
                          >
                            <strong>{attachment.filename}</strong>
                            <span>
                              {attachment.content_type} · {formatBytes(attachment.size_bytes)}
                            </span>
                          </a>
                        ))}
                      </div>
                    </div>
                  ) : null}
                </div>
              ) : null}
            </Panel>

            <div className={showBoardRuntimePanel ? 'board-action-grid player-action-grid' : 'board-action-grid single-action-grid'}>
              <Panel eyebrow="提交" title="直接提交 Flag" subtitle="把当前题目的输入、反馈和结果放在同一个区块中。" className="primary-action-panel player-primary-action-panel">
                <NoticeBanner notice={submitNotice} />
                <div className="primary-action-layout">
                  <div className="primary-action-intro player-action-intro">
                    <div className="primary-action-copy">
                      <span className="section-tag">当前题目</span>
                      <h3>{selectedChallengeSummary?.title ?? '尚未选择题目'}</h3>
                      <p className="hint-text">
                        {selectedChallengeSummary
                          ? selectedChallengeSolve
                            ? '这道题已经通过，仍然可以继续查看题面或打开实例。'
                            : '读完题面后直接在这里提交，无需再切换到其他区域。'
                          : '先从左侧题目列表选择一题，再在这里提交 Flag。'}
                      </p>
                    </div>
                    <div className="primary-action-status">
                      <span>当前状态</span>
                      <strong>{selectedChallengeSolve ? '已解出' : authUser ? '待提交' : '需登录'}</strong>
                    </div>
                  </div>

                  {!authUser ? <div className="empty-state">登录后可提交 Flag。</div> : null}
                  {authUser ? (
                    <form className="form-grid single-column action-form" onSubmit={handleSubmitFlag}>
                      <label className="field">
                        <span>Flag</span>
                        <input
                          id="flag-input" inputMode="text" autoCapitalize="none" autoCorrect="off" spellCheck={false}
                          onChange={(event) => setFlagInput(event.target.value)}
                          placeholder="flag{...}"
                          value={flagInput}
                        />
                      </label>
                      <div className="inline-actions wrap-actions">
                        <button className="primary-button" disabled={submitBusy || !selectedChallengeSummary} type="submit">
                          {submitBusy ? '判题中…' : '提交 Flag'}
                        </button>
                        {showBoardRuntimePanel ? (
                          <button className="ghost-button" onClick={() => setView('runtime')} type="button">
                            管理实例
                          </button>
                        ) : null}
                        <button className="ghost-button" onClick={focusFlagInput} type="button">
                          聚焦输入框
                        </button>
                      </div>
                    </form>
                  ) : null}
                </div>
              </Panel>

              {showBoardRuntimePanel ? (
                <Panel eyebrow="实例" className="runtime-control-panel compact-runtime-panel" title="动态题实例" subtitle="只保留实例状态和操作，不再暴露接口说明。">
                  <NoticeBanner notice={runtimeNotice} />
                  <div className="runtime-quick-card">
                    <div className="runtime-metrics-grid compact-runtime-metrics">
                      <div className="runtime-metric">
                        <span>状态</span>
                        <strong>{runtimeInstance?.status ?? 'idle'}</strong>
                      </div>
                      <div className="runtime-metric">
                        <span>剩余时间</span>
                        <strong>{runtimeInstance ? formatRemaining(runtimeInstance.expires_at) : '未启动'}</strong>
                      </div>
                      <div className="runtime-metric">
                        <span>续期次数</span>
                        <strong>{runtimeInstance?.renew_count ?? 0}</strong>
                      </div>
                    </div>

                    <div className="detail-row runtime-availability-row">
                      <span>实例入口</span>
                      <strong>{runtimeInstance?.access_url ? '已分配，可直接打开' : '启动实例后生成'}</strong>
                    </div>

                    <div className="inline-actions wrap-actions">
                      <button
                        className="primary-button"
                        disabled={runtimeLoading || !selectedChallengeSummary}
                        onClick={() => void handleRuntimeAction('start')}
                        type="button"
                      >
                        {runtimeLoading ? '处理中…' : runtimeInstance ? '重取实例' : '启动实例'}
                      </button>
                      <button
                        className="ghost-button"
                        disabled={runtimeLoading || !runtimeInstance}
                        onClick={() => void handleRuntimeAction('renew')}
                        type="button"
                      >
                        续期实例
                      </button>
                      <button
                        className="ghost-button danger-button"
                        disabled={runtimeLoading || !runtimeInstance}
                        onClick={() => void handleRuntimeAction('delete')}
                        type="button"
                      >
                        回收实例
                      </button>
                      {runtimeInstance?.access_url ? (
                        <a className="link-button" href={runtimeInstance.access_url} rel="noreferrer" target="_blank">
                          打开实例
                        </a>
                      ) : null}
                    </div>
                  </div>
                </Panel>
              ) : null}
            </div>
          </div>
        </div>
      </div>
    )
  }

  function renderRuntime(): React.JSX.Element {
    return (
      <div className="view-stack runtime-view page-enter page-enter-1 player-focused-view">
        <NoticeBanner notice={contestNotice ?? publicNotice} />

        <section className="board-summary panel panel-hero runtime-stage-hero">
          <div className="board-summary-grid">
            <div>
              <p className="eyebrow">Runtime Control</p>
              <h2>{selectedChallengeSummary?.dynamic ? selectedChallengeSummary.title : '动态实例'}</h2>
              <p className="panel-subtitle">
                {selectedChallengeSummary?.dynamic
                  ? `${selectedChallengeSummary.category} · ${selectedChallengeSummary.points} pts · 这里仅处理实例状态与操作，题面和附件继续留在题目页。`
                  : '动态题实例已经从主导航中收起，只在需要时进入这里处理启动、续期和回收。'}
              </p>
            </div>
            <div className="board-summary-metrics compact-stat-row">
              <div className="summary-card">
                <span>动态题</span>
                <strong>{dynamicChallenges.length}</strong>
              </div>
              <div className="summary-card">
                <span>实例状态</span>
                <strong>{runtimeInstance?.status ?? 'idle'}</strong>
              </div>
              <div className="summary-card">
                <span>剩余时间</span>
                <strong>{runtimeInstance ? formatRemaining(runtimeInstance.expires_at) : '未启动'}</strong>
              </div>
            </div>
          </div>
        </section>

        <div className="workspace-grid runtime-layout runtime-focused-layout">
          <Panel eyebrow="动态题" title="选择需要实例的题目" subtitle="题面仍在题目页查看，这里只处理实例。" className="rail-panel player-side-panel">
            <div className="challenge-card-list">
              {dynamicChallenges.map((item) => (
                <button
                  className={selectedChallengeId === item.id ? 'challenge-card active' : 'challenge-card'}
                  key={item.id}
                  onClick={() => setSelectedChallengeId(item.id)}
                  type="button"
                >
                  <div className="challenge-card-head">
                    <strong>{item.title}</strong>
                    <span>{item.points} pts</span>
                  </div>
                  <div className="badge-row">
                    <span className="badge">{item.category}</span>
                    <span className={`badge difficulty-${item.difficulty}`}>{formatDifficultyLabel(item.difficulty)}</span>
                  </div>
                </button>
              ))}
              {dynamicChallenges.length === 0 ? <div className="empty-state">当前没有公开动态题。</div> : null}
            </div>
          </Panel>

          <div className="content-stack">
            <Panel
              eyebrow="实例控制"
              title={selectedChallengeSummary?.dynamic ? selectedChallengeSummary.title : '选择动态题'}
              subtitle={
                selectedChallengeSummary?.dynamic
                  ? `${selectedChallengeSummary.category} · ${selectedChallengeSummary.points} pts · 只保留选手真正需要的状态和动作。`
                  : '从左侧选择一题后再启动实例。'
              }
              actions={
                <button className="ghost-button" onClick={() => setView('board')} type="button">
                  返回题目页
                </button>
              }
            >
              <NoticeBanner notice={runtimeNotice} />
              {!selectedChallengeSummary?.dynamic ? (
                <div className="empty-state">从左侧选择动态题后，可以在这里启动、续期或回收实例。</div>
              ) : (
                <div className="runtime-focus-stack">
                  <div className="runtime-context-card">
                    <div>
                      <span className="section-tag">关联题目</span>
                      <h3>{selectedChallengeSummary.title}</h3>
                      <p className="hint-text">题面、附件和提交通道仍保留在题目页，实例页只负责环境操作。</p>
                    </div>
                    <button className="ghost-button" onClick={() => setView('board')} type="button">
                      返回当前题目
                    </button>
                  </div>

                  <div className="runtime-metrics-grid runtime-status-grid">
                    <div className="runtime-metric">
                      <span>状态</span>
                      <strong>{runtimeInstance?.status ?? 'idle'}</strong>
                    </div>
                    <div className="runtime-metric">
                      <span>剩余时间</span>
                      <strong>{runtimeInstance ? formatRemaining(runtimeInstance.expires_at) : '未启动'}</strong>
                    </div>
                    <div className="runtime-metric">
                      <span>续期次数</span>
                      <strong>{runtimeInstance?.renew_count ?? 0}</strong>
                    </div>
                    <div className="runtime-metric">
                      <span>实例入口</span>
                      <strong>{runtimeInstance?.access_url ? '已分配' : '未生成'}</strong>
                    </div>
                  </div>

                  <div className="runtime-action-strip">
                    <button
                      className="primary-button"
                      disabled={runtimeLoading || !selectedChallengeSummary || !selectedChallengeSummary.dynamic}
                      onClick={() => void handleRuntimeAction('start')}
                      type="button"
                    >
                      {runtimeLoading ? '处理中…' : runtimeInstance ? '重取实例' : '启动实例'}
                    </button>
                    <button
                      className="ghost-button"
                      disabled={runtimeLoading || !runtimeInstance}
                      onClick={() => void handleRuntimeAction('renew')}
                      type="button"
                    >
                      续期实例
                    </button>
                    <button
                      className="ghost-button danger-button"
                      disabled={runtimeLoading || !runtimeInstance}
                      onClick={() => void handleRuntimeAction('delete')}
                      type="button"
                    >
                      回收实例
                    </button>
                    {runtimeInstance?.access_url ? (
                      <a className="link-button" href={runtimeInstance.access_url} rel="noreferrer" target="_blank">
                        打开实例
                      </a>
                    ) : null}
                  </div>
                </div>
              )}
            </Panel>

            <Panel eyebrow="使用提示" title="实例使用建议" subtitle="把说明改成操作导向，而不是暴露底层接口。">
              <div className="detail-list compact-list runtime-help-grid">
                <div className="detail-row">
                  <span>查看题面</span>
                  <strong>题目说明、附件和提交通道都保留在题目页</strong>
                </div>
                <div className="detail-row">
                  <span>启动时机</span>
                  <strong>只有在题目明确需要运行环境时再启动实例</strong>
                </div>
                <div className="detail-row">
                  <span>释放资源</span>
                  <strong>完成调试后及时回收，避免占用配额</strong>
                </div>
              </div>
            </Panel>
          </div>
        </div>
      </div>
    )
  }

  function renderScoreboard(): React.JSX.Element {
    return (
      <div className="view-stack scoreboard-view page-enter page-enter-1">
        <NoticeBanner notice={contestNotice ?? publicNotice} />

        <section className="board-summary panel panel-hero scoreboard-stage-hero">
          <div className="board-summary-grid">
            <div>
              <p className="eyebrow">Scoreboard</p>
              <h2>公开排行榜</h2>
              <p className="panel-subtitle">改成更紧凑的单列表结构，先看名次，再按需要展开单人的解题明细。</p>
            </div>
            <div className="board-summary-metrics compact-stat-row">
              <div className="summary-card">
                <span>上榜人数</span>
                <strong>{scoreboard.length}</strong>
              </div>
              <div className="summary-card">
                <span>我的位置</span>
                <strong>{personalRankLabel}</strong>
              </div>
              <div className="summary-card">
                <span>我的积分</span>
                <strong>{authUser ? totalScore : '--'}</strong>
              </div>
            </div>
          </div>
        </section>

        <Panel eyebrow="Rankings" title="榜单明细" subtitle="列表优先，明细次之。">
          <NoticeBanner notice={publicNotice} />

          {authUser ? (
            <div className="scoreboard-personal-strip">
              <div className="summary-card">
                <span>累计得分</span>
                <strong>{totalScore}</strong>
              </div>
              <div className="summary-card">
                <span>解题数</span>
                <strong>{mySolves.length}</strong>
              </div>
              <div className="summary-card">
                <span>提交数</span>
                <strong>{mySubmissions.length}</strong>
              </div>
            </div>
          ) : null}

          {scoreboard.length === 0 ? <div className="empty-state">当前还没有公开排行。</div> : null}
          {scoreboard.length > 0 ? (
            <div className="scoreboard-table">
              <div className="scoreboard-table-header">
                <span>排名</span>
                <span>选手</span>
                <span>总分</span>
                <span>解题</span>
                <span>最后解题</span>
                <span>明细</span>
              </div>

              {scoreboard.map((item) => {
                const expanded = Boolean(expandedRanks[item.user_id])
                const isCurrentUser = authUser?.id === item.user_id
                return (
                  <div className={expanded ? (isCurrentUser ? 'scoreboard-table-entry expanded current-user' : 'scoreboard-table-entry expanded') : isCurrentUser ? 'scoreboard-table-entry current-user' : 'scoreboard-table-entry'} key={item.user_id}>
                    <div className="scoreboard-table-row">
                      <strong className="scoreboard-rank-cell">#{item.rank}</strong>
                      <div className="scoreboard-player-cell">
                        <strong>{item.display_name || item.username}</strong>
                        <small>{isCurrentUser ? '@' + item.username + ' · 我' : '@' + item.username}</small>
                      </div>
                      <div className="scoreboard-value-cell">
                        <strong>{item.score}</strong>
                        <span>pts</span>
                      </div>
                      <div className="scoreboard-value-cell">
                        <strong>{item.solves.length}</strong>
                        <span>题</span>
                      </div>
                      <div className="scoreboard-time-cell">{formatDateTime(item.last_solve_at)}</div>
                      <div className="scoreboard-actions-cell">
                        <button
                          className="ghost-button"
                          onClick={() => setExpandedRanks((current) => ({ ...current, [item.user_id]: !expanded }))}
                          type="button"
                        >
                          {expanded ? '收起' : '展开'}
                        </button>
                      </div>
                    </div>

                    {expanded ? (
                      <div className="scoreboard-row-details">
                        {item.solves.length > 0 ? (
                          item.solves.map((solve) => (
                            <div className="scoreboard-table-solve" key={`${item.user_id}-${solve.challenge_id}-${solve.solved_at}`}>
                              <div>
                                <strong>{solve.challenge_title}</strong>
                                <span>{solve.challenge_slug}</span>
                              </div>
                              <div className="scoreboard-solve-meta">
                                <span>{solve.category}</span>
                                <span>{formatDifficultyLabel(solve.difficulty)}</span>
                                <span>{solve.awarded_points} pts</span>
                                <span>{formatDateTime(solve.solved_at)}</span>
                              </div>
                            </div>
                          ))
                        ) : (
                          <div className="empty-state small">当前还没有解题记录。</div>
                        )}
                      </div>
                    ) : null}
                  </div>
                )
              })}
            </div>
          ) : null}
        </Panel>
      </div>
    )
  }

  function renderAdminChallenges(): React.JSX.Element {
    return (
      <div className="workspace-grid admin-grid">
        <Panel
          eyebrow="Challenge Catalog"
          title="题目管理"
          subtitle="左侧是题目目录，右侧是实际写入后端的编辑表单。"
          className="rail-panel"
          actions={
            canWriteChallenges ? (
              <button
                className="ghost-button"
                onClick={() => {
                  setSelectedAdminChallenge('new')
                  setAdminChallengeDraft(createBlankChallengeDraft())
                  setAdminChallengeNotice(null)
                }}
                type="button"
              >
                新建题目
              </button>
            ) : null
          }
        >
          <NoticeBanner notice={adminChallengesNotice} />
          <div className="toolbar-row">
            <label className="field">
              <span>状态筛选</span>
              <select value={adminChallengeStatusFilter} onChange={(event) => setAdminChallengeStatusFilter(event.target.value as AdminChallengeStatusFilter)}>
                <option value="all">全部状态</option>
                {challengeStatusOptions.map((option) => (
                  <option key={option} value={option}>
                    {formatChallengeStatus(option)}
                  </option>
                ))}
              </select>
            </label>
          </div>
          {adminChallengesLoading ? <div className="empty-state">正在读取后台题目目录…</div> : null}
          <div className="challenge-card-list">
            {filteredAdminChallenges.map((item) => (
              <button
                className={selectedAdminChallenge === item.id ? 'challenge-card active' : 'challenge-card'}
                key={item.id}
                onClick={() => {
                  setSelectedAdminChallenge(item.id)
                  setAdminChallengeNotice(null)
                }}
                type="button"
              >
                <div className="challenge-card-head">
                  <strong>{item.title}</strong>
                  <span>#{item.id}</span>
                </div>
                <div className="badge-row">
                  <span className="badge">{item.category}</span>
                  <span className={challengeStatusBadgeClass(item.status)}>{formatChallengeStatus(item.status)}</span>
                  {item.dynamic_enabled ? <span className="badge badge-accent">Dynamic</span> : null}
                </div>
              </button>
            ))}
            {filteredAdminChallenges.length === 0 && !adminChallengesLoading ? <div className="empty-state">当前筛选条件下没有题目。</div> : null}
          </div>
        </Panel>

        <div className="content-stack">
          <Panel
            eyebrow="Challenge Editor"
            title={selectedAdminChallenge === 'new' ? '新建题目' : adminChallengeDetail?.title ?? '题目编辑'}
            subtitle={canWriteChallenges ? '保存会调用真实的 `POST/PATCH /api/v1/admin/challenges`。' : '当前账号为只读，仍可查看详情并按权限上传附件。'}
          >
            <NoticeBanner notice={adminChallengeNotice} />
            {adminChallengeDetailLoading ? <div className="empty-state">正在读取题目详情…</div> : null}
            {!adminChallengeDetailLoading ? (
              <form className="form-grid" onSubmit={handleSaveAdminChallenge}>
                <label className="field">
                  <span>Slug</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, slug: event.target.value }))}
                    value={adminChallengeDraft.slug}
                  />
                </label>
                <label className="field">
                  <span>标题</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, title: event.target.value }))}
                    value={adminChallengeDraft.title}
                  />
                </label>
                <label className="field">
                  <span>分类</span>
                  <select
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, category_slug: event.target.value }))}
                    value={adminChallengeDraft.category_slug}
                  >
                    {categoryOptions.map((option) => (
                      <option key={option} value={option}>
                        {option}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field">
                  <span>难度</span>
                  <select
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, difficulty: event.target.value }))}
                    value={adminChallengeDraft.difficulty}
                  >
                    {difficultyOptions.map((option) => (
                      <option key={option} value={option}>
                        {option}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field">
                  <span>分值</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, points: event.target.value }))}
                    type="number"
                    value={adminChallengeDraft.points}
                  />
                </label>
                <label className="field">
                  <span>排序</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, sort_order: event.target.value }))}
                    type="number"
                    value={adminChallengeDraft.sort_order}
                  />
                </label>
                <div className="detail-list compact-list">
                  <div className="detail-row">
                    <span>当前状态</span>
                    <strong>{formatChallengeStatus(adminChallengeDraft.status)}</strong>
                  </div>
                  <div className="detail-row">
                    <span>公开可见</span>
                    <strong>{adminChallengeDraft.status === 'published' ? '是' : '否'}</strong>
                  </div>
                </div>
                <label className="field wide-field">
                  <span>题面</span>
                  <textarea
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, description: event.target.value }))}
                    rows={8}
                    value={adminChallengeDraft.description}
                  />
                </label>
                <label className="field">
                  <span>判题类型</span>
                  <select
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, flag_type: event.target.value }))}
                    value={adminChallengeDraft.flag_type}
                  >
                    {flagTypeOptions.map((option) => (
                      <option key={option} value={option}>
                        {option}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field">
                  <span>{adminChallengeDraft.flag_type === 'regex' ? '正则模式' : 'Flag'}</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, flag_value: event.target.value }))}
                    placeholder={adminChallengeDraft.flag_type === 'regex' ? '^flag\{welcome(-[0-9]{2})?\}$' : 'flag{welcome}'}
                    value={adminChallengeDraft.flag_value}
                  />
                </label>
                <div className="detail-list compact-list wide-field">
                  <div className="detail-row">
                    <span>static</span>
                    <strong>严格区分大小写，全字符串相等</strong>
                  </div>
                  <div className="detail-row">
                    <span>case_insensitive</span>
                    <strong>忽略大小写，并会去掉首尾空白</strong>
                  </div>
                  <div className="detail-row">
                    <span>regex</span>
                    <strong>按 Go 正则表达式匹配提交内容</strong>
                  </div>
                </div>
                <label className="field">
                  <span>发布状态</span>
                  <select
                    onChange={(event) =>
                      setAdminChallengeDraft((current) => ({
                        ...current,
                        status: event.target.value,
                        visible: event.target.value === 'published',
                      }))
                    }
                    value={adminChallengeDraft.status}
                  >
                    {challengeStatusOptions.map((option) => (
                      <option key={option} value={option}>
                        {formatChallengeStatus(option)}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="toggle-field">
                  <input
                    checked={adminChallengeDraft.dynamic_enabled}
                    onChange={(event) =>
                      setAdminChallengeDraft((current) => ({
                        ...current,
                        dynamic_enabled: event.target.checked,
                        runtime_enabled: event.target.checked || current.runtime_enabled,
                      }))
                    }
                    type="checkbox"
                  />
                  <span>动态题</span>
                </label>

                <div className="divider-line wide-field">
                  <span>运行时配置</span>
                </div>

                <label className="toggle-field">
                  <input
                    checked={adminChallengeDraft.runtime_enabled}
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, runtime_enabled: event.target.checked }))}
                    type="checkbox"
                  />
                  <span>启用运行配置</span>
                </label>
                <label className="field">
                  <span>镜像名</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, image_name: event.target.value }))}
                    value={adminChallengeDraft.image_name}
                  />
                </label>
                <label className="field">
                  <span>协议</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, exposed_protocol: event.target.value }))}
                    value={adminChallengeDraft.exposed_protocol}
                  />
                </label>
                <label className="field">
                  <span>容器端口</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, container_port: event.target.value }))}
                    type="number"
                    value={adminChallengeDraft.container_port}
                  />
                </label>
                <label className="field">
                  <span>默认 TTL</span>
                  <input
                    onChange={(event) =>
                      setAdminChallengeDraft((current) => ({ ...current, default_ttl_seconds: event.target.value }))
                    }
                    type="number"
                    value={adminChallengeDraft.default_ttl_seconds}
                  />
                </label>
                <label className="field">
                  <span>最大续期次数</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, max_renew_count: event.target.value }))}
                    type="number"
                    value={adminChallengeDraft.max_renew_count}
                  />
                </label>
                <label className="field">
                  <span>内存限制</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, memory_limit_mb: event.target.value }))}
                    type="number"
                    value={adminChallengeDraft.memory_limit_mb}
                  />
                </label>
                <label className="field">
                  <span>CPU 限制</span>
                  <input
                    onChange={(event) =>
                      setAdminChallengeDraft((current) => ({ ...current, cpu_limit_millicores: event.target.value }))
                    }
                    type="number"
                    value={adminChallengeDraft.cpu_limit_millicores}
                  />
                </label>
                <label className="field">
                  <span>题目并发上限</span>
                  <input
                    onChange={(event) =>
                      setAdminChallengeDraft((current) => ({ ...current, max_active_instances: event.target.value }))
                    }
                    type="number"
                    value={adminChallengeDraft.max_active_instances}
                  />
                </label>
                <label className="field">
                  <span>用户冷却秒数</span>
                  <input
                    onChange={(event) =>
                      setAdminChallengeDraft((current) => ({ ...current, user_cooldown_seconds: event.target.value }))
                    }
                    type="number"
                    value={adminChallengeDraft.user_cooldown_seconds}
                  />
                </label>
                <label className="field wide-field">
                  <span>环境变量</span>
                  <textarea
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, env_text: event.target.value }))}
                    placeholder="KEY=value"
                    rows={6}
                    value={adminChallengeDraft.env_text}
                  />
                </label>
                <label className="field wide-field">
                  <span>启动命令</span>
                  <textarea
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, command_text: event.target.value }))}
                    placeholder="每行一个参数"
                    rows={5}
                    value={adminChallengeDraft.command_text}
                  />
                </label>

                <div className="form-footer wide-field">
                  <button className="primary-button" disabled={!canWriteChallenges || adminChallengeSubmitting} type="submit">
                    {adminChallengeSubmitting ? '保存中…' : selectedAdminChallenge === 'new' ? '创建题目' : '保存修改'}
                  </button>
                  {!canWriteChallenges ? <span className="hint-text">当前角色无题目写权限。</span> : null}
                </div>
              </form>
            ) : null}
          </Panel>

          <Panel eyebrow="Ownership" title="负责人" subtitle={canManageChallengeAuthors ? '管理员可以直接调整题目负责人，author 只读。' : '当前账号只能查看负责人。'}>
            <div className="attachment-manager">
              <div className="detail-list compact-list">
                {(adminChallengeDetail?.authors ?? []).map((author) => (
                  <label className="detail-row" key={author.user_id}>
                    <span>
                      {author.display_name || author.username} · @{author.username}
                    </span>
                    <strong>
                      {author.role} · {author.email}
                    </strong>
                  </label>
                ))}
                {(adminChallengeDetail?.authors ?? []).length === 0 ? <div className="empty-state small">当前题目还没有负责人。</div> : null}
              </div>
              {canManageChallengeAuthors ? (
                <div className="detail-list compact-list">
                  {adminUsers
                    .filter((user) => user.role === 'author' || user.role === 'admin')
                    .map((user) => (
                      <label className="detail-row" key={`author-candidate-${user.id}`}>
                        <span>{user.display_name || user.username}</span>
                        <strong>
                          <input
                            checked={adminChallengeAuthorIDs.includes(user.id)}
                            onChange={(event) =>
                              setAdminChallengeAuthorIDs((current) =>
                                event.target.checked
                                  ? current.includes(user.id)
                                    ? current
                                    : [...current, user.id]
                                  : current.filter((item) => item !== user.id),
                              )
                            }
                            type="checkbox"
                          />
                        </strong>
                      </label>
                    ))}
                </div>
              ) : null}
              {canManageChallengeAuthors ? (
                <div className="inline-actions wrap-actions">
                  <button
                    className="ghost-button"
                    disabled={adminChallengeAuthorSubmitting || typeof selectedAdminChallenge !== 'number'}
                    onClick={() => void handleSaveChallengeAuthors()}
                    type="button"
                  >
                    {adminChallengeAuthorSubmitting ? '保存中…' : '保存负责人'}
                  </button>
                </div>
              ) : null}
            </div>
          </Panel>

          <Panel eyebrow="Attachments" title="附件管理" subtitle="上传后立即可在公开详情页通过下载接口访问。">
            <div className="attachment-manager">
              <div className="detail-list compact-list">
                {(adminChallengeDetail?.attachments ?? []).map((attachment) => (
                  <div className="detail-row" key={attachment.id}>
                    <span>{attachment.filename}</span>
                    <strong>
                      {attachment.content_type} · {formatBytes(attachment.size_bytes)}
                    </strong>
                  </div>
                ))}
                {(adminChallengeDetail?.attachments ?? []).length === 0 ? <div className="empty-state small">当前题目还没有附件。</div> : null}
              </div>
              <div className="inline-actions wrap-actions">
                <input
                  className="file-input"
                  onChange={(event) => setAttachmentFile(event.target.files?.[0] ?? null)}
                  type="file"
                />
                <button
                  className="ghost-button"
                  disabled={!canUploadAttachments || attachmentUploading || typeof selectedAdminChallenge !== 'number'}
                  onClick={() => void handleUploadAttachment()}
                  type="button"
                >
                  {attachmentUploading ? '上传中…' : '上传附件'}
                </button>
              </div>
            </div>
          </Panel>
        </div>
      </div>
    )
  }

  function renderAdminAnnouncements(): React.JSX.Element {
    return (
      <div className="two-column-layout admin-column-layout">
        <Panel eyebrow="Studio Bulletin Cabinet" title="公告陈列柜" subtitle="过去的公告现在可以直接下架，不再只是只读陈列。">
          <NoticeBanner notice={adminAnnouncementsNotice} />
          {adminAnnouncementsLoading ? <div className="empty-state">正在读取后台公告…</div> : null}
          <div className="card-list archive-notice-list">
            {adminAnnouncements.map((item, index) => (
              <article className={index === 0 ? 'entry-card archive-entry featured-entry' : 'entry-card archive-entry'} key={item.id}>
                <div className="entry-head">
                  <strong>{item.title}</strong>
                  <span>{formatDateTime(item.published_at)}</span>
                </div>
                <p>{item.content}</p>
                <div className="badge-row wrap-actions">
                  {item.pinned ? <span className="badge badge-solid">置顶</span> : <span className="badge">普通</span>}
                  <span className="badge">{item.published ? 'Published' : 'Draft'}</span>
                  <span className="badge">编号 #{item.id}</span>
                </div>
                {canWriteAnnouncements ? (
                  <div className="entry-toolbar">
                    <button
                      className="ghost-button danger-button"
                      disabled={deletingAnnouncementId === item.id}
                      onClick={() => void handleDeleteAnnouncement(item.id)}
                      type="button"
                    >
                      {deletingAnnouncementId === item.id ? '移除中…' : '删除公告'}
                    </button>
                  </div>
                ) : null}
              </article>
            ))}
            {adminAnnouncements.length === 0 && !adminAnnouncementsLoading ? <div className="empty-state">当前没有后台公告。</div> : null}
          </div>
        </Panel>

        <Panel eyebrow="Compose Notice" title="发布公告" subtitle={canWriteAnnouncements ? '表单直接提交到后台创建接口。' : '当前账号没有公告写权限，仅可查看列表。'}>
          {canWriteAnnouncements ? (
            <div className="notice-composer-stack">
              <div className="composer-intro">
                <span>公告编辑</span>
                <p>用于发布赛事实况、提醒和流程说明，支持置顶和立即发布。</p>
              </div>
              <form className="form-grid single-column" onSubmit={handleCreateAnnouncement}>
                <label className="field">
                  <span>标题</span>
                  <input
                    onChange={(event) => setAnnouncementDraft((current) => ({ ...current, title: event.target.value }))}
                    value={announcementDraft.title}
                  />
                </label>
                <label className="field">
                  <span>内容</span>
                  <textarea
                    onChange={(event) => setAnnouncementDraft((current) => ({ ...current, content: event.target.value }))}
                    rows={8}
                    value={announcementDraft.content}
                  />
                </label>
                <div className="toggle-row-grid">
                  <label className="toggle-field">
                    <input
                      checked={announcementDraft.pinned}
                      onChange={(event) => setAnnouncementDraft((current) => ({ ...current, pinned: event.target.checked }))}
                      type="checkbox"
                    />
                    <span>置顶</span>
                  </label>
                  <label className="toggle-field">
                    <input
                      checked={announcementDraft.published}
                      onChange={(event) => setAnnouncementDraft((current) => ({ ...current, published: event.target.checked }))}
                      type="checkbox"
                    />
                    <span>立即发布</span>
                  </label>
                </div>
                <button className="primary-button" disabled={announcementSubmitting} type="submit">
                  {announcementSubmitting ? '提交中…' : '写入公告'}
                </button>
              </form>
            </div>
          ) : (
            <div className="empty-state">当前账号没有公告写权限。</div>
          )}
        </Panel>
      </div>
    )
  }

  function renderAdminTraffic(): React.JSX.Element {
    return (
      <div className="two-column-layout admin-column-layout">
        <Panel eyebrow="Submission Feed" title="提交记录" subtitle="直接读取后台提交流，便于核对判题行为。">
          <NoticeBanner notice={adminSubmissionsNotice} />
          {adminSubmissionsLoading ? <div className="empty-state">正在读取提交记录…</div> : null}
          <div className="table-stack">
            {adminSubmissions.map((item) => (
              <div className="table-row multi-column" key={item.id}>
                <strong>#{item.id}</strong>
                <span>{item.challenge_slug}</span>
                <span>{item.username}</span>
                <span>{item.correct ? 'Accepted' : 'Wrong'}</span>
                <small>{formatDateTime(item.submitted_at)}</small>
              </div>
            ))}
            {adminSubmissions.length === 0 && !adminSubmissionsLoading ? <div className="empty-state">当前没有提交记录。</div> : null}
          </div>
        </Panel>

        <Panel eyebrow="Runtime Feed" title="实例监控" subtitle="支持按权限终止实例，便于运维处置。">
          <NoticeBanner notice={adminInstancesNotice} />
          {adminInstancesLoading ? <div className="empty-state">正在读取实例记录…</div> : null}
          <div className="card-list">
            {adminInstances.map((item) => (
              <article className="entry-card" key={item.id}>
                <div className="entry-head">
                  <strong>
                    #{item.id} · {item.challenge_slug}
                  </strong>
                  <span>{item.username}</span>
                </div>
                <p>
                  状态 {item.status} · 端口 {item.host_port} · 到期 {formatDateTime(item.expires_at)}
                </p>
                <div className="inline-actions wrap-actions">
                  <span className="badge">{item.container_id}</span>
                  <button
                    className="ghost-button danger-button"
                    disabled={!canTerminateInstances || item.status === 'terminated' || terminatingInstanceId === item.id}
                    onClick={() => void handleTerminateInstance(item.id)}
                    type="button"
                  >
                    {terminatingInstanceId === item.id ? '处理中…' : '终止实例'}
                  </button>
                </div>
              </article>
            ))}
            {adminInstances.length === 0 && !adminInstancesLoading ? <div className="empty-state">当前没有实例记录。</div> : null}
          </div>
        </Panel>
      </div>
    )
  }

  function renderAdminUsers(): React.JSX.Element {
    return (
      <div className="workspace-grid admin-grid">
        <Panel eyebrow="User Catalog" title="用户列表" subtitle="可修改角色、显示名和状态。" className="rail-panel">
          <NoticeBanner notice={adminUsersNotice} />
          {adminUsersLoading ? <div className="empty-state">正在读取用户…</div> : null}
          <div className="challenge-card-list">
            {adminUsers.map((item) => (
              <button
                className={selectedAdminUserId === item.id ? 'challenge-card active' : 'challenge-card'}
                key={item.id}
                onClick={() => setSelectedAdminUserId(item.id)}
                type="button"
              >
                <div className="challenge-card-head">
                  <strong>{item.display_name || item.username}</strong>
                  <span>#{item.id}</span>
                </div>
                <div className="badge-row">
                  <span className="badge">{item.role}</span>
                  <span className="badge">{item.status}</span>
                </div>
              </button>
            ))}
          </div>
        </Panel>

        <Panel eyebrow="User Editor" title={selectedAdminUser?.username ?? '选择用户'} subtitle="这个表单直接对应 `PATCH /api/v1/admin/users/:id`。">
          {!selectedAdminUser ? <div className="empty-state">从左侧选择一个用户开始编辑。</div> : null}
          {selectedAdminUser ? (
            <form className="form-grid" onSubmit={handleSaveUser}>
              <label className="field">
                <span>用户名</span>
                <input disabled value={selectedAdminUser.username} />
              </label>
              <label className="field">
                <span>邮箱</span>
                <input disabled value={selectedAdminUser.email} />
              </label>
              <label className="field">
                <span>角色</span>
                <select
                  onChange={(event) => setAdminUserDraft((current) => ({ ...current, role: event.target.value }))}
                  value={adminUserDraft.role}
                >
                  {userRoleOptions.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field">
                <span>状态</span>
                <select
                  onChange={(event) => setAdminUserDraft((current) => ({ ...current, status: event.target.value }))}
                  value={adminUserDraft.status}
                >
                  {userStatusOptions.map((option) => (
                    <option key={option} value={option}>
                      {option}
                    </option>
                  ))}
                </select>
              </label>
              <label className="field wide-field">
                <span>显示名</span>
                <input
                  onChange={(event) =>
                    setAdminUserDraft((current) => ({ ...current, display_name: event.target.value }))
                  }
                  value={adminUserDraft.display_name}
                />
              </label>
              <div className="detail-list compact-list wide-field">
                <div className="detail-row">
                  <span>最后登录</span>
                  <strong>{formatDateTime(selectedAdminUser.last_login_at)}</strong>
                </div>
                <div className="detail-row">
                  <span>注册时间</span>
                  <strong>{formatDateTime(selectedAdminUser.created_at)}</strong>
                </div>
              </div>
              <div className="form-footer wide-field">
                <button className="primary-button" disabled={adminUserSubmitting} type="submit">
                  {adminUserSubmitting ? '保存中…' : '保存用户'}
                </button>
              </div>
            </form>
          ) : null}
        </Panel>
      </div>
    )
  }

  function renderAdminAudit(): React.JSX.Element {
    return (
      <Panel eyebrow="Audit Trail" title="审计日志" subtitle="关键后台动作都会落入审计表。">
        <NoticeBanner notice={adminAuditNotice} />
        {adminAuditLoading ? <div className="empty-state">正在读取审计日志…</div> : null}
        <div className="card-list audit-list">
          {adminAuditLogs.map((item) => (
            <article className="entry-card" key={item.id}>
              <div className="entry-head">
                <strong>
                  {item.action} · {item.resource_type}#{item.resource_id}
                </strong>
                <span>{formatDateTime(item.created_at)}</span>
              </div>
              <p>actor: {item.actor_user_id ?? 'system'}</p>
              <pre className="code-block">{JSON.stringify(item.details ?? {}, null, 2)}</pre>
            </article>
          ))}
          {adminAuditLogs.length === 0 && !adminAuditLoading ? <div className="empty-state">当前没有审计记录。</div> : null}
        </div>
      </Panel>
    )
  }

  function renderAdmin(): React.JSX.Element {
    return (
      <div className="view-stack">
        <Panel
          eyebrow="管理后台"
          title="后台工作区"
          subtitle="按后端实际权限拆分。admin 拥有全量后台能力；ops 聚焦运维读写；author 仅处理题目与附件。"
        >
          <div className="tab-strip">
            {availableAdminSections.map((item) => (
              <button
                className={adminSection === item.id ? 'tab-pill active' : 'tab-pill'}
                key={item.id}
                onClick={() => setAdminSection(item.id)}
                type="button"
              >
                <span>{item.label}</span>
                <small>{item.note}</small>
              </button>
            ))}
          </div>
        </Panel>

        {adminSection === 'contest' ? renderAdminContest() : null}
        {adminSection === 'challenges' ? renderAdminChallenges() : null}
        {adminSection === 'announcements' ? renderAdminAnnouncements() : null}
        {adminSection === 'traffic' ? renderAdminTraffic() : null}
        {adminSection === 'users' ? renderAdminUsers() : null}
        {adminSection === 'audit' ? renderAdminAudit() : null}
      </div>
    )
  }

  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">
        跳到主内容
      </a>
      <header className="topbar">
        <div className="brand-block">
          <div className="brand-mark">
            <img alt="御林工作室标识" src={studioMarkUrl} />
          </div>
          <div className="brand-copy">
            <p className="eyebrow">御林工作室</p>
            <h1>招新赛工作台</h1>
            <small>{contestStatusLabel}</small>
          </div>
        </div>

        <nav className="main-nav" aria-label="Primary">
          {views
            .filter((item) => {
              if (item.id === 'admin') {
                return canAccessAdmin
              }
              if (item.id === 'runtime') {
                return false
              }
              return true
            })
            .map((item) => (
              <button
                className={view === item.id ? 'nav-pill active' : 'nav-pill'}
                key={item.id}
                onClick={() => setView(item.id)}
                type="button"
              >
                <span>{item.label}</span>
              </button>
            ))}
        </nav>

        <div className="session-block">
          {authUser ? (
            <div className="user-chip">
              <strong>{authUser.display_name || authUser.username}</strong>
              <span>{authUser.role}</span>
            </div>
          ) : (
            <div className="user-chip ghost-chip">
              <strong>Guest</strong>
              <span>未登录</span>
            </div>
          )}
          {authUser ? (
            <button className="ghost-button" onClick={handleLogout} type="button">
              退出
            </button>
          ) : (
            <button className="primary-button" onClick={() => setView('briefing')} type="button">
              登录
            </button>
          )}
        </div>
      </header>

      <main className="page-shell" id="main-content">
        {view === 'briefing' ? renderBriefing() : null}
        {view === 'board' ? renderBoard() : null}
        {view === 'runtime' ? renderRuntime() : null}
        {view === 'scoreboard' ? renderScoreboard() : null}
        {view === 'admin' ? renderAdmin() : null}
      </main>
    </div>
  )
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
