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
  { id: 'briefing', label: 'Briefing' },
  { id: 'board', label: 'Challenges', gatedByPhase: 'challenge_list_visible' },
  { id: 'scoreboard', label: 'Scoreboard', gatedByPhase: 'scoreboard_visible' },
  { id: 'me', label: 'My', gatedByPhase: 'challenge_list_visible' },
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
  return <span className={`badge ${cls}`}>{difficulty}</span>
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
    if (!query) return challenges
    return challenges.filter((item) => {
      return (
        item.title.toLowerCase().includes(query) ||
        item.slug.toLowerCase().includes(query) ||
        item.category.toLowerCase().includes(query) ||
        item.difficulty.toLowerCase().includes(query)
      )
    })
  }, [challengeFilter, challenges])

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
    const now = new Date()
    return (
      <div className="badge-row">
        {start ? <span className="badge">Start {formatDateTime(start)} ({formatRelative(start, now)})</span> : null}
        {end ? <span className="badge">End {formatDateTime(end)} ({formatRelative(end, now)})</span> : null}
      </div>
    )
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
              <SectionHeader eyebrow="Briefing" title="先做题，再做花哨" subtitle={contestPhase?.message ?? contestInfo?.description ?? '比赛信息加载中…'}>
                {contestStatusBadge}
              </SectionHeader>
              {contestWindow}
              <div className="card-list player-mini-grid">
                <div className="hero-stat-card">
                  <span className="capability-value">题目列表</span>
                  <strong>{contestPhase?.challenge_list_visible ? '开放' : '未开放'}</strong>
                  <small>根据阶段动态控制</small>
                </div>
                <div className="hero-stat-card">
                  <span className="capability-value">提交 Flag</span>
                  <strong>{contestPhase?.submission_allowed ? '允许' : '关闭'}</strong>
                  <small>提交入口在题目详情</small>
                </div>
                <div className="hero-stat-card">
                  <span className="capability-value">动态实例</span>
                  <strong>{contestPhase?.runtime_allowed ? '开放' : '未开放'}</strong>
                  <small>动态题才显示控制面板</small>
                </div>
                <div className="hero-stat-card">
                  <span className="capability-value">排行榜</span>
                  <strong>{contestPhase?.scoreboard_visible ? '公开' : '未公开'}</strong>
                  <small>按阶段控制可见性</small>
                </div>
              </div>
            </section>

            <section className="panel announcement-subpanel page-enter page-enter-1">
              <SectionHeader eyebrow="Announcements" title="公告" subtitle={contestPhase?.announcement_visible ? '重要变更会置顶展示。' : '当前阶段未开放公告。'} />
              {contestPhase?.announcement_visible ? <AnnouncementList items={announcements} loading={announcementsLoading} /> : <div className="empty-state">公告未开放</div>}
            </section>

            <section className="panel page-enter page-enter-2">
              <SectionHeader
                eyebrow="Access"
                title={authUser ? '已登录' : '登录 / 注册'}
                subtitle={authUser ? '可以直接去 Challenges 做题。' : '为了聚焦做题：登录模块保持最少干扰。'}
              />
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
            <section className="panel panel-hero board-hero-compact">
              <SectionHeader
                eyebrow="Challenges"
                title="题目面板"
                subtitle={contestPhase?.challenge_detail_visible ? '左侧筛选 + 右侧做题；所有操作围绕做题。' : '当前阶段仅开放题目列表，详情未开放。'}
              >
                <span className="badge">{challengesLoading ? 'loading…' : `${challenges.length} total`}</span>
              </SectionHeader>
              <div className="board-summary-grid card-list capability-grid">
                <div className="summary-card">
                  <span className="capability-value">Focus</span>
                  <strong>题目列表 → 详情 → 提交</strong>
                  <small>最短路径设计</small>
                </div>
                <div className="summary-card">
                  <span className="capability-value">Phase</span>
                  <strong>{contestPhase?.status ?? 'unknown'}</strong>
                  <small>{contestPhase?.message ?? '—'}</small>
                </div>
              </div>
            </section>

            <div className="player-board-shell workspace-grid">
              <aside className="panel rail-panel player-side-panel">
                <SectionHeader eyebrow="Browse" title="题目列表" subtitle="输入关键词快速定位。" />
                <div className="board-list-toolbar">
                  <label className="field">
                    <span>搜索</span>
                    <input value={challengeFilter} onChange={(event) => setChallengeFilter(event.target.value)} placeholder="web / crypto / easy / welcome" />
                  </label>
                  <div className="board-filter-meta">
                    <div>阶段 gating：{contestPhase?.challenge_list_visible ? 'list on' : 'list off'} / {contestPhase?.challenge_detail_visible ? 'detail on' : 'detail off'}</div>
                    <div>{filteredChallenges.length} shown</div>
                  </div>
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
                      <div className="challenge-card-head badge-row">
                        <span className="badge">{item.category}</span>
                        <DifficultyBadge difficulty={item.difficulty} />
                        {item.dynamic ? <span className="badge badge-solid">Dynamic</span> : <span className="badge">Static</span>}
                      </div>
                      <strong>{item.title}</strong>
                      <div className="challenge-card-subline">
                        <small>{item.slug}</small>
                        <small>{item.points} pts</small>
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

                      {activeChallenge ? (
                        <div className="detail-stack" style={{ marginTop: 16 }}>
                          <strong>附件</strong>
                          {!contestPhase?.attachment_visible ? (
                            <div className="empty-state">当前阶段未开放附件。</div>
                          ) : activeChallenge.attachments.length ? (
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
                          ) : (
                            <div className="empty-state">无附件</div>
                          )}
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
                      <div className="primary-action-intro player-action-intro">
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
                        <div className="primary-action-status">
                          <span>Submission</span>
                          <strong>{contestPhase?.submission_allowed ? 'Open' : 'Closed'}</strong>
                          <span style={{ fontSize: 12, opacity: 0.8 }}>{submissionLoading ? 'working…' : 'ready'}</span>
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

                        <div className="runtime-help-grid">
                          <div className="detail-row">
                            <strong>常见失败原因</strong>
                            <div className="hint-text" style={{ marginTop: 8 }}>
                              <div>· instance_capacity_reached：题目并发上限</div>
                              <div>· instance_cooldown_active：用户冷却中</div>
                              <div>· instance_port_exhausted：端口池耗尽</div>
                            </div>
                          </div>
                        </div>
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
            <section className="panel panel-hero scoreboard-stage-hero">
              <SectionHeader eyebrow="Scoreboard" title="排行榜" subtitle={contestPhase?.scoreboard_visible ? '默认展示 Top 50；点击展开查看解题细节。' : '当前阶段未公开排行榜。'}>
                <button className="ghost-button" type="button" onClick={() => void loadScoreboard()} disabled={!contestPhase?.scoreboard_visible || scoreboardLoading}>
                  刷新
                </button>
              </SectionHeader>

              {!contestPhase?.scoreboard_visible ? <div className="empty-state">排行榜未开放</div> : null}
              {contestPhase?.scoreboard_visible ? (
                <div className="scoreboard-personal-strip card-list">
                  <div className="summary-card">
                    <span className="eyebrow">My Score</span>
                    <strong>{authUser ? (scoreboard.find((item) => item.user_id === authUser.id)?.score ?? '—') : '—'}</strong>
                    <small>登录后展示更准确</small>
                  </div>
                  <div className="summary-card">
                    <span className="eyebrow">Entries</span>
                    <strong>{scoreboardLoading ? '…' : scoreboard.length}</strong>
                    <small>公开榜单</small>
                  </div>
                  <div className="summary-card">
                    <span className="eyebrow">Updated</span>
                    <strong>{scoreboardLoading ? 'loading…' : 'just now'}</strong>
                    <small>手动刷新</small>
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
            <section className="panel panel-hero player-hero">
              <SectionHeader eyebrow="My" title="我的进度" subtitle={authUser ? '复盘自己的提交和解题路径。' : '请先登录。'}>
                <button className="ghost-button" type="button" onClick={() => void loadMyProgress()} disabled={!token || myLoading}>
                  刷新
                </button>
              </SectionHeader>
              <NoticeBanner notice={authNotice} />
              {!authUser ? <div className="empty-state">未登录</div> : null}
              {authUser ? (
                <div className="player-summary-stack card-list player-mini-grid">
                  <div className="hero-stat-card">
                    <span className="eyebrow">Solves</span>
                    <strong>{myLoading ? '…' : mySolves.length}</strong>
                    <small>已解</small>
                  </div>
                  <div className="hero-stat-card">
                    <span className="eyebrow">Submissions</span>
                    <strong>{myLoading ? '…' : mySubmissions.length}</strong>
                    <small>提交记录</small>
                  </div>
                  <div className="hero-stat-card hero-stat-card-strong">
                    <span className="eyebrow">Focus</span>
                    <strong>下一题</strong>
                    <small>回到 Challenges 继续</small>
                  </div>
                  <div className="hero-stat-card">
                    <span className="eyebrow">Role</span>
                    <strong>{authUser.role}</strong>
                    <small>{authUser.status}</small>
                  </div>
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

        <footer style={{ marginTop: 24, color: 'var(--muted)', fontSize: 12, textAlign: 'center' }}>
          Demo UI · focus on solving challenges · phase-aware gating
        </footer>
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
              </div>
            </div>
          ))}
        </div>
      ) : null}
    </article>
  )
}
