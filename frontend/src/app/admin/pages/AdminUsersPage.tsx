import React, { useEffect, useMemo, useState } from 'react'

import { api, type AdminUser } from '../../../api'
import { NoticeBanner } from '../../components/NoticeBanner'
import type { Notice } from '../../utils/errors'
import { errorToNotice } from '../../utils/errors'

const ROLES = ['admin', 'ops', 'author', 'player'] as const
const STATUSES = ['active', 'disabled'] as const

export function AdminUsersPage(props: { token: string }): React.JSX.Element {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)

  const [items, setItems] = useState<AdminUser[]>([])
  const [filter, setFilter] = useState('')

  const load = async (): Promise<void> => {
    setLoading(true)
    setNotice(null)
    try {
      const response = await api.adminUsers(props.token)
      setItems(response.items)
    } catch (error) {
      setNotice(errorToNotice(error, '用户列表加载失败。'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase()
    if (!q) return items
    return items.filter((u) => {
      return (
        u.username.toLowerCase().includes(q) ||
        u.email.toLowerCase().includes(q) ||
        (u.display_name ?? '').toLowerCase().includes(q) ||
        u.role.toLowerCase().includes(q) ||
        u.status.toLowerCase().includes(q)
      )
    })
  }, [filter, items])

  const update = async (user: AdminUser, patch: Partial<Pick<AdminUser, 'role' | 'display_name' | 'status'>>): Promise<void> => {
    setSaving(true)
    setNotice(null)
    try {
      const payload = {
        role: patch.role ?? user.role,
        display_name: patch.display_name ?? user.display_name,
        status: patch.status ?? user.status,
      }
      const response = await api.updateAdminUser(props.token, user.id, payload)
      setItems((list) => list.map((item) => (item.id === user.id ? response.user : item)))
      setNotice({ tone: 'ok', text: `已更新用户：${user.username}` })
    } catch (error) {
      setNotice(errorToNotice(error, '更新用户失败。'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="admin-traffic-view">
      <section className="panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">Users</p>
            <h2>用户</h2>
            <p className="panel-subtitle">管理角色与状态；上线前务必确认管理员账号与禁用策略。</p>
          </div>
          <div className="inline-actions">
            <button className="ghost-button" type="button" onClick={() => void load()} disabled={loading || saving}>
              刷新
            </button>
          </div>
        </header>

        <NoticeBanner notice={notice} />

        <div className="board-list-toolbar" style={{ marginTop: 10 }}>
          <label className="field" style={{ width: 'min(520px, 100%)' }}>
            <span>搜索</span>
            <input value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="username / email / role / status" />
          </label>
          <div className="board-filter-meta">{loading ? '加载中…' : `${filtered.length}/${items.length}`}</div>
        </div>
      </section>

      <section className="panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">List</p>
            <h2>用户列表</h2>
          </div>
        </header>

        {loading ? <div className="empty-state">加载中…</div> : null}
        {!loading && filtered.length === 0 ? <div className="empty-state">暂无匹配用户</div> : null}
        {!loading && filtered.length ? (
          <div className="table-stack">
            {filtered.map((user) => (
              <div key={user.id} className="row-card" style={{ gridTemplateColumns: 'minmax(0, 1fr)' }}>
                <div className="badge-row">
                  <span className="badge">#{user.id}</span>
                  <span className="badge">{user.role}</span>
                  <span className="badge">{user.status}</span>
                  {user.last_login_at ? <span className="badge">last {user.last_login_at}</span> : null}
                </div>
                <strong>{user.display_name || user.username}</strong>
                <div className="hint-text">@{user.username} · {user.email}</div>

                <div className="admin-focus-grid" style={{ marginTop: 12 }}>
                  <label className="field">
                    <span>display_name</span>
                    <input
                      value={user.display_name}
                      onChange={(e) => {
                        const next = e.target.value
                        setItems((list) => list.map((item) => (item.id === user.id ? { ...item, display_name: next } : item)))
                      }}
                    />
                  </label>

                  <div className="card-list" style={{ gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 10 }}>
                    <label className="field">
                      <span>role</span>
                      <select
                        value={user.role}
                        onChange={(e) => {
                          const next = e.target.value
                          setItems((list) => list.map((item) => (item.id === user.id ? { ...item, role: next } : item)))
                        }}
                      >
                        {ROLES.map((role) => (
                          <option key={role} value={role}>
                            {role}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="field">
                      <span>status</span>
                      <select
                        value={user.status}
                        onChange={(e) => {
                          const next = e.target.value
                          setItems((list) => list.map((item) => (item.id === user.id ? { ...item, status: next } : item)))
                        }}
                      >
                        {STATUSES.map((s) => (
                          <option key={s} value={s}>
                            {s}
                          </option>
                        ))}
                      </select>
                    </label>
                  </div>
                </div>

                <div className="wrap-actions" style={{ marginTop: 12 }}>
                  <button className="primary-button" type="button" disabled={saving} onClick={() => void update(user, {})}>
                    保存
                  </button>
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </section>
    </section>
  )
}

