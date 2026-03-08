import React, { useCallback, useEffect, useMemo, useState } from 'react'
import ReactDOM from 'react-dom/client'
import {
  api,
  type AdminAnnouncement,
  type AdminAuditLog,
  type AdminChallengeDetail,
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

type View = 'briefing' | 'board' | 'runtime' | 'scoreboard' | 'admin'
type AuthMode = 'login' | 'register'
type AdminSection = 'challenges' | 'announcements' | 'traffic' | 'users' | 'audit'
type NoticeTone = 'neutral' | 'success' | 'danger'

type Notice = {
  tone: NoticeTone
  text: string
}

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

const views: Array<{ id: View; label: string; note: string }> = [
  { id: 'briefing', label: '总览', note: 'Contest Brief' },
  { id: 'board', label: '题目', note: 'Challenge Board' },
  { id: 'runtime', label: '实例', note: 'Runtime Center' },
  { id: 'scoreboard', label: '排行', note: 'Scoreboard' },
  { id: 'admin', label: '管理', note: 'Admin Console' },
]

const categoryOptions = ['web', 'pwn', 'misc', 'crypto', 'reverse']
const difficultyOptions = ['easy', 'normal', 'hard']
const userRoleOptions = ['player', 'admin']
const userStatusOptions = ['active', 'disabled']

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
    visible: draft.visible,
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

function NoticeBanner({ notice }: { notice: Notice | null }): React.JSX.Element | null {
  if (!notice) {
    return null
  }
  return <div className={`notice notice-${notice.tone}`}>{notice.text}</div>
}

function App(): React.JSX.Element {
  const [view, setView] = useState<View>('briefing')
  const [authMode, setAuthMode] = useState<AuthMode>('login')
  const [token, setToken] = useState<string>(() => window.localStorage.getItem(TOKEN_STORAGE_KEY) ?? '')
  const [sessionLoading, setSessionLoading] = useState(Boolean(token))
  const [authBusy, setAuthBusy] = useState(false)
  const [authNotice, setAuthNotice] = useState<Notice | null>(null)
  const [authUser, setAuthUser] = useState<AuthUser | null>(null)
  const [loginForm, setLoginForm] = useState({ identifier: 'admin@ctf.local', password: 'Admin123!' })
  const [registerForm, setRegisterForm] = useState({
    username: '',
    email: '',
    display_name: '',
    password: '',
  })

  const [announcements, setAnnouncements] = useState<PublicAnnouncement[]>([])
  const [challengeList, setChallengeList] = useState<PublicChallengeSummary[]>([])
  const [scoreboard, setScoreboard] = useState<ScoreboardEntry[]>([])
  const [publicLoading, setPublicLoading] = useState(true)
  const [publicNotice, setPublicNotice] = useState<Notice | null>(null)

  const [challengeSearch, setChallengeSearch] = useState('')
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

  const [adminSection, setAdminSection] = useState<AdminSection>('challenges')
  const [adminChallenges, setAdminChallenges] = useState<AdminChallengeSummary[]>([])
  const [adminChallengesLoading, setAdminChallengesLoading] = useState(false)
  const [adminChallengesNotice, setAdminChallengesNotice] = useState<Notice | null>(null)
  const [selectedAdminChallenge, setSelectedAdminChallenge] = useState<number | 'new' | null>(null)
  const [adminChallengeDetail, setAdminChallengeDetail] = useState<AdminChallengeDetail | null>(null)
  const [adminChallengeDraft, setAdminChallengeDraft] = useState<AdminChallengeDraft>(createBlankChallengeDraft())
  const [adminChallengeDetailLoading, setAdminChallengeDetailLoading] = useState(false)
  const [adminChallengeNotice, setAdminChallengeNotice] = useState<Notice | null>(null)
  const [adminChallengeSubmitting, setAdminChallengeSubmitting] = useState(false)
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

  const canAccessAdmin = authUser?.role === 'admin' || authUser?.role === 'ops'
  const canWriteChallenges = authUser?.role === 'admin'
  const canWriteAnnouncements = authUser?.role === 'admin'
  const canUploadAttachments = authUser?.role === 'admin' || authUser?.role === 'ops'
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

  const loadScoreboard = useCallback(async () => {
    const response = await api.scoreboard()
    setScoreboard(response.items)
  }, [])

  const loadPublicData = useCallback(async () => {
    setPublicLoading(true)
    setPublicNotice(null)
    try {
      const [announcementResponse, challengeResponse, scoreboardResponse] = await Promise.all([
        api.announcements(),
        api.challenges(),
        api.scoreboard(),
      ])
      setAnnouncements(announcementResponse.items)
      setChallengeList(challengeResponse.items)
      setScoreboard(scoreboardResponse.items)
    } catch (error) {
      setPublicNotice({ tone: 'danger', text: describeError(error, '公开数据加载失败。') })
    } finally {
      setPublicLoading(false)
    }
  }, [])

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
    sections.push({ id: 'challenges', label: '题目', note: 'Catalog' })
    sections.push({ id: 'announcements', label: '公告', note: 'Broadcast' })
    sections.push({ id: 'traffic', label: '流量', note: 'Ops Feed' })
    if (canManageUsers) {
      sections.push({ id: 'users', label: '用户', note: 'Identity' })
    }
    sections.push({ id: 'audit', label: '审计', note: 'Audit Trail' })
    return sections
  }, [canAccessAdmin, canManageUsers])

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
    if (adminSection === 'challenges') {
      void loadAdminChallenges()
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
    canManageUsers,
    loadAdminAnnouncements,
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
      return
    }
    if (typeof selectedAdminChallenge !== 'number') {
      return
    }
    void loadAdminChallengeDetail(selectedAdminChallenge)
  }, [adminSection, canAccessAdmin, loadAdminChallengeDetail, selectedAdminChallenge, token, view])

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
    for (const item of challengeList) {
      if (
        needle &&
        !item.title.toLowerCase().includes(needle) &&
        !item.category.toLowerCase().includes(needle) &&
        !item.slug.toLowerCase().includes(needle)
      ) {
        continue
      }
      const current = groups.get(item.category) ?? []
      current.push(item)
      groups.set(item.category, current)
    }
    return Array.from(groups.entries()).map(([category, items]) => ({ category, items }))
  }, [challengeList, challengeSearch])

  const totalScore = useMemo(() => mySolves.reduce((sum, item) => sum + item.awarded_points, 0), [mySolves])

  const selectedChallengeAttempts = useMemo(
    () => mySubmissions.filter((item) => String(item.challenge_id) === selectedChallengeId).slice(0, 6),
    [mySubmissions, selectedChallengeId],
  )

  const selectedChallengeSolve = useMemo(
    () => mySolves.find((item) => String(item.challenge_id) === selectedChallengeId) ?? null,
    [mySolves, selectedChallengeId],
  )

  const dynamicChallenges = useMemo(() => challengeList.filter((item) => item.dynamic), [challengeList])

  const selectedChallengeMeta = [
    { label: '分类', value: challengeDetail?.category ?? selectedChallengeSummary?.category ?? '未分类' },
    { label: '分值', value: `${challengeDetail?.points ?? selectedChallengeSummary?.points ?? 0} pts` },
    { label: '难度', value: challengeDetail?.difficulty ?? '未标注' },
    { label: '实例', value: selectedChallengeSummary?.dynamic ? '动态题' : '静态题' },
  ]

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
      setView(response.user.role === 'admin' || response.user.role === 'ops' ? 'admin' : 'board')
    } catch (error) {
      setAuthNotice({ tone: 'danger', text: describeError(error, '登录失败。') })
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
      setAuthNotice({ tone: 'danger', text: describeError(error, '注册失败。') })
    } finally {
      setAuthBusy(false)
    }
  }

  function handleLogout() {
    clearSession('已退出当前账号。')
    setView('briefing')
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
      setAdminChallengeNotice({ tone: 'danger', text: guardedError(error, '题目保存失败。') })
    } finally {
      setAdminChallengeSubmitting(false)
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
      setAdminChallengeNotice({ tone: 'danger', text: guardedError(error, '附件上传失败。') })
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
      setAdminAnnouncementsNotice({ tone: 'danger', text: guardedError(error, '公告创建失败。') })
    } finally {
      setAnnouncementSubmitting(false)
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
      setAdminInstancesNotice({ tone: 'danger', text: guardedError(error, '实例终止失败。') })
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
      setAdminUsersNotice({ tone: 'danger', text: guardedError(error, '用户更新失败。') })
    } finally {
      setAdminUserSubmitting(false)
    }
  }

  const briefingCards = [
    { label: '公开题目', value: String(challengeList.length), note: 'visible' },
    { label: '动态题', value: String(dynamicChallenges.length), note: 'runtime' },
    { label: '已发布公告', value: String(announcements.length), note: 'announcements' },
    { label: '登录状态', value: authUser ? authUser.role : 'guest', note: authUser ? authUser.username : 'anonymous' },
  ]

  function renderBriefing(): React.JSX.Element {
    return (
      <div className="view-stack">
        <Panel
          eyebrow="Contest Surface"
          title="Recruit 2025 前后端已对齐的工作台"
          subtitle="公开区直接读取后端公告、题目和排行榜；登录后会拉取个人历史，管理员可进入真实管理接口。"
          actions={
            <div className="hero-stats">
              {briefingCards.map((item) => (
                <div className="stat-chip" key={item.label}>
                  <span>{item.label}</span>
                  <strong>{item.value}</strong>
                  <small>{item.note}</small>
                </div>
              ))}
            </div>
          }
        >
          <NoticeBanner notice={publicNotice} />
          <div className="hero-copy-grid">
            <div className="editorial-copy">
              <p>
                这一版前端不再依赖本地 mock。题目列表、详情、实例状态、用户历史和后台数据都直接走 `/api/v1`，界面布局也收敛成单一主导航，避免之前重复 bar 和内容拥挤的问题。
              </p>
              <div className="inline-actions">
                <button className="primary-button" onClick={() => setView('board')} type="button">
                  查看题目
                </button>
                <button className="ghost-button" onClick={() => setView('runtime')} type="button">
                  打开实例中心
                </button>
              </div>
            </div>
            <div className="detail-list">
              <div className="detail-row">
                <span>默认管理员</span>
                <strong>`admin@ctf.local / Admin123!`</strong>
              </div>
              <div className="detail-row">
                <span>实例操作</span>
                <strong>启动、续期、回收均接后端真实接口</strong>
              </div>
              <div className="detail-row">
                <span>管理接口</span>
                <strong>题目、公告、用户、审计、实例监控</strong>
              </div>
            </div>
          </div>
        </Panel>

        <div className="two-column-layout">
          <Panel eyebrow="Announcements" title="公告流" subtitle="读取公开公告接口，按置顶和发布时间排序。">
            <div className="card-list">
              {publicLoading ? <div className="empty-state">正在读取公告与题目概况…</div> : null}
              {!publicLoading && announcements.length === 0 ? <div className="empty-state">当前还没有已发布公告。</div> : null}
              {announcements.map((item) => (
                <article className="entry-card" key={item.id}>
                  <div className="entry-head">
                    <strong>{item.title}</strong>
                    <span>{formatDateTime(item.published_at)}</span>
                  </div>
                  <p>{item.content}</p>
                  <div className="badge-row">
                    {item.pinned ? <span className="badge badge-solid">Pinned</span> : <span className="badge">Update</span>}
                  </div>
                </article>
              ))}
            </div>
          </Panel>

          <Panel
            eyebrow="Access"
            title={authUser ? `已登录：${authUser.display_name || authUser.username}` : '身份入口'}
            subtitle={authUser ? '用户信息来自 /me，登出只清理本地 Token。' : '可直接登录默认管理员，也可注册一个 player 账号测试完整流程。'}
          >
            <NoticeBanner notice={authNotice} />
            {sessionLoading ? <div className="empty-state">正在恢复登录态…</div> : null}
            {!sessionLoading && !authUser ? (
              <div className="auth-stack">
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
                        placeholder="admin@ctf.local"
                        value={loginForm.identifier}
                      />
                    </label>
                    <label className="field">
                      <span>密码</span>
                      <input
                        onChange={(event) => setLoginForm((current) => ({ ...current, password: event.target.value }))}
                        placeholder="Admin123!"
                        type="password"
                        value={loginForm.password}
                      />
                    </label>
                    <button className="primary-button" disabled={authBusy} type="submit">
                      {authBusy ? '登录中…' : '进入工作台'}
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
                      {authBusy ? '注册中…' : '注册并登录'}
                    </button>
                  </form>
                )}
              </div>
            ) : null}

            {!sessionLoading && authUser ? (
              <div className="auth-summary">
                <div className="detail-list compact-list">
                  <div className="detail-row">
                    <span>角色</span>
                    <strong>{authUser.role}</strong>
                  </div>
                  <div className="detail-row">
                    <span>邮箱</span>
                    <strong>{authUser.email}</strong>
                  </div>
                  <div className="detail-row">
                    <span>最后登录</span>
                    <strong>{formatDateTime(authUser.last_login_at)}</strong>
                  </div>
                </div>
                <div className="inline-actions">
                  <button className="primary-button" onClick={() => setView(canAccessAdmin ? 'admin' : 'board')} type="button">
                    {canAccessAdmin ? '进入管理区' : '进入题目区'}
                  </button>
                  <button className="ghost-button" onClick={handleLogout} type="button">
                    退出登录
                  </button>
                </div>
              </div>
            ) : null}
          </Panel>
        </div>

        <div className="two-column-layout">
          <Panel eyebrow="Personal Feed" title="我的历史" subtitle="登录后展示真实提交与解题记录。">
            <NoticeBanner notice={historyNotice} />
            {historyLoading ? <div className="empty-state">正在读取个人历史…</div> : null}
            {!authUser ? <div className="empty-state">登录后可查看提交记录、解题历史和累计得分。</div> : null}
            {authUser ? (
              <div className="mini-grid">
                <div className="summary-card">
                  <span>累计得分</span>
                  <strong>{totalScore}</strong>
                </div>
                <div className="summary-card">
                  <span>解出题目</span>
                  <strong>{mySolves.length}</strong>
                </div>
                <div className="summary-card">
                  <span>总提交数</span>
                  <strong>{mySubmissions.length}</strong>
                </div>
              </div>
            ) : null}
            {authUser && mySolves.length === 0 && mySubmissions.length === 0 ? <div className="empty-state">当前账号还没有任何比赛行为。</div> : null}
            {authUser && (mySolves.length > 0 || mySubmissions.length > 0) ? (
              <div className="card-list split-list">
                <div className="subpanel">
                  <h3>最近解题</h3>
                  <div className="compact-list">
                    {mySolves.slice(0, 5).map((item) => (
                      <div className="row-card" key={`solve-${item.id}`}>
                        <strong>{item.challenge_title}</strong>
                        <span>{item.category}</span>
                        <span>{formatDateTime(item.solved_at)}</span>
                      </div>
                    ))}
                  </div>
                </div>
                <div className="subpanel">
                  <h3>最近提交</h3>
                  <div className="compact-list">
                    {mySubmissions.slice(0, 5).map((item) => (
                      <div className="row-card" key={`submission-${item.id}`}>
                        <strong>{item.challenge_title}</strong>
                        <span>{item.correct ? 'Accepted' : 'Wrong'}</span>
                        <span>{formatDateTime(item.submitted_at)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            ) : null}
          </Panel>

          <Panel eyebrow="Board Snapshot" title="分类概况" subtitle="题目列表支持搜索、分类折叠和动态题筛选。">
            <div className="category-summary-list">
              {Array.from(new Set(challengeList.map((item) => item.category))).map((category) => {
                const items = challengeList.filter((item) => item.category === category)
                const solvedCount = items.filter((item) => solvedChallengeIds.has(item.id)).length
                return (
                  <div className="category-summary-card" key={category}>
                    <div>
                      <strong>{category}</strong>
                      <span>{items.length} 题</span>
                    </div>
                    <small>{solvedCount} 题已解</small>
                  </div>
                )
              })}
            </div>
          </Panel>
        </div>
      </div>
    )
  }

  function renderBoard(): React.JSX.Element {
    return (
      <div className="workspace-grid">
        <Panel eyebrow="Challenge Rail" title="题目列表" subtitle="按分类折叠，支持标题 / slug / 分类检索。" className="rail-panel">
          <label className="field compact-field">
            <span>检索</span>
            <input
              onChange={(event) => setChallengeSearch(event.target.value)}
              placeholder="welcome / web / crypto"
              value={challengeSearch}
            />
          </label>
          <div className="accordion-stack">
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
                  <small>{collapsedCategories[group.category] ? '展开' : '收起'}</small>
                </button>
                {!collapsedCategories[group.category] ? (
                  <div className="challenge-card-list">
                    {group.items.map((item) => (
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
                          {item.dynamic ? <span className="badge badge-accent">Dynamic</span> : null}
                          {solvedChallengeIds.has(item.id) ? <span className="badge badge-solid">Solved</span> : null}
                        </div>
                      </button>
                    ))}
                  </div>
                ) : null}
              </section>
            ))}
            {filteredChallengeGroups.length === 0 ? <div className="empty-state">没有匹配的题目。</div> : null}
          </div>
        </Panel>

        <div className="content-stack">
          <Panel
            eyebrow="Challenge Detail"
            title={challengeDetail?.title ?? selectedChallengeSummary?.title ?? '选择题目'}
            subtitle={challengeDetail?.slug ?? selectedChallengeSummary?.slug ?? '从左侧题目列表选择一题查看详情。'}
            actions={
              <div className="badge-row spread-row">
                {selectedChallengeMeta.map((item) => (
                  <span className="badge" key={item.label}>
                    {item.label}: {item.value}
                  </span>
                ))}
              </div>
            }
          >
            <NoticeBanner notice={challengeDetailNotice} />
            {challengeDetailLoading ? <div className="empty-state">正在读取题面…</div> : null}
            {!challengeDetailLoading && challengeDetail ? (
              <div className="detail-stack">
                <article className="statement-card">
                  <h3>题面</h3>
                  <p className="statement-text">{challengeDetail.description}</p>
                </article>
                <div className="two-column-layout compact-columns">
                  <div className="subpanel">
                    <h3>附件</h3>
                    <div className="attachment-list">
                      {challengeDetail.attachments.length === 0 ? <div className="empty-state small">当前没有附件。</div> : null}
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
                  <div className="subpanel">
                    <h3>解题状态</h3>
                    <div className="detail-list compact-list">
                      <div className="detail-row">
                        <span>是否解出</span>
                        <strong>{selectedChallengeSolve ? '已解出' : '未解出'}</strong>
                      </div>
                      <div className="detail-row">
                        <span>解题时间</span>
                        <strong>{formatDateTime(selectedChallengeSolve?.solved_at)}</strong>
                      </div>
                      <div className="detail-row">
                        <span>最近实例</span>
                        <strong>{runtimeInstance ? formatRemaining(runtimeInstance.expires_at) : '未启动'}</strong>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            ) : null}
          </Panel>

          <div className="two-column-layout compact-columns">
            <Panel eyebrow="Judge" title="Flag 与判题" subtitle="提交记录来自后端 `submissions` 与 `solves`。">
              <NoticeBanner notice={submitNotice} />
              {!authUser ? <div className="empty-state">登录后可提交 Flag。</div> : null}
              {authUser ? (
                <form className="form-grid single-column" onSubmit={handleSubmitFlag}>
                  <label className="field">
                    <span>Flag</span>
                    <input
                      onChange={(event) => setFlagInput(event.target.value)}
                      placeholder="flag{...}"
                      value={flagInput}
                    />
                  </label>
                  <button className="primary-button" disabled={submitBusy || !selectedChallengeSummary} type="submit">
                    {submitBusy ? '判题中…' : '提交 Flag'}
                  </button>
                </form>
              ) : null}
            </Panel>

            <Panel eyebrow="Recent Attempts" title="当前题最近提交" subtitle="只展示当前登录用户在该题上的近几次提交。">
              {authUser && selectedChallengeAttempts.length === 0 ? <div className="empty-state">当前题还没有提交记录。</div> : null}
              {!authUser ? <div className="empty-state">登录后展示该题的个人提交历史。</div> : null}
              {authUser && selectedChallengeAttempts.length > 0 ? (
                <div className="compact-list">
                  {selectedChallengeAttempts.map((item) => (
                    <div className="row-card" key={item.id}>
                      <strong>{item.correct ? 'Accepted' : 'Wrong Answer'}</strong>
                      <span>{item.source_ip}</span>
                      <span>{formatDateTime(item.submitted_at)}</span>
                    </div>
                  ))}
                </div>
              ) : null}
            </Panel>
          </div>
        </div>
      </div>
    )
  }

  function renderRuntime(): React.JSX.Element {
    return (
      <div className="workspace-grid">
        <Panel eyebrow="Dynamic Catalog" title="动态题目录" subtitle="只显示 `dynamic = true` 的公开题目。" className="rail-panel">
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
                  <span className="badge badge-accent">Dynamic</span>
                </div>
              </button>
            ))}
            {dynamicChallenges.length === 0 ? <div className="empty-state">当前没有公开动态题。</div> : null}
          </div>
        </Panel>

        <div className="content-stack">
          <Panel
            eyebrow="Runtime Stage"
            title={selectedChallengeSummary?.title ?? '选择动态题'}
            subtitle={selectedChallengeSummary ? `${selectedChallengeSummary.slug} · 通过真实实例 API 管理生命周期。` : '从左侧选择动态题。'}
          >
            <NoticeBanner notice={runtimeNotice} />
            <div className="runtime-hero">
              <div className="runtime-summary">
                <div className="detail-list compact-list">
                  <div className="detail-row">
                    <span>实例状态</span>
                    <strong>{runtimeInstance?.status ?? 'idle'}</strong>
                  </div>
                  <div className="detail-row">
                    <span>访问地址</span>
                    <strong>{runtimeInstance?.access_url ?? '尚未分配'}</strong>
                  </div>
                  <div className="detail-row">
                    <span>到期时间</span>
                    <strong>{runtimeInstance ? `${formatDateTime(runtimeInstance.expires_at)} · 剩余 ${formatRemaining(runtimeInstance.expires_at)}` : '未启动'}</strong>
                  </div>
                  <div className="detail-row">
                    <span>续期次数</span>
                    <strong>{runtimeInstance?.renew_count ?? 0}</strong>
                  </div>
                </div>
              </div>
              <div className="runtime-actions">
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
          </Panel>

          <div className="two-column-layout compact-columns">
            <Panel eyebrow="Rules" title="实例策略" subtitle="由后端 runtime service 控制实例存在性与续期次数。">
              <div className="detail-list compact-list">
                <div className="detail-row">
                  <span>启动接口</span>
                  <strong>`POST /api/v1/challenges/:id/instances/me`</strong>
                </div>
                <div className="detail-row">
                  <span>续期接口</span>
                  <strong>`POST /api/v1/challenges/:id/instances/me/renew`</strong>
                </div>
                <div className="detail-row">
                  <span>回收接口</span>
                  <strong>`DELETE /api/v1/challenges/:id/instances/me`</strong>
                </div>
              </div>
            </Panel>

            <Panel eyebrow="Related Activity" title="相关历史" subtitle="结合题目详情和个人行为，便于快速排查。">
              {!authUser ? <div className="empty-state">登录后显示该题相关提交记录。</div> : null}
              {authUser && selectedChallengeAttempts.length === 0 ? <div className="empty-state">当前题还没有提交记录。</div> : null}
              {authUser && selectedChallengeAttempts.length > 0 ? (
                <div className="compact-list">
                  {selectedChallengeAttempts.map((item) => (
                    <div className="row-card" key={`runtime-submission-${item.id}`}>
                      <strong>{item.challenge_title}</strong>
                      <span>{item.correct ? 'Accepted' : 'Wrong'}</span>
                      <span>{formatDateTime(item.submitted_at)}</span>
                    </div>
                  ))}
                </div>
              ) : null}
            </Panel>
          </div>
        </div>
      </div>
    )
  }

  function renderScoreboard(): React.JSX.Element {
    return (
      <div className="two-column-layout scoreboard-layout">
        <Panel eyebrow="Ranked Board" title="排行榜" subtitle="公开接口只统计激活的 player 用户。">
          <NoticeBanner notice={publicNotice} />
          <div className="table-stack">
            {scoreboard.map((item) => (
              <div className="table-row" key={item.user_id}>
                <strong>#{item.rank}</strong>
                <span>{item.display_name || item.username}</span>
                <span>{item.score} pts</span>
                <small>{formatDateTime(item.last_solve_at)}</small>
              </div>
            ))}
            {scoreboard.length === 0 ? <div className="empty-state">当前还没有公开排行。</div> : null}
          </div>
        </Panel>

        <Panel eyebrow="My Position" title="个人摘要" subtitle="登录后可以用真实数据校验前后端是否一致。">
          {!authUser ? <div className="empty-state">登录后展示你的积分、解题数和最近解题记录。</div> : null}
          {authUser ? (
            <>
              <div className="mini-grid">
                <div className="summary-card">
                  <span>积分</span>
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
              <div className="compact-list">
                {mySolves.slice(0, 8).map((item) => (
                  <div className="row-card" key={`rank-solve-${item.id}`}>
                    <strong>{item.challenge_title}</strong>
                    <span>{item.awarded_points} pts</span>
                    <span>{formatDateTime(item.solved_at)}</span>
                  </div>
                ))}
                {mySolves.length === 0 ? <div className="empty-state small">你还没有解出任何题目。</div> : null}
              </div>
            </>
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
          {adminChallengesLoading ? <div className="empty-state">正在读取后台题目目录…</div> : null}
          <div className="challenge-card-list">
            {adminChallenges.map((item) => (
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
                  {item.visible ? <span className="badge badge-solid">Visible</span> : <span className="badge">Hidden</span>}
                  {item.dynamic_enabled ? <span className="badge badge-accent">Dynamic</span> : null}
                </div>
              </button>
            ))}
            {adminChallenges.length === 0 && !adminChallengesLoading ? <div className="empty-state">当前还没有题目。</div> : null}
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
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, flag_type: event.target.value }))}
                    value={adminChallengeDraft.flag_type}
                  />
                </label>
                <label className="field">
                  <span>Flag</span>
                  <input
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, flag_value: event.target.value }))}
                    value={adminChallengeDraft.flag_value}
                  />
                </label>
                <label className="toggle-field">
                  <input
                    checked={adminChallengeDraft.visible}
                    onChange={(event) => setAdminChallengeDraft((current) => ({ ...current, visible: event.target.checked }))}
                    type="checkbox"
                  />
                  <span>公开显示</span>
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
        <Panel eyebrow="Announcement Feed" title="公告列表" subtitle="后台数据包含已发布与草稿状态。">
          <NoticeBanner notice={adminAnnouncementsNotice} />
          {adminAnnouncementsLoading ? <div className="empty-state">正在读取后台公告…</div> : null}
          <div className="card-list">
            {adminAnnouncements.map((item) => (
              <article className="entry-card" key={item.id}>
                <div className="entry-head">
                  <strong>{item.title}</strong>
                  <span>{formatDateTime(item.published_at)}</span>
                </div>
                <p>{item.content}</p>
                <div className="badge-row">
                  {item.pinned ? <span className="badge badge-solid">Pinned</span> : null}
                  <span className="badge">{item.published ? 'Published' : 'Draft'}</span>
                </div>
              </article>
            ))}
            {adminAnnouncements.length === 0 && !adminAnnouncementsLoading ? <div className="empty-state">当前没有后台公告。</div> : null}
          </div>
        </Panel>

        <Panel eyebrow="Compose" title="发布公告" subtitle={canWriteAnnouncements ? '表单直接提交到后台创建接口。' : '当前账号没有公告写权限，仅可查看列表。'}>
          {canWriteAnnouncements ? (
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
              <button className="primary-button" disabled={announcementSubmitting} type="submit">
                {announcementSubmitting ? '提交中…' : '写入公告'}
              </button>
            </form>
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
          eyebrow="Admin Console"
          title="后台工作区"
          subtitle="按后端实际权限拆分。管理员可写题目、公告和用户；ops 仅展示允许的运维与只读模块。"
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
      <header className="topbar">
        <div className="brand-block">
          <div className="brand-mark">CTF</div>
          <div>
            <p className="eyebrow">Recruit Platform</p>
            <h1>Contest Workspace</h1>
          </div>
        </div>

        <nav className="main-nav" aria-label="Primary">
          {views
            .filter((item) => item.id !== 'admin' || canAccessAdmin)
            .map((item) => (
              <button
                className={view === item.id ? 'nav-pill active' : 'nav-pill'}
                key={item.id}
                onClick={() => setView(item.id)}
                type="button"
              >
                <span>{item.label}</span>
                <small>{item.note}</small>
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

      <main className="page-shell">
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
