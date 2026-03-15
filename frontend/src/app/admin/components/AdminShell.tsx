import React from 'react'

import type { AuthUser } from '../../../api'
import { NoticeBanner } from '../../components/NoticeBanner'
import type { Notice } from '../../utils/errors'

import { hasAdminPermission, type AdminPermission } from '../utils/permissions'

export type AdminView =
  | 'contest'
  | 'challenges'
  | 'announcements'
  | 'submissions'
  | 'instances'
  | 'users'
  | 'audit'

const tabs: Array<{ id: AdminView; label: string; requires?: AdminPermission }> = [
  { id: 'contest', label: '比赛', requires: 'contest:read' },
  { id: 'challenges', label: '题目', requires: 'challenge:read' },
  { id: 'announcements', label: '公告', requires: 'announcement:read' },
  { id: 'submissions', label: '提交', requires: 'submission:read' },
  { id: 'instances', label: '实例', requires: 'instance:read' },
  { id: 'users', label: '用户', requires: 'user:read' },
  { id: 'audit', label: '审计', requires: 'audit:read' },
]

export function AdminShell(props: {
  user: AuthUser
  view: AdminView
  onViewChange: (value: AdminView) => void
  notice: Notice | null
  children: React.ReactNode
  actions?: React.ReactNode
}): React.JSX.Element {
  const visibleTabs = tabs.filter((tab) => (tab.requires ? hasAdminPermission(props.user, tab.requires) : true))
  const activeVisible = visibleTabs.some((tab) => tab.id === props.view)
  const currentView = activeVisible ? props.view : visibleTabs[0]?.id ?? 'challenges'

  return (
    <section className="view-stack page-enter admin-content-stack">
      <section className="panel panel-hero">
        <header className="panel-head" style={{ marginBottom: 10 }}>
          <div>
            <p className="eyebrow">Admin</p>
            <h2>管理面板</h2>
            <p className="panel-subtitle">最短路径：改比赛状态 / 题目 / 公告 / 用户。</p>
          </div>
          {props.actions ? <div className="inline-actions">{props.actions}</div> : null}
        </header>

        <nav className="tab-strip" aria-label="Admin navigation">
          {visibleTabs.map((tab) => (
            <button
              key={tab.id}
              className={`tab-pill ${currentView === tab.id ? 'active' : ''}`}
              type="button"
              aria-current={currentView === tab.id ? 'page' : undefined}
              onClick={() => props.onViewChange(tab.id)}
            >
              {tab.label}
            </button>
          ))}
        </nav>
      </section>

      <NoticeBanner notice={props.notice} />

      {props.children}
    </section>
  )
}

