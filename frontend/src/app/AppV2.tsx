import React, { useCallback, useEffect, useMemo, useState } from 'react'

import { api, type AuthUser, type ContestInfo, type ContestPhase } from '../api'
import { NoticeBanner } from './components/NoticeBanner'
import { describeError, errorToNotice, isUnauthorized, type Notice } from './utils/errors'

import './design-tokens.css'
import './app.css'

type View = 'briefing' | 'board' | 'scoreboard' | 'admin'

const TOKEN_STORAGE_KEY = 'ctf.frontend.token'

const views: Array<{ id: View; label: string }> = [
  { id: 'briefing', label: 'Briefing' },
  { id: 'board', label: 'Challenges' },
  { id: 'scoreboard', label: 'Scoreboard' },
  { id: 'admin', label: 'Admin' },
]

export function AppV2(): React.JSX.Element {
  const [view, setView] = useState<View>('briefing')
  const [token, setToken] = useState<string>(() => window.localStorage.getItem(TOKEN_STORAGE_KEY) ?? '')
  const [authUser, setAuthUser] = useState<AuthUser | null>(null)
  const [sessionLoading, setSessionLoading] = useState(Boolean(token))

  const [contestInfo, setContestInfo] = useState<ContestInfo | null>(null)
  const [contestPhase, setContestPhase] = useState<ContestPhase | null>(null)

  const [publicNotice, setPublicNotice] = useState<Notice | null>(null)
  const [authNotice, setAuthNotice] = useState<Notice | null>(null)

  const canAccessAdmin = authUser?.role === 'admin' || authUser?.role === 'ops' || authUser?.role === 'author'

  const visibleViews = useMemo(() => {
    return views.filter((item) => {
      if (item.id === 'admin') {
        return canAccessAdmin
      }
      return true
    })
  }, [canAccessAdmin])

  const clearSession = useCallback((message?: string) => {
    setToken('')
    setAuthUser(null)
    if (message) {
      setAuthNotice({ tone: 'danger', text: message })
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
    if (!token) {
      window.localStorage.removeItem(TOKEN_STORAGE_KEY)
    }
  }, [token])

  useEffect(() => {
    let active = true
    setPublicNotice(null)
    void api
      .contest()
      .then((response) => {
        if (!active) return
        setContestInfo(response.contest)
        setContestPhase(response.phase)
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
    if (view === 'admin' && !canAccessAdmin) {
      setView('briefing')
    }
  }, [canAccessAdmin, view])

  return (
    <div className="ds-shell">
      <header className="ds-topbar">
        <div className="ds-brand">
          <p className="eyebrow">YulinSec CTF</p>
          <h1>Recruit Console v2</h1>
        </div>

        <nav className="ds-nav" aria-label="Primary">
          {visibleViews.map((item) => (
            <button
              className="ds-pill"
              key={item.id}
              type="button"
              aria-current={view === item.id ? 'page' : undefined}
              onClick={() => setView(item.id)}
            >
              {item.label}
            </button>
          ))}
        </nav>

        <div className="ds-session">
          <div className="ds-user">
            <strong>{authUser ? authUser.display_name || authUser.username : 'Guest'}</strong>
            <span>
              {sessionLoading ? 'loading session…' : authUser ? authUser.role : 'not authenticated'}
              {contestInfo?.status ? ` | contest: ${contestInfo.status}` : ''}
            </span>
          </div>
          {authUser ? (
            <button className="ds-btn danger" type="button" onClick={() => clearSession('已退出当前账号。')}>
              Logout
            </button>
          ) : (
            <button className="ds-btn primary" type="button" onClick={() => setView('briefing')}>
              Login
            </button>
          )}
        </div>
      </header>

      <main className="ds-main" id="main-content">
        <NoticeBanner notice={publicNotice} />
        {contestPhase ? (
          <div className="ds-card">
            <div className="ds-card-header">
              <h2>Phase</h2>
              <p>{contestPhase.message}</p>
            </div>
            <div className="ds-card-body">
              <div style={{ display: 'grid', gap: 10, fontFamily: 'var(--mono)', fontSize: 12 }}>
                <div>announcement_visible: {String(contestPhase.announcement_visible)}</div>
                <div>challenge_list_visible: {String(contestPhase.challenge_list_visible)}</div>
                <div>submission_allowed: {String(contestPhase.submission_allowed)}</div>
                <div>runtime_allowed: {String(contestPhase.runtime_allowed)}</div>
                <div>scoreboard_visible: {String(contestPhase.scoreboard_visible)}</div>
                <div>registration_allowed: {String(contestPhase.registration_allowed)}</div>
              </div>
            </div>
          </div>
        ) : null}

        <div className="ds-card">
          <div className="ds-card-header">
            <h2>V2 Progress</h2>
            <p>
              This is the new shell + tokens. Next step: implement Briefing/Board/Scoreboard pages with the new design.
            </p>
          </div>
          <div className="ds-card-body">
            <NoticeBanner notice={authNotice} />
            <div style={{ display: 'grid', gap: 10, fontSize: 12, color: 'var(--text-dim)' }}>
              <div>Current view: {view}</div>
              <div>Admin access: {String(canAccessAdmin)}</div>
              <div>Server-driven gating will be enforced via error codes and phase flags.</div>
            </div>
            <div style={{ marginTop: 12, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
              <button className="ds-btn" type="button" onClick={() => setAuthNotice(guardedNotice({ code: 'runtime_closed' }, ''))}>
                Demo runtime_closed
              </button>
              <button className="ds-btn" type="button" onClick={() => setAuthNotice(guardedNotice({ code: 'submission_closed' }, ''))}>
                Demo submission_closed
              </button>
              <button className="ds-btn" type="button" onClick={() => setAuthNotice(guardedNotice({ code: 'contest_not_public' }, ''))}>
                Demo contest_not_public
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
