import React, { useCallback, useEffect, useMemo, useState } from 'react'

import {
  api,
  type AuthResponse,
  type AuthUser,
  type ContestInfo,
  type ContestPhase,
  type PublicAnnouncement,
  type PublicChallengeSummary,
  type RuntimeInstance,
  type ScoreboardEntry,
  type SubmissionResult,
  type UserSolve,
  type UserSubmission,
} from '../api'
import { NoticeBanner } from './components/NoticeBanner'
import { describeError, errorToNotice, isUnauthorized, type Notice } from './utils/errors'

import brandMark from '../assets/yulin-long.svg'

type View = 'briefing' | 'board' | 'scoreboard' | 'me'

const TOKEN_STORAGE_KEY = 'ctf.frontend.token'

const views: Array<{ id: View; label: string; gatedByPhase?: keyof ContestPhase }> = [
  { id: 'briefing', label: '概览' },
  { id: 'board', label: '题目', gatedByPhase: 'challenge_list_visible' },
  { id: 'scoreboard', label: '排行榜', gatedByPhase: 'scoreboard_visible' },
  { id: 'me', label: '我的', gatedByPhase: 'challenge_list_visible' },
]

function clamp(n: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, n))
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const idx = clamp(Math.floor(Math.log(bytes) / Math.log(1024)), 0, units.length - 1)
  const value = bytes / 1024 ** idx
  return `${value.toFixed(value >= 10 || idx === 0 ? 0 : 1)} ${units[idx]}`
}

function parseRfc3339(input?: string | null): Date | null {
  if (!input) return null
  const date = new Date(input)
  return Number.isNaN(date.getTime()) ? null : date
}

function safeNow(): Date {
  const date = new Date()
  return Number.isNaN(date.getTime()) ? new Date(0) : date
}

function makeSolveTrend(items: Array<{ solved_at: string }>, buckets = 24): { counts: number[]; normalized: number[] } | null {
  if (!items.length) return null
  const timestamps = items
    .map((item) => parseRfc3339(item.solved_at))
    .filter((value): value is Date => Boolean(value))
    .map((value) => value.getTime())
    .sort((a, b) => a - b)

  if (timestamps.length < 2) return null

  const min = timestamps[0]
  const max = timestamps[timestamps.length - 1]
  const span = Math.max(1, max - min)
  const counts = Array.from({ length: buckets }, () => 0)
  for (const t of timestamps) {
    const idx = clamp(Math.floor(((t - min) / span) * (buckets - 1)), 0, buckets - 1)
    counts[idx] += 1
  }
  const peak = Math.max(...counts)
  const normalized = counts.map((value) => (peak ? value / peak : 0))
  return { counts, normalized }
}

function formatRelative(date: Date, now = safeNow()): string {
  const diffMs = date.getTime() - now.getTime()
  const diffSec = Math.round(diffMs / 1000)
  const abs = Math.abs(diffSec)
  if (abs < 30) return diffSec >= 0 ? 'in a moment' : 'just now'
  const mins = Math.round(abs / 60)
  if (mins < 60) return diffSec >= 0 ? `in ${mins}m` : `${mins}m ago`
  const hours = Math.round(mins / 60)
  if (hours < 48) return diffSec >= 0 ? `in ${hours}h` : `${hours}h ago`
  const days = Math.round(hours / 24)
  return diffSec >= 0 ? `in ${days}d` : `${days}d ago`
}

