import React, { useEffect, useMemo, useState } from 'react'

import { api, type AdminAuditLog } from '../../../api'
import { NoticeBanner } from '../../components/NoticeBanner'
import type { Notice } from '../../utils/errors'
import { errorToNotice } from '../../utils/errors'

export function AdminAuditLogsPage(props: { token: string }): React.JSX.Element {
  const [loading, setLoading] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)

  const [items, setItems] = useState<AdminAuditLog[]>([])
  const [filter, setFilter] = useState('')

  const load = async (): Promise<void> => {
    setLoading(true)
    setNotice(null)
    try {
      const response = await api.adminAuditLogs(props.token)
      setItems(response.items)
    } catch (error) {
      setNotice(errorToNotice(error, '审计日志加载失败。'))
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
    return items.filter((it) => {
      const raw = JSON.stringify(it.details ?? {})
      return it.action.toLowerCase().includes(q) || it.resource_type.toLowerCase().includes(q) || it.resource_id.toLowerCase().includes(q) || raw.toLowerCase().includes(q)
    })
  }, [filter, items])

  return (
    <section className="admin-audit-view">
      <section className="panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">Audit</p>
            <h2>审计日志</h2>
            <p className="panel-subtitle">记录后台关键写操作，用于追溯。</p>
          </div>
          <div className="inline-actions">
            <button className="ghost-button" type="button" onClick={() => void load()} disabled={loading}>
              刷新
            </button>
          </div>
        </header>

        <NoticeBanner notice={notice} />

        <div className="board-list-toolbar" style={{ marginTop: 10 }}>
          <label className="field" style={{ width: 'min(520px, 100%)' }}>
            <span>搜索</span>
            <input value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="action / resource / keyword" />
          </label>
          <div className="board-filter-meta">{loading ? '加载中…' : `${filtered.length}/${items.length}`}</div>
        </div>
      </section>

      <section className="panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">List</p>
            <h2>最近审计</h2>
          </div>
        </header>

        {loading ? <div className="empty-state">加载中…</div> : null}
        {!loading && filtered.length === 0 ? <div className="empty-state">暂无数据</div> : null}
        {!loading && filtered.length ? (
          <div className="compact-list audit-list">
            {filtered.slice(0, 300).map((item) => (
              <article key={item.id} className="entry-card">
                <div className="badge-row">
                  <span className="badge">#{item.id}</span>
                  <span className="badge">{item.action}</span>
                  <span className="badge">{item.resource_type}</span>
                  <span className="badge">{item.resource_id}</span>
                  <span className="badge">actor {item.actor_user_id ?? '—'}</span>
                </div>
                <strong>{item.created_at}</strong>
                {item.details ? <pre className="code-block" style={{ marginTop: 12 }}>{JSON.stringify(item.details, null, 2)}</pre> : null}
              </article>
            ))}
          </div>
        ) : null}
      </section>
    </section>
  )
}