function formatDateTime(date: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

function formatDateTimeCompact(date: Date): string {
  return new Intl.DateTimeFormat(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date)
}

function resolveVisibleViews(phase: ContestPhase | null): Array<{ id: View; label: string; disabledReason?: string }> {
  return views.map((item) => {
    const gate = item.gatedByPhase
    if (!gate) return item
    const allowed = Boolean(phase?.[gate])
    return allowed ? item : { ...item, disabledReason: '未开放' }
  })
}

function isViewAccessible(view: View, phase: ContestPhase | null): boolean {
  const entry = views.find((item) => item.id === view)
  if (!entry?.gatedByPhase) return true
  return Boolean(phase?.[entry.gatedByPhase])
}

function pickInitialView(phase: ContestPhase | null): View {
  if (phase?.challenge_list_visible) return 'board'
  return 'briefing'
}

function DifficultyBadge({ difficulty }: { difficulty: string }): React.JSX.Element {
  const cls = difficulty === 'easy' ? 'difficulty-easy' : difficulty === 'hard' ? 'difficulty-hard' : 'difficulty-normal'
  const label = difficulty === 'easy' ? 'Easy' : difficulty === 'hard' ? 'Hard' : 'Normal'
  return <span className={`badge ${cls}`}>{label}</span>
}

function PillButton(props: {
  active?: boolean
  disabled?: boolean
  label: string
  onClick?: () => void
  title?: string
}): React.JSX.Element {
  return (
    <button
      className={`nav-pill ${props.active ? 'active' : ''}`}
      type="button"
      disabled={props.disabled}
      aria-current={props.active ? 'page' : undefined}
      onClick={props.onClick}
      title={props.title}
    >
      {props.label}
    </button>
  )
}

function TextInput(props: {
  label: string
  value: string
  onChange: (value: string) => void
  placeholder?: string
  type?: 'text' | 'password' | 'email'
  autoComplete?: string
  helpText?: string
  errorText?: string
}): React.JSX.Element {
  const id = React.useId()
  return (
    <label className="field" htmlFor={id}>
      <span>{props.label}</span>
      <input
        id={id}
        type={props.type ?? 'text'}
        value={props.value}
        onChange={(event) => props.onChange(event.target.value)}
        placeholder={props.placeholder}
        autoComplete={props.autoComplete}
        aria-invalid={props.errorText ? true : undefined}
        aria-describedby={props.helpText || props.errorText ? `${id}-help` : undefined}
      />
      {props.helpText || props.errorText ? (
        <small id={`${id}-help`} className="hint-text" style={{ color: props.errorText ? 'var(--danger)' : undefined }}>
          {props.errorText ?? props.helpText}
        </small>
      ) : null}
    </label>
  )
}

function SectionHeader(props: { eyebrow: string; title: string; subtitle?: string; children?: React.ReactNode }): React.JSX.Element {
  return (
    <header className="panel-head">
      <div>
        <p className="eyebrow">{props.eyebrow}</p>
        <h2>{props.title}</h2>
        {props.subtitle ? <p className="panel-subtitle">{props.subtitle}</p> : null}
      </div>
      {props.children ? <div className="inline-actions">{props.children}</div> : null}
    </header>
  )
}

function AnnouncementList({ items, loading }: { items: PublicAnnouncement[]; loading: boolean }): React.JSX.Element {
  if (loading) {
    return <div className="empty-state">加载公告中…</div>
  }
  if (!items.length) {
    return <div className="empty-state">暂无公告</div>
  }
  return (
    <div className="compact-list announcement-compact-list">
      {items.map((item) => (
        <article key={item.id} className="entry-card announcement-row">
          <div className="badge-row">
            {item.pinned ? <span className="badge badge-accent">Pinned</span> : null}
            {item.published_at ? <span className="badge">{formatDateTime(parseRfc3339(item.published_at) ?? safeNow())}</span> : null}
          </div>
          <strong>{item.title}</strong>
          <p>{item.content}</p>
        </article>
      ))}
    </div>
  )
}

type DemoChallengeDetail = {
  id: number
  slug: string
  title: string
  category: string
  points: number
  difficulty: string
  description: string
  dynamic: boolean
  attachments: Array<{ id: number; filename: string; size_bytes: number; content_type: string }>
}

export function App(): React.JSX.Element {
  const [theme, setTheme] = useState<'dark' | 'light'>(() => {
    const stored = window.localStorage.getItem('ctf.frontend.theme')
    return stored === 'light' ? 'light' : 'dark'
  })

  const [token, setToken] = useState<string>(() => window.localStorage.getItem(TOKEN_STORAGE_KEY) ?? '')
  const [authUser, setAuthUser] = useState<AuthUser | null>(null)
  const [sessionLoading, setSessionLoading] = useState(Boolean(token))

  const [contestInfo, setContestInfo] = useState<ContestInfo | null>(null)
  const [contestPhase, setContestPhase] = useState<ContestPhase | null>(null)

  const [view, setView] = useState<View>(() => pickInitialView(null))
  const visibleViews = useMemo(() => resolveVisibleViews(contestPhase), [contestPhase])

  const [publicNotice, setPublicNotice] = useState<Notice | null>(null)
  const [authNotice, setAuthNotice] = useState<Notice | null>(null)

  const [announcements, setAnnouncements] = useState<PublicAnnouncement[]>([])
  const [announcementsLoading, setAnnouncementsLoading] = useState(false)

  const [challenges, setChallenges] = useState<PublicChallengeSummary[]>([])
  const [challengesLoading, setChallengesLoading] = useState(false)
  const [challengeFilter, setChallengeFilter] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<string>('')
  const [activeChallengeID, setActiveChallengeID] = useState<string>('')
  const [activeChallenge, setActiveChallenge] = useState<DemoChallengeDetail | null>(null)
  const [challengeLoading, setChallengeLoading] = useState(false)

  const [flag, setFlag] = useState('')
  const [submissionResult, setSubmissionResult] = useState<SubmissionResult | null>(null)
  const [submissionLoading, setSubmissionLoading] = useState(false)

  const [instance, setInstance] = useState<RuntimeInstance | null>(null)
  const [instanceLoading, setInstanceLoading] = useState(false)

  const [scoreboard, setScoreboard] = useState<ScoreboardEntry[]>([])
  const [scoreboardLoading, setScoreboardLoading] = useState(false)

  const [scoreboardLeadersTopN, setScoreboardLeadersTopN] = useState<3 | 5 | 10>(5)

  const [mySolves, setMySolves] = useState<UserSolve[]>([])
  const [mySubmissions, setMySubmissions] = useState<UserSubmission[]>([])
  const [myLoading, setMyLoading] = useState(false)

  const clearSession = useCallback((message?: string) => {
    setToken('')
    setAuthUser(null)
    setSubmissionResult(null)
    setInstance(null)
    if (message) {
      setAuthNotice({ tone: 'neutral', text: message })
    }
  }, [])

  const guardedNotice = useCallback(
    (error: unknown, fallback: string): Notice => {
      if (isUnauthorized(error)) {
        clearSession('登录态已失效，请重新登录。')
        return { tone: 'danger', text: '登录态已失效，请重新登录。' }
      }
      return errorToNotice(error, fallback)
    },
    [clearSession],
  )

  useEffect(() => {
    window.localStorage.setItem(TOKEN_STORAGE_KEY, token)
    if (!token) window.localStorage.removeItem(TOKEN_STORAGE_KEY)
  }, [token])

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    window.localStorage.setItem('ctf.frontend.theme', theme)
  }, [theme])

  useEffect(() => {
    let active = true
    setPublicNotice(null)
    void api
      .contest()
      .then((response) => {
        if (!active) return
        setContestInfo(response.contest)
        setContestPhase(response.phase)
        setView((current) => {
          if (isViewAccessible(current, response.phase)) return current
          return pickInitialView(response.phase)
        })
      })
      .catch((error) => {
        if (!active) return
        setPublicNotice({ tone: 'danger', text: describeError(error, '公开数据加载失败。') })
      })
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    if (!token) {
      setAuthUser(null)
      setSessionLoading(false)
      return
    }

    let active = true
    setSessionLoading(true)
    void api
      .me(token)
      .then((response) => {
        if (!active) return
        setAuthUser(response.user)
      })
      .catch((error) => {
        if (!active) return
        clearSession(describeError(error, '登录态已失效，请重新登录。'))
      })
      .finally(() => {
        if (active) setSessionLoading(false)
      })

    return () => {
      active = false
    }
  }, [clearSession, token])

  useEffect(() => {
    if (!contestPhase?.announcement_visible) {
      setAnnouncements([])
      return
    }
    let active = true
    setAnnouncementsLoading(true)
    void api
      .announcements()
      .then((response) => {
        if (!active) return
        setAnnouncements(response.items)
      })
      .catch((error) => {
        if (!active) return
        setPublicNotice({ tone: 'danger', text: describeError(error, '公告加载失败。') })
      })
      .finally(() => {
        if (active) setAnnouncementsLoading(false)
      })
    return () => {
      active = false
    }
  }, [contestPhase?.announcement_visible])

  useEffect(() => {
    if (!contestPhase?.challenge_list_visible) {
      setChallenges([])
      return
    }
    let active = true
    setChallengesLoading(true)
    void api
      .challenges()
      .then((response) => {
        if (!active) return
        setChallenges(response.items)
      })
      .catch((error) => {
        if (!active) return
        setPublicNotice({ tone: 'danger', text: describeError(error, '题目列表加载失败。') })
      })
      .finally(() => {
        if (active) setChallengesLoading(false)
      })
    return () => {
      active = false
    }
  }, [contestPhase?.challenge_list_visible])

  useEffect(() => {
    if (activeChallengeID) return
    if (!challenges.length) return
    setActiveChallengeID(challenges[0].id)
  }, [activeChallengeID, challenges])

  useEffect(() => {
    if (!activeChallengeID) {
      setActiveChallenge(null)
      return
    }
    if (!contestPhase?.challenge_detail_visible) {
      setActiveChallenge(null)
      return
    }
    let active = true
    setChallengeLoading(true)
    setSubmissionResult(null)
    setInstance(null)
    void api
      .challenge(activeChallengeID)
      .then((response) => {
        if (!active) return
        const challenge = response.challenge
        setActiveChallenge({
          id: challenge.id,
          slug: challenge.slug,
          title: challenge.title,
          category: challenge.category,
          points: challenge.points,
          difficulty: challenge.difficulty,
          description: challenge.description,
          dynamic: challenge.dynamic,
          attachments: challenge.attachments.map((item) => ({
            id: item.id,
            filename: item.filename,
            size_bytes: item.size_bytes,
            content_type: item.content_type,
          })),
        })
      })
      .catch((error) => {
        if (!active) return
        setAuthNotice(guardedNotice(error, '题目详情加载失败。'))
      })
      .finally(() => {
        if (active) setChallengeLoading(false)
      })
    return () => {
      active = false
    }
  }, [activeChallengeID, contestPhase?.challenge_detail_visible, guardedNotice])

  const filteredChallenges = useMemo(() => {
    const query = challengeFilter.trim().toLowerCase()
    const selectedCategory = categoryFilter.trim().toLowerCase()

    return challenges.filter((item) => {
      if (selectedCategory && item.category.toLowerCase() !== selectedCategory) {
        return false
      }
      if (!query) return true
      return item.title.toLowerCase().includes(query) || item.slug.toLowerCase().includes(query) || item.difficulty.toLowerCase().includes(query)
    })
  }, [categoryFilter, challengeFilter, challenges])

  const availableCategories = useMemo(() => {
    const unique = new Set<string>()
    for (const item of challenges) {
      if (item.category.trim()) unique.add(item.category)
    }
    return Array.from(unique).sort((a, b) => a.localeCompare(b))
  }, [challenges])

  const submitFlag = useCallback(async () => {
    if (!token || !activeChallenge) {
      setAuthNotice({ tone: 'neutral', text: '请先登录并选择题目。' })
      return
    }
    if (!contestPhase?.submission_allowed) {
      setAuthNotice({ tone: 'neutral', text: '当前阶段不允许提交 Flag。' })
      return
    }
    if (!flag.trim()) {
      setAuthNotice({ tone: 'neutral', text: '请输入 Flag。' })
      return
    }
    setSubmissionLoading(true)
    setSubmissionResult(null)
    setAuthNotice(null)
    try {
      const result = await api.submitFlag(token, String(activeChallenge.id), flag.trim())
      setSubmissionResult(result)
    } catch (error) {
      setAuthNotice(guardedNotice(error, '提交失败。'))
    } finally {
      setSubmissionLoading(false)
    }
  }, [activeChallenge, contestPhase?.submission_allowed, flag, guardedNotice, token])

  const loadInstance = useCallback(async () => {
    if (!token || !activeChallenge) return
    if (!contestPhase?.runtime_allowed) {
      setAuthNotice({ tone: 'neutral', text: '当前阶段未开放动态实例。' })
      return
    }
    setInstanceLoading(true)
    setAuthNotice(null)
    try {
      const next = await api.getInstance(token, String(activeChallenge.id))
      setInstance(next)
    } catch (error) {
      setAuthNotice(guardedNotice(error, '实例查询失败。'))
    } finally {
      setInstanceLoading(false)
    }
  }, [activeChallenge, contestPhase?.runtime_allowed, guardedNotice, token])

  const startInstance = useCallback(async () => {
    if (!token || !activeChallenge) {
      setAuthNotice({ tone: 'neutral', text: '请先登录并选择题目。' })
      return
    }
    if (!contestPhase?.runtime_allowed) {
      setAuthNotice({ tone: 'neutral', text: '当前阶段未开放动态实例。' })
      return
    }
    setInstanceLoading(true)
    setAuthNotice(null)
    try {
      const next = await api.startInstance(token, String(activeChallenge.id))
      setInstance(next)
    } catch (error) {
      setAuthNotice(guardedNotice(error, '实例创建失败。'))
    } finally {
      setInstanceLoading(false)
    }
  }, [activeChallenge, contestPhase?.runtime_allowed, guardedNotice, token])

  const renewInstance = useCallback(async () => {
    if (!token || !activeChallenge) return
    setInstanceLoading(true)
    setAuthNotice(null)
    try {
      const next = await api.renewInstance(token, String(activeChallenge.id))
      setInstance(next)
    } catch (error) {
      setAuthNotice(guardedNotice(error, '实例续期失败。'))
    } finally {
      setInstanceLoading(false)
    }
  }, [activeChallenge, guardedNotice, token])

  const terminateInstance = useCallback(async () => {
    if (!token || !activeChallenge) return
    setInstanceLoading(true)
    setAuthNotice(null)
    try {
      const next = await api.deleteInstance(token, String(activeChallenge.id))
      setInstance(next)
    } catch (error) {
      setAuthNotice(guardedNotice(error, '实例删除失败。'))
    } finally {
      setInstanceLoading(false)
    }
  }, [activeChallenge, guardedNotice, token])

  const loadScoreboard = useCallback(async () => {
    if (!contestPhase?.scoreboard_visible) {
      setScoreboard([])
      return
    }
    setScoreboardLoading(true)
    setPublicNotice(null)
    try {
      const response = await api.scoreboard()
      setScoreboard(response.items)
    } catch (error) {
      setPublicNotice({ tone: 'danger', text: describeError(error, '排行榜加载失败。') })
    } finally {
      setScoreboardLoading(false)
    }
  }, [contestPhase?.scoreboard_visible])

  const loadMyProgress = useCallback(async () => {
    if (!token) return
    setMyLoading(true)
    setAuthNotice(null)
    try {
      const [submissions, solves] = await Promise.all([api.mySubmissions(token), api.mySolves(token)])
      setMySubmissions(submissions.items)
      setMySolves(solves.items)
    } catch (error) {
      setAuthNotice(guardedNotice(error, '个人记录加载失败。'))
    } finally {
      setMyLoading(false)
    }
  }, [guardedNotice, token])

  useEffect(() => {
    if (view === 'scoreboard') {
      void loadScoreboard()
    }
  }, [loadScoreboard, view])

  const scoreboardByRank = useMemo(() => {
    return [...scoreboard].sort((a, b) => a.rank - b.rank)
  }, [scoreboard])

  const scoreboardTop3 = useMemo(() => scoreboardByRank.slice(0, 3), [scoreboardByRank])

  const scoreboardTotalSolves = useMemo(() => {
    return scoreboard.reduce((sum, entry) => sum + entry.solves.length, 0)
  }, [scoreboard])

  const scoreboardLastSolveAt = useMemo(() => {
    let latest: string | null = null
    for (const entry of scoreboard) {
      if (!entry.last_solve_at) continue
      if (!latest) {
        latest = entry.last_solve_at
        continue
      }
      const curr = parseRfc3339(entry.last_solve_at)?.getTime() ?? 0
      const prev = parseRfc3339(latest)?.getTime() ?? 0
      if (curr > prev) latest = entry.last_solve_at
    }
    return latest
  }, [scoreboard])

  const myScoreboardEntry = useMemo(() => {
    if (!authUser) return null
    return scoreboardByRank.find((entry) => entry.user_id === authUser.id) ?? null
  }, [authUser, scoreboardByRank])

  useEffect(() => {
    if (view === 'me' && token) {
      void loadMyProgress()
    }
  }, [loadMyProgress, token, view])

  const [authTab, setAuthTab] = useState<'login' | 'register'>('login')
  const [loginIdentifier, setLoginIdentifier] = useState('')
  const [loginPassword, setLoginPassword] = useState('')
  const [registerUsername, setRegisterUsername] = useState('')
  const [registerEmail, setRegisterEmail] = useState('')
  const [registerDisplayName, setRegisterDisplayName] = useState('')
  const [registerPassword, setRegisterPassword] = useState('')
  const [authLoading, setAuthLoading] = useState(false)

  const loginIdentifierError = useMemo(() => {
    if (!loginIdentifier.trim()) return null
    return null
  }, [loginIdentifier])

  const loginPasswordError = useMemo(() => {
    if (!loginPassword) return null
    return null
  }, [loginPassword])

  const registerUsernameError = useMemo(() => {
    if (!registerUsername.trim()) return null
    if (!/^[a-zA-Z0-9_\-.]{3,32}$/.test(registerUsername.trim())) {
      return '仅允许字母/数字/下划线/点/短横线，长度 3-32。'
    }
    return null
  }, [registerUsername])

  const registerEmailError = useMemo(() => {
    if (!registerEmail.trim()) return null
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(registerEmail.trim()) ? null : '邮箱格式看起来不正确。'
  }, [registerEmail])

  const registerPasswordError = useMemo(() => {
    if (!registerPassword) return null
    if (registerPassword.length < 8) return '至少 8 位。'
    return null
  }, [registerPassword])

  const applyAuthResponse = useCallback((response: AuthResponse) => {
    setToken(response.token)
    setAuthUser(response.user)
    setAuthNotice({ tone: 'ok', text: `欢迎，${response.user.display_name || response.user.username}。` })
  }, [])

  const performLogin = useCallback(async () => {
    if (!loginIdentifier.trim() || !loginPassword) {
      setAuthNotice({ tone: 'neutral', text: '请填写完整登录信息。' })
      return
    }
    setAuthLoading(true)
    setAuthNotice(null)
    try {
      const response = await api.login(loginIdentifier.trim(), loginPassword)
      applyAuthResponse(response)
    } catch (error) {
      setAuthNotice(errorToNotice(error, '登录失败。'))
    } finally {
      setAuthLoading(false)
    }
  }, [applyAuthResponse, loginIdentifier, loginPassword])

  const performRegister = useCallback(async () => {
    if (!contestPhase?.registration_allowed) {
      setAuthNotice({ tone: 'neutral', text: '当前阶段未开放注册入口。' })
      return
    }
    if (!registerUsername.trim() || !registerEmail.trim() || !registerDisplayName.trim() || !registerPassword) {
      setAuthNotice({ tone: 'neutral', text: '请填写完整注册信息。' })
      return
    }
    if (registerUsernameError || registerEmailError || registerPasswordError) {
      setAuthNotice({ tone: 'neutral', text: '请先修正表单中的提示。' })
      return
    }
    setAuthLoading(true)
    setAuthNotice(null)
    try {
      const response = await api.register({
        username: registerUsername.trim(),
        email: registerEmail.trim(),
        display_name: registerDisplayName.trim(),
        password: registerPassword,
      })
      applyAuthResponse(response)
      setAuthTab('login')
    } catch (error) {
      setAuthNotice(errorToNotice(error, '注册失败。'))
    } finally {
      setAuthLoading(false)
    }
  }, [applyAuthResponse, contestPhase?.registration_allowed, registerDisplayName, registerEmail, registerPassword, registerUsername])

  const contestStatusBadge = contestInfo?.status ? <span className="badge badge-accent">{contestInfo.status}</span> : null

  const contestWindow = useMemo(() => {
    const start = parseRfc3339(contestInfo?.starts_at)
    const end = parseRfc3339(contestInfo?.ends_at)
    if (!start && !end) return null
    const now = safeNow()

    const startText = start ? `${formatDateTime(start)} (${formatRelative(start, now)})` : ''
    const endText = end ? `${formatDateTime(end)} (${formatRelative(end, now)})` : ''
    const text = start && end ? `${startText} → ${endText}` : start ? startText : endText

    return <span className="badge">{text}</span>
  }, [contestInfo?.ends_at, contestInfo?.starts_at])

  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">
        跳到内容
      </a>

      <header className="topbar" aria-label="Top navigation">
        <div className="brand-block">
          <div className="brand-mark" aria-hidden="true">
            <img alt="" src={brandMark} />
          </div>
          <div className="brand-copy">
            <small>YulinSec CTF</small>
            <h1>{contestInfo?.title ?? 'CTF Recruit Platform'}</h1>
          </div>
        </div>

        <nav className="main-nav" aria-label="Primary">
          {visibleViews.map((item) => (
            <PillButton
              key={item.id}
              active={view === item.id}
              disabled={Boolean(item.disabledReason)}
              label={item.label}
              title={item.disabledReason}
              onClick={() => {
                if (item.disabledReason) {
                  setPublicNotice({ tone: 'neutral', text: `${item.label}：当前阶段未开放。` })
                  return
                }
                setView(item.id)
              }}
            />
          ))}
        </nav>

        <div className="session-block">
          <button
            className="ghost-button"
            type="button"
            onClick={() => setTheme((current) => (current === 'dark' ? 'light' : 'dark'))}
            title={theme === 'dark' ? '切换到日间模式' : '切换到夜间模式'}
          >
            {theme === 'dark' ? 'Light' : 'Dark'}
          </button>
          <div className={`user-chip ${authUser ? '' : 'ghost-chip'}`}>
            <span>{sessionLoading ? 'loading session…' : authUser ? authUser.display_name || authUser.username : 'Guest'}</span>
            <small>
              {sessionLoading ? 'checking token' : authUser ? authUser.role : 'not authenticated'}
              {contestInfo?.status ? ` · ${contestInfo.status}` : ''}
            </small>
          </div>
          {authUser ? (
            <button className="ghost-button danger-button" type="button" onClick={() => clearSession('已退出当前账号。')}>
              Logout
            </button>
          ) : (
            <button className="primary-button" type="button" onClick={() => setView('briefing')}>
              Login
            </button>
          )}
        </div>
      </header>

      <main className="page-shell" id="main-content">
        <NoticeBanner notice={publicNotice} />

        {view === 'briefing' ? (
          <section className="view-stack">
            <section className="panel panel-hero page-enter">
              <SectionHeader eyebrow="概览" title={contestInfo?.title ?? 'CTF Recruit Platform'} subtitle={contestPhase?.message ?? contestInfo?.description ?? '比赛信息加载中…'}>
                {contestStatusBadge}
              </SectionHeader>

              <div className="badge-row">
                {contestWindow}
                <span className="badge">
                  题目 {contestPhase?.challenge_list_visible ? '开放' : '关闭'} · 提交 {contestPhase?.submission_allowed ? '开放' : '关闭'} · 实例{' '}
                  {contestPhase?.runtime_allowed ? '开放' : '关闭'} · 榜单 {contestPhase?.scoreboard_visible ? '开放' : '关闭'}
                </span>
              </div>
            </section>

            <section className="panel announcement-subpanel page-enter page-enter-1">
              <SectionHeader eyebrow="公告" title="公告" subtitle={contestPhase?.announcement_visible ? undefined : '当前阶段未开放公告。'} />
              {contestPhase?.announcement_visible ? <AnnouncementList items={announcements} loading={announcementsLoading} /> : <div className="empty-state">公告未开放</div>}
            </section>

            <section className="panel page-enter page-enter-2">
              <SectionHeader eyebrow="账号" title={authUser ? '已登录' : '登录 / 注册'} subtitle={authUser ? undefined : '仅保留必要字段，避免打断做题。'} />
              <NoticeBanner notice={authNotice} />
              {authUser ? (
                <div className="wrap-actions">
                  <button className="primary-button" type="button" onClick={() => setView('board')} disabled={!contestPhase?.challenge_list_visible}>
                    去做题
                  </button>
                  <button className="ghost-button" type="button" onClick={() => setView('me')} disabled={!contestPhase?.challenge_list_visible}>
                    查看我的进度
                  </button>
                </div>
              ) : (
                <div className="auth-layout">
                  <div className="tab-strip">
                    <button className={`tab-pill ${authTab === 'login' ? 'active' : ''}`} type="button" onClick={() => setAuthTab('login')}>
                      登录
                    </button>
                    <button
                      className={`tab-pill ${authTab === 'register' ? 'active' : ''}`}
                      type="button"
                      onClick={() => setAuthTab('register')}
                      disabled={!contestPhase?.registration_allowed}
                      title={contestPhase?.registration_allowed ? undefined : '当前阶段未开放注册'}
                    >
                      注册
                    </button>
                  </div>

                  {authTab === 'login' ? (
                    <div className="panel small">
                      <div className="form-grid single-column">
                        <TextInput
                          label="用户名或邮箱"
                          value={loginIdentifier}
                          onChange={setLoginIdentifier}
                          placeholder="player / player@ctf.local"
                          autoComplete="username"
                          helpText="支持用户名或邮箱。"
                          errorText={loginIdentifierError ?? undefined}
                        />
                        <TextInput
                          label="密码"
                          type="password"
                          value={loginPassword}
                          onChange={setLoginPassword}
                          placeholder="••••••••"
                          autoComplete="current-password"
                          errorText={loginPasswordError ?? undefined}
                        />
                      </div>
                      <div className="form-footer" style={{ marginTop: 14 }}>
                        <button className="primary-button" type="button" disabled={authLoading} onClick={() => void performLogin()}>
                          {authLoading ? '登录中…' : '登录'}
                        </button>
                        <button className="ghost-button" type="button" onClick={() => setAuthTab('register')}>
                          去注册
                        </button>
                      </div>
                    </div>
                  ) : (
                    <div className="panel small">
                      <div className="form-grid">
                        <TextInput
                          label="用户名"
                          value={registerUsername}
                          onChange={setRegisterUsername}
                          placeholder="player"
                          autoComplete="username"
                          helpText="3-32 位，字母/数字/下划线/点/短横线。"
                          errorText={registerUsernameError ?? undefined}
                        />
                        <TextInput
                          label="邮箱"
                          type="email"
                          value={registerEmail}
                          onChange={setRegisterEmail}
                          placeholder="player@example.com"
                          autoComplete="email"
                          errorText={registerEmailError ?? undefined}
                        />
                        <div className="wide-field">
                          <TextInput
                            label="展示名"
                            value={registerDisplayName}
                            onChange={setRegisterDisplayName}
                            placeholder="Player"
                            autoComplete="nickname"
                            helpText="显示在排行榜和个人页面。"
                          />
                        </div>
                        <div className="wide-field">
                          <TextInput
                            label="密码"
                            type="password"
                            value={registerPassword}
                            onChange={setRegisterPassword}
                            placeholder="至少 8 位，建议包含大小写和符号"
                            autoComplete="new-password"
                            errorText={registerPasswordError ?? undefined}
                          />
                        </div>
                      </div>
                      <div className="form-footer" style={{ marginTop: 14 }}>
                        <button className="primary-button" type="button" disabled={authLoading} onClick={() => void performRegister()}>
                          {authLoading ? '提交中…' : '创建账号'}
                        </button>
                        <button className="ghost-button" type="button" onClick={() => setAuthTab('login')}>
                          去登录
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </section>
          </section>
        ) : null}

        {view === 'board' ? (
          <section className="player-focused-board page-enter">
            <div className="player-board-shell workspace-grid">
              <aside className="panel rail-panel player-side-panel">
                <SectionHeader eyebrow="题目" title="题目列表" subtitle={undefined} />
                <div className="board-list-toolbar">
                  <label className="field">
                    <span>搜索</span>
                    <input value={challengeFilter} onChange={(event) => setChallengeFilter(event.target.value)} placeholder="web / crypto / easy / welcome" />
                  </label>
                  <label className="field">
                    <span>分类</span>
                    <select value={categoryFilter} onChange={(event) => setCategoryFilter(event.target.value)}>
                      <option value="">全部</option>
                      {availableCategories.map((category) => (
                        <option key={category} value={category}>
                          {category}
                        </option>
                      ))}
                    </select>
                  </label>
                  <div className="board-filter-meta">{challengesLoading ? '加载中…' : `${filteredChallenges.length}/${challenges.length}`}</div>
                </div>

                <div className="challenge-card-list">
                  {challengesLoading ? <div className="empty-state">加载题目中…</div> : null}
                  {!challengesLoading && filteredChallenges.length === 0 ? <div className="empty-state">没有匹配的题目</div> : null}
                  {filteredChallenges.map((item) => (
                    <button
                      key={item.id}
                      type="button"
                      className={`challenge-card challenge-card-player ${activeChallengeID === item.id ? 'active' : ''}`}
                      onClick={() => setActiveChallengeID(item.id)}
                    >
                      <strong>{item.title}</strong>
                      <div className="challenge-card-subline">
                        <small>
                          {item.category} · <DifficultyBadge difficulty={item.difficulty} /> · {item.points} pts{item.dynamic ? ' · Dyn' : ''}
                        </small>
                        <small>{item.slug}</small>
                      </div>
                    </button>
                  ))}
                </div>
              </aside>

              <section className="content-stack">
                <section className="panel challenge-workspace-panel">
                  <SectionHeader
                    eyebrow="Workspace"
                    title={activeChallenge ? activeChallenge.title : challengeLoading ? '加载中…' : '选择一道题'}
                    subtitle={activeChallenge ? `${activeChallenge.category} · ${activeChallenge.points} pts · ${activeChallenge.dynamic ? 'dynamic' : 'static'}` : '先从左侧选题开始。'}
                  >
                    {activeChallenge ? <DifficultyBadge difficulty={activeChallenge.difficulty} /> : null}
                  </SectionHeader>

                  <NoticeBanner notice={authNotice} />

                  {contestPhase?.challenge_detail_visible ? (
                    <article className="statement-card player-statement-card">
                      <div className="statement-topline">
                        <span className="section-tag">Statement</span>
                        {activeChallenge ? <span className="badge">ID {activeChallenge.id}</span> : null}
                      </div>
                      <p className="statement-text">{activeChallenge ? activeChallenge.description : challengeLoading ? '加载中…' : '请选择题目。'}</p>

                      {activeChallenge && contestPhase?.attachment_visible && activeChallenge.attachments.length ? (
                        <div className="detail-stack" style={{ marginTop: 12 }}>
                          <strong>附件</strong>
                          <div className="attachment-list">
                            {activeChallenge.attachments.map((item) => (
                              <a
                                key={item.id}
                                className="attachment-row"
                                href={`/api/v1/challenges/${activeChallenge.id}/attachments/${item.id}`}
                                target="_blank"
                                rel="noreferrer"
                              >
                                <div style={{ display: 'grid', gap: 2 }}>
                                  <strong>{item.filename}</strong>
                                  <small className="hint-text">{item.content_type}</small>
                                </div>
                                <span className="badge">{formatBytes(item.size_bytes)}</span>
                              </a>
                            ))}
                          </div>
                        </div>
                      ) : null}
                    </article>
                  ) : (
                    <div className="empty-state">题目详情未开放</div>
                  )}

                  <section className="panel primary-action-panel" style={{ marginTop: 18 }}>
                    <SectionHeader eyebrow="Submit" title="提交 Flag" subtitle={contestPhase?.submission_allowed ? '提交后立即返回判定结果。' : '当前阶段关闭提交入口。'} />
                    {submissionResult ? (
                      <div className={`notice ${submissionResult.correct ? 'notice-success' : 'notice-danger'}`}>
                        <strong>{submissionResult.correct ? 'Accepted' : 'Wrong'}</strong>
                        <div style={{ marginTop: 6 }}>{submissionResult.message}</div>
                        <div style={{ marginTop: 6 }}>
                          <span className="badge">+{submissionResult.awarded_points} pts</span>
                          {submissionResult.solved ? <span className="badge badge-solid">Solved</span> : null}
                        </div>
                      </div>
                    ) : null}
                    <div className="primary-action-layout">
                      <div className="primary-action-intro single-action-grid">
                        <div className="primary-action-copy">
                          <TextInput
                            label="Flag"
                            value={flag}
                            onChange={setFlag}
                            placeholder="flag{...}"
                            autoComplete="off"
                          />
                          <div className="hint-text" style={{ marginTop: 10 }}>
                            建议：复制粘贴，避免手输错误。提交频率过高会触发限流。
                          </div>
                        </div>
                      </div>

                      <div className="wrap-actions">
                        <button className="primary-button" type="button" disabled={!contestPhase?.submission_allowed || submissionLoading} onClick={() => void submitFlag()}>
                          {submissionLoading ? '提交中…' : '提交'}
                        </button>
                        <button className="ghost-button" type="button" onClick={() => setFlag('')} disabled={submissionLoading}>
                          清空
                        </button>
                      </div>
                    </div>
                  </section>

                  {activeChallenge && activeChallenge.dynamic ? (
                    <section className="panel compact-runtime-panel" style={{ marginTop: 18 }}>
                      <SectionHeader
                        eyebrow="Runtime"
                        title="动态实例"
                        subtitle={contestPhase?.runtime_allowed ? '一键启动，复制访问地址；支持续期与删除。' : '当前阶段未开放动态实例。'}
                      />

                      <div className="runtime-focus-stack">
                        <div className="runtime-context-card">
                          <div>
                            <span className="eyebrow">当前题目</span>
                            <h3 style={{ margin: '10px 0 0' }}>{activeChallenge.title}</h3>
                            <p className="panel-subtitle" style={{ margin: '10px 0 0' }}>
                              若实例不可用，请先确认阶段、题目并发、冷却时间或端口池。
                            </p>
                          </div>
                          <div className="badge-row" style={{ justifyContent: 'flex-end' }}>
                            <span className="badge">{instance?.status ?? 'unknown'}</span>
                            {instance?.host_port ? <span className="badge badge-accent">:{instance.host_port}</span> : null}
                          </div>
                        </div>

                        <div className="runtime-action-strip">
                          <button className="primary-button" type="button" disabled={!contestPhase?.runtime_allowed || instanceLoading} onClick={() => void startInstance()}>
                            {instanceLoading ? '启动中…' : '启动/复用'}
                          </button>
                          <button className="ghost-button" type="button" disabled={!contestPhase?.runtime_allowed || instanceLoading} onClick={() => void loadInstance()}>
                            刷新状态
                          </button>
                          <button className="ghost-button" type="button" disabled={!contestPhase?.runtime_allowed || instanceLoading} onClick={() => void renewInstance()}>
                            续期
                          </button>
                          <button className="ghost-button danger-button" type="button" disabled={!contestPhase?.runtime_allowed || instanceLoading} onClick={() => void terminateInstance()}>
                            删除
                          </button>
                        </div>

                        <div className="compact-runtime-metrics card-list">
                          <div className={`runtime-metric ${instance?.status === 'running' ? 'runtime-availability-row' : ''}`}>
                            <span className="eyebrow">Access</span>
                            <strong>{instance?.access_url ? 'Ready' : '—'}</strong>
                            <div className="hint-text" style={{ marginTop: 6 }}>
                              {instance?.access_url ? (
                                <a className="link-button" href={instance.access_url} target="_blank" rel="noreferrer">
                                  打开
                                </a>
                              ) : (
                                '启动后提供访问地址'
                              )}
                            </div>
                          </div>
                          <div className="runtime-metric">
                            <span className="eyebrow">TTL</span>
                            <strong>
                              {instance?.expires_at
                                ? formatRelative(parseRfc3339(instance.expires_at) ?? safeNow())
                                : instanceLoading
                                  ? '…'
                                  : '—'}
                            </strong>
                            <small className="hint-text">expires at {instance?.expires_at ?? '—'}</small>
                          </div>
                          <div className="runtime-metric">
                            <span className="eyebrow">Renew</span>
                            <strong>{instance?.renew_count ?? 0}</strong>
                            <small className="hint-text">续期次数</small>
                          </div>
                        </div>

                        <details className="detail-row">
                          <summary style={{ cursor: 'pointer' }}>
                            <strong>常见失败原因</strong>
                          </summary>
                          <div className="hint-text" style={{ marginTop: 8 }}>
                            <div>· instance_capacity_reached：题目并发上限</div>
                            <div>· instance_cooldown_active：用户冷却中</div>
                            <div>· instance_port_exhausted：端口池耗尽</div>
                          </div>
                        </details>
                      </div>
                    </section>
                  ) : null}
                </section>
              </section>
            </div>
          </section>
        ) : null}

        {view === 'scoreboard' ? (
          <section className="view-stack page-enter">
            <section className="panel scoreboard-stage-hero">
              <SectionHeader eyebrow="Scoreboard" title="排行榜" subtitle={contestPhase?.scoreboard_visible ? '默认展示 Top 50；点击展开查看解题细节。' : '当前阶段未公开排行榜。'}>
                <button className="ghost-button" type="button" onClick={() => void loadScoreboard()} disabled={!contestPhase?.scoreboard_visible || scoreboardLoading}>
                  刷新
                </button>
              </SectionHeader>

              {!contestPhase?.scoreboard_visible ? <div className="empty-state">排行榜未开放</div> : null}
              {contestPhase?.scoreboard_visible ? (
                <div className="board-summary-grid" style={{ marginTop: 12 }}>
                  <div className="detail-row">
                    <div className="scoreboard-trend-head">
                      <strong>Top {scoreboardLeadersTopN} 得分趋势</strong>
                      <div className="badge-row">
                        <PillButton active={scoreboardLeadersTopN === 3} label="Top 3" onClick={() => setScoreboardLeadersTopN(3)} />
                        <PillButton active={scoreboardLeadersTopN === 5} label="Top 5" onClick={() => setScoreboardLeadersTopN(5)} />
                        <PillButton active={scoreboardLeadersTopN === 10} label="Top 10" onClick={() => setScoreboardLeadersTopN(10)} />
                      </div>
                    </div>

                    {scoreboardLoading && scoreboard.length === 0 ? <div className="empty-state small">加载趋势数据中…</div> : null}
                    {!scoreboardLoading && scoreboard.length === 0 ? <div className="empty-state small">暂无趋势数据</div> : null}
                    {scoreboard.length ? <ScoreboardLeadersChart entries={scoreboardByRank} topN={scoreboardLeadersTopN} /> : null}
                    {scoreboard.length ? <div className="hint-text">按解题时间累计得分。</div> : null}
                  </div>

                  <div className="scoreboard-kpi-grid">
                    <div className="runtime-metric">
                      <span className="eyebrow">Leader</span>
                      <strong>
                        {scoreboardTop3[0]
                          ? `#1 ${scoreboardTop3[0].display_name || scoreboardTop3[0].username}`
                          : scoreboardLoading
                            ? '…'
                            : '—'}
                      </strong>
                      <small className="hint-text">
                        {scoreboardTop3[0] ? `${scoreboardTop3[0].score} pts · ${scoreboardTop3[0].solves.length} solves` : '当前暂无领先者'}
                      </small>
                    </div>

                    <div className="runtime-metric">
                      <span className="eyebrow">My Rank</span>
                      <strong>
                        {myScoreboardEntry
                          ? `#${myScoreboardEntry.rank}`
                          : authUser
                            ? scoreboardLoading
                              ? '…'
                              : '—'
                            : 'Login'}
                      </strong>
                      <small className="hint-text">
                        {myScoreboardEntry
                          ? `${myScoreboardEntry.score} pts · ${myScoreboardEntry.solves.length} solves`
                          : authUser
                            ? '未上榜或暂无得分'
                            : '登录后显示'}
                      </small>
                    </div>

                    <div className="runtime-metric">
                      <span className="eyebrow">Players</span>
                      <strong>{scoreboardLoading && scoreboard.length === 0 ? '…' : scoreboard.length}</strong>
                      <small className="hint-text">total solves {scoreboardTotalSolves}</small>
                    </div>

                    <div className="runtime-metric">
                      <span className="eyebrow">Latest</span>
                      <strong>
                        {scoreboardLastSolveAt
                          ? formatRelative(parseRfc3339(scoreboardLastSolveAt) ?? safeNow())
                          : scoreboardLoading
                            ? '…'
                            : '—'}
                      </strong>
                      <small className="hint-text">
                        {scoreboardLastSolveAt ? formatDateTime(parseRfc3339(scoreboardLastSolveAt) ?? safeNow()) : '暂无记录'}
                      </small>
                    </div>
                  </div>
                </div>
              ) : null}
            </section>

            {contestPhase?.scoreboard_visible ? (
              <section className="panel">
                <SectionHeader eyebrow="Top" title="排名" subtitle="点击条目展开 solves。" />
                {scoreboardLoading ? <div className="empty-state">加载排行榜中…</div> : null}
                {!scoreboardLoading && scoreboard.length === 0 ? <div className="empty-state">暂无数据</div> : null}
                {!scoreboardLoading && scoreboard.length ? (
                  <div className="scoreboard-table">
                    <div className="scoreboard-table-header">
                      <span>Rank</span>
                      <span>Player</span>
                      <span>Score</span>
                      <span>Solves</span>
                      <span>Last solve</span>
                      <span></span>
                    </div>
                    {scoreboard.map((entry) => (
                      <ScoreboardRow key={entry.user_id} entry={entry} currentUserID={authUser?.id ?? null} />
                    ))}
                  </div>
                ) : null}
              </section>
            ) : null}
          </section>
        ) : null}

        {view === 'me' ? (
          <section className="view-stack page-enter">
            <section className="panel player-hero">
              <SectionHeader eyebrow="My" title="我的进度" subtitle={authUser ? '复盘自己的提交和解题路径。' : '请先登录。'}>
                <button className="ghost-button" type="button" onClick={() => void loadMyProgress()} disabled={!token || myLoading}>
                  刷新
                </button>
              </SectionHeader>
              <NoticeBanner notice={authNotice} />
              {!authUser ? <div className="empty-state">未登录</div> : null}
              {authUser ? (
                <div className="badge-row">
                  <span className="badge">Solves {myLoading ? '…' : mySolves.length}</span>
                  <span className="badge">Submissions {myLoading ? '…' : mySubmissions.length}</span>
                  <span className="badge">Role {authUser.role}</span>
                </div>
              ) : null}
            </section>

            {authUser ? (
              <section className="panel">
                <SectionHeader eyebrow="Solves" title="解题记录" subtitle="按时间倒序。" />
                {myLoading ? <div className="empty-state">加载中…</div> : null}
                {!myLoading && mySolves.length === 0 ? <div className="empty-state">暂无解题记录</div> : null}
                {!myLoading && mySolves.length ? (
                  <div className="compact-list">
                    {mySolves.map((item) => (
                      <div key={item.id} className="entry-card">
                        <div className="badge-row">
                          <span className="badge">{item.category}</span>
                          <span className="badge badge-solid">+{item.awarded_points}</span>
                          <span className="badge">{formatDateTime(parseRfc3339(item.solved_at) ?? safeNow())}</span>
                        </div>
                        <strong>{item.challenge_title}</strong>
                        <p>{item.challenge_slug}</p>
                      </div>
                    ))}
                  </div>
                ) : null}
              </section>
            ) : null}

            {authUser ? (
              <section className="panel">
                <SectionHeader eyebrow="Submissions" title="提交记录" subtitle="用于自查：错在哪、什么时候开始对了。" />
                {myLoading ? <div className="empty-state">加载中…</div> : null}
                {!myLoading && mySubmissions.length === 0 ? <div className="empty-state">暂无提交记录</div> : null}
                {!myLoading && mySubmissions.length ? (
                  <div className="compact-list">
                    {mySubmissions.slice(0, 25).map((item) => (
                      <div key={item.id} className="entry-card">
                        <div className="badge-row">
                          <span className={`badge ${item.correct ? 'badge-solid' : ''}`}>{item.correct ? 'Correct' : 'Wrong'}</span>
                          <span className="badge">{item.category}</span>
                          <span className="badge">{formatDateTime(parseRfc3339(item.submitted_at) ?? safeNow())}</span>
                        </div>
                        <strong>{item.challenge_title}</strong>
                        <p>{item.challenge_slug}</p>
                      </div>
                    ))}
                  </div>
                ) : null}
              </section>
            ) : null}
          </section>
        ) : null}

        <footer style={{ marginTop: 16, color: 'var(--muted)', fontSize: 12, textAlign: 'center' }}>focus on solving challenges</footer>
      </main>
    </div>
  )
}

function ScoreboardRow(props: { entry: ScoreboardEntry; currentUserID: number | null }): React.JSX.Element {
  const [expanded, setExpanded] = useState(false)
  const entry = props.entry
  const isCurrent = props.currentUserID != null && entry.user_id === props.currentUserID

  return (
    <article className={`scoreboard-table-entry ${expanded ? 'expanded' : ''} ${isCurrent ? 'current-user' : ''}`}>
      <div className="scoreboard-table-row">
        <div className="scoreboard-rank-cell">#{entry.rank}</div>
        <div className="scoreboard-player-cell">
          <strong>{entry.display_name || entry.username}</strong>
          <small>@{entry.username}</small>
        </div>
        <div className="scoreboard-value-cell">
          <strong>{entry.score}</strong>
          <span>points</span>
        </div>
        <div className="scoreboard-value-cell">
          <strong>{entry.solves.length}</strong>
          <span>solves</span>
        </div>
        <div className="scoreboard-time-cell">
          <strong>{entry.last_solve_at ? formatRelative(parseRfc3339(entry.last_solve_at) ?? safeNow()) : '—'}</strong>
          <span>{entry.last_solve_at ? formatDateTime(parseRfc3339(entry.last_solve_at) ?? safeNow()) : ''}</span>
        </div>
        <div className="scoreboard-actions-cell">
          <button className="ghost-button" type="button" onClick={() => setExpanded((v) => !v)}>
            {expanded ? '收起' : '展开'}
          </button>
        </div>
      </div>

      {expanded ? (
        <div className="scoreboard-row-details">
          {entry.solves.length === 0 ? <div className="empty-state">暂无 solves</div> : null}
          {entry.solves.map((solve) => (
            <div key={`${solve.challenge_id}-${solve.solved_at}`} className="scoreboard-table-solve">
              <div>
                <strong>{solve.challenge_title}</strong>
                <div className="hint-text">{solve.challenge_slug}</div>
              </div>
              <div className="badge-row">
                <span className="badge">{solve.category}</span>
                <span className="badge">{solve.difficulty}</span>
                <span className="badge badge-solid">+{solve.awarded_points}</span>
                <span className="badge">{formatDateTimeCompact(parseRfc3339(solve.solved_at) ?? safeNow())}</span>
              </div>
            </div>
          ))}
        </div>
      ) : null}
    </article>
  )
}

function ScoreboardLeadersChart(props: { entries: ScoreboardEntry[]; topN: 3 | 5 | 10 }): React.JSX.Element {
  const [hoverT, setHoverT] = useState<number | null>(null)
  const [hoverLocked, setHoverLocked] = useState(false)

  const leaders = useMemo(() => {
    const ranked = [...props.entries].sort((a, b) => a.rank - b.rank)
    const withSolves = ranked.filter((entry) => entry.solves.length > 0 || entry.score > 0)
    const base = withSolves.length ? withSolves : ranked
    return base.slice(0, props.topN)
  }, [props.entries, props.topN])

  const series = useMemo(() => {
    return leaders.map((entry) => {
      let total = 0
      const points: Array<{ t: number; score: number }> = []
      for (const solve of entry.solves) {
        const t = parseRfc3339(solve.solved_at)?.getTime()
        if (!t) continue
        total += solve.awarded_points
        points.push({ t, score: total })
      }
      return { entry, points, finalScore: total }
    })
  }, [leaders])

  const solveTimes = useMemo(() => {
    const times = series.flatMap((item) => item.points.map((point) => point.t))
    const unique = Array.from(new Set(times))
    unique.sort((a, b) => a - b)
    return unique
  }, [series])

  const domain = useMemo(() => {
    const times = solveTimes
    if (times.length < 2) return null
    const min = Math.min(...times)
    const max = Math.max(...times)
    return { min, max }
  }, [solveTimes])

  const yMax = useMemo(() => {
    const maxScore = Math.max(...series.map((item) => Math.max(item.entry.score, item.finalScore, 0)), 0)
    return Math.max(1, maxScore)
  }, [series])

  if (!domain) {
    return <div className="hint-text">趋势数据不足（至少需要两个不同时间点的解题记录）。</div>
  }

  const chartWidth = 640
  const chartHeight = 180
  const padX = 16
  const padTop = 10
  const padBottom = 20
  const innerWidth = chartWidth - padX * 2
  const innerHeight = chartHeight - padTop - padBottom
  const span = Math.max(1, domain.max - domain.min)

  const xOf = (t: number): number => padX + ((t - domain.min) / span) * innerWidth
  const yOf = (score: number): number => padTop + innerHeight - (score / yMax) * innerHeight

  const findClosestTime = useCallback(
    (t: number): number | null => {
      if (!solveTimes.length) return null
      let closest = solveTimes[0]
      let best = Math.abs(closest - t)
      for (let i = 1; i < solveTimes.length; i += 1) {
        const candidate = solveTimes[i]
        const diff = Math.abs(candidate - t)
        if (diff < best) {
          best = diff
          closest = candidate
        }
      }
      return closest
    },
    [solveTimes],
  )

  const lineStyles: Array<{ stroke: string; dash?: string; opacity: number; width: number }> = [
    { stroke: 'var(--series-1)', opacity: 1, width: 2.6 },
    { stroke: 'var(--series-2)', opacity: 0.98, width: 2.4 },
    { stroke: 'var(--series-3)', opacity: 0.96, width: 2.3 },
    { stroke: 'var(--series-4)', opacity: 0.94, width: 2.2 },
    { stroke: 'var(--series-5)', opacity: 0.92, width: 2.1 },
    { stroke: 'var(--series-6)', opacity: 0.9, width: 2.0 },
    { stroke: 'var(--series-7)', opacity: 0.88, width: 2.0 },
    { stroke: 'var(--series-8)', opacity: 0.86, width: 1.9 },
    { stroke: 'var(--series-9)', opacity: 0.84, width: 1.9 },
    { stroke: 'var(--series-10)', opacity: 0.82, width: 1.9 },
  ]

  const buildStepPath = (points: Array<{ t: number; score: number }>): string => {
    const startX = xOf(domain.min)
    const startY = yOf(0)

    let d = `M ${startX.toFixed(1)} ${startY.toFixed(1)}`
    let prevScore = 0
    for (const point of points) {
      const x = xOf(point.t)
      const yBefore = yOf(prevScore)
      const yAfter = yOf(point.score)
      d += ` L ${x.toFixed(1)} ${yBefore.toFixed(1)}`
      d += ` L ${x.toFixed(1)} ${yAfter.toFixed(1)}`
      prevScore = point.score
    }

    const endX = xOf(domain.max)
    const endY = yOf(prevScore)
    d += ` L ${endX.toFixed(1)} ${endY.toFixed(1)}`
    return d
  }

  const rangeLeft = formatDateTimeCompact(new Date(domain.min))
  const rangeRight = formatDateTimeCompact(new Date(domain.max))

  const tooltip = useMemo(() => {
    if (hoverT == null) return null
    const at = new Date(hoverT)
    const timeLabel = formatDateTimeCompact(at)
    const rows = series
      .map((item, index) => {
        let scoreAt = 0
        let delta = 0
        for (const point of item.points) {
          if (point.t > hoverT) break
          scoreAt = point.score
          delta = point.score
        }
        // delta: score increase at this exact time (if solved at hoverT)
        const solvedHere = item.points.find((point) => point.t === hoverT)
        if (solvedHere) {
          const beforeIdx = item.points.findIndex((point) => point.t === hoverT) - 1
          const before = beforeIdx >= 0 ? item.points[beforeIdx].score : 0
          delta = solvedHere.score - before
        } else {
          delta = 0
        }
        const label = item.entry.display_name || item.entry.username
        const style = lineStyles[index % lineStyles.length]
        return {
          key: item.entry.user_id,
          rank: item.entry.rank,
          label,
          scoreAt,
          delta,
          stroke: style.stroke,
        }
      })
      .sort((a, b) => b.scoreAt - a.scoreAt)

    return { timeLabel, rows }
  }, [hoverT, lineStyles, series])

  const hoverX = hoverT != null ? xOf(hoverT) : null

  const onMouseMove = useCallback(
    (event: React.MouseEvent<SVGSVGElement>) => {
      if (hoverLocked) return
      const rect = event.currentTarget.getBoundingClientRect()
      const x = event.clientX - rect.left
      const t = domain.min + clamp((x - padX) / innerWidth, 0, 1) * span
      const closest = findClosestTime(t)
      if (closest == null) return
      setHoverT(closest)
    },
    [domain.min, findClosestTime, hoverLocked, innerWidth, padX, span],
  )

  const onMouseLeave = useCallback(() => {
    if (hoverLocked) return
    setHoverT(null)
  }, [hoverLocked])

  return (
    <div className="scoreboard-trend-block">
      <svg
        className="scoreboard-sparkline"
        viewBox={`0 0 ${chartWidth} ${chartHeight}`}
        role="img"
        aria-label="top players score over time"
        onMouseMove={onMouseMove}
        onMouseLeave={onMouseLeave}
        onClick={() => setHoverLocked((v) => !v)}
      >
        <g className="scoreboard-sparkline-grid">
          {[0, 0.5, 1].map((ratio) => {
            const y = padTop + innerHeight * ratio
            return (
              <line
                key={ratio}
                x1={padX}
                y1={y}
                x2={chartWidth - padX}
                y2={y}
              />
            )
          })}
        </g>

        {hoverX != null ? (
          <g className="scoreboard-sparkline-hover">
            <line x1={hoverX} y1={padTop} x2={hoverX} y2={padTop + innerHeight} />
          </g>
        ) : null}

        {series.map((item, index) => {
          const style = lineStyles[index % lineStyles.length]
          const path = buildStepPath(item.points)
          const solvedHere = hoverT != null && item.points.some((point) => point.t === hoverT)
          const highlight = hoverT != null
          const opacity = highlight ? (solvedHere ? 1 : 0.32) : style.opacity
          const strokeWidth = highlight ? (solvedHere ? style.width + 0.6 : Math.max(1.6, style.width - 0.5)) : style.width

          const activePoint =
            hoverT != null
              ? item.points.find((point) => point.t === hoverT) ?? item.points.filter((point) => point.t < hoverT).slice(-1)[0] ?? null
              : null

          const cx = activePoint ? xOf(activePoint.t) : null
          const cy = activePoint ? yOf(activePoint.score) : null

          return (
            <g key={item.entry.user_id} className="scoreboard-sparkline-series">
              <path
                d={path}
                className="scoreboard-sparkline-line"
                stroke={style.stroke}
                strokeWidth={strokeWidth}
                strokeDasharray={style.dash}
                strokeOpacity={opacity}
              />
              {cx != null && cy != null ? (
                <circle cx={cx} cy={cy} r={solvedHere ? 4.6 : 3.2} fill={style.stroke} opacity={opacity} />
              ) : null}
            </g>
          )
        })}
      </svg>

      {tooltip ? (
        <div className={`scoreboard-tooltip ${hoverLocked ? 'locked' : ''}`}>
          <div className="scoreboard-tooltip-head">
            <strong>{tooltip.timeLabel}</strong>
            <span className="hint-text">{hoverLocked ? '锁定' : '悬浮'} · 点击可锁定/取消</span>
          </div>
          <div className="scoreboard-tooltip-body">
            {tooltip.rows.map((row) => (
              <div key={row.key} className="scoreboard-tooltip-row">
                <span className="scoreboard-tooltip-dot" style={{ background: row.stroke }} aria-hidden="true" />
                <span className="scoreboard-tooltip-name">#{row.rank} {row.label}</span>
                <span className="scoreboard-tooltip-score">{row.scoreAt}</span>
                <span className={`scoreboard-tooltip-delta ${row.delta ? 'active' : ''}`}>{row.delta ? `+${row.delta}` : ''}</span>
              </div>
            ))}
          </div>
        </div>
      ) : null}

      <div className="scoreboard-range hint-text">
        <span>{rangeLeft}</span>
        <span>{rangeRight}</span>
      </div>

      <div className="scoreboard-legend">
        {series.map((item, index) => {
          const style = lineStyles[index % lineStyles.length]
          const label = item.entry.display_name || item.entry.username
          return (
            <div key={item.entry.user_id} className="scoreboard-legend-item" title={`#${item.entry.rank} ${label}`}>
              <svg className="scoreboard-legend-sample" viewBox="0 0 24 10" aria-hidden="true">
                <line
                  x1="1"
                  y1="5"
                  x2="23"
                  y2="5"
                  stroke={style.stroke}
                  strokeWidth="2.6"
                  strokeDasharray={style.dash}
                  strokeOpacity={style.opacity}
                  strokeLinecap="round"
                />
              </svg>
              <span className="scoreboard-legend-name">#{item.entry.rank} {label}</span>
              <span className="scoreboard-legend-meta">{item.entry.score} pts</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function ScoreboardTrend(props: {
  items: ScoreboardEntry[]
  scope: 'top10' | 'top50' | 'all'
  onScopeChange: (value: 'top10' | 'top50' | 'all') => void
}): React.JSX.Element {
  const ranked = useMemo(() => {
    const sorted = [...props.items].sort((a, b) => a.rank - b.rank)
    if (props.scope === 'top10') return sorted.slice(0, 10)
    if (props.scope === 'top50') return sorted.slice(0, 50)
    return sorted
  }, [props.items, props.scope])

  const solves = useMemo(() => {
    return ranked.flatMap((entry) => entry.solves)
  }, [ranked])

  const trend = useMemo(() => makeSolveTrend(solves, 28), [solves])

  if (!trend) {
    return <div className="hint-text">暂无趋势数据</div>
  }

  const total = solves.length

  return (
    <div className="detail-row" style={{ display: 'grid', gap: 10 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, alignItems: 'center', flexWrap: 'wrap' }}>
        <strong>解题趋势</strong>
        <div className="badge-row">
          <button
            className={`nav-pill ${props.scope === 'top10' ? 'active' : ''}`}
            type="button"
            onClick={() => props.onScopeChange('top10')}
          >
            Top 10
          </button>
          <button
            className={`nav-pill ${props.scope === 'top50' ? 'active' : ''}`}
            type="button"
            onClick={() => props.onScopeChange('top50')}
          >
            Top 50
          </button>
          <button className={`nav-pill ${props.scope === 'all' ? 'active' : ''}`} type="button" onClick={() => props.onScopeChange('all')}>
            All
          </button>
        </div>
      </div>

      <div className="trend" role="img" aria-label="scoreboard solve trend">
        {trend.normalized.map((value, idx) => (
          <span
            // eslint-disable-next-line react/no-array-index-key
            key={idx}
            className="trend-bar"
            style={{ height: `${Math.round(6 + value * 22)}px` }}
            title={`${trend.counts[idx]} solves`}
          />
        ))}
      </div>

      <div className="hint-text">范围：{props.scope} · solves: {total}</div>
    </div>
  )
}
