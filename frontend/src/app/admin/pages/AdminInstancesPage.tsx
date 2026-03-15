import React, { useEffect, useMemo, useState } from 'react'

import { api, type AdminInstance } from '../../../api'
import { NoticeBanner } from '../../components/NoticeBanner'
import type { Notice } from '../../utils/errors'
import { errorToNotice } from '../../utils/errors'

export function AdminInstancesPage(props: { token: string }): React.JSX.Element {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)

  const [items, setItems] = useState<AdminInstance[]>([])
  const [filter, setFilter] = useState('')

  const load = async (): Promise<void> => {
    setLoading(true)
    setNotice(null)
    try {
      const response = await api.adminInstances(props.token)
      setItems(response.items)
    } catch (error) {
      setNotice(errorToNotice(error, '实例列表加载失败。'))
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
    return items.filter((it) => it.username.toLowerCase().includes(q) || it.challenge_slug.toLowerCase().includes(q) || it.status.toLowerCase().includes(q))
  }, [filter, items])

  const terminate = async (instanceID: number): Promise<void> => {
    if (!confirm(`确认终止实例 #${instanceID}？`)) return
    setSaving(true)
    setNotice(null)
    try {
      await api.terminateAdminInstance(props.token, instanceID)
      setNotice({ tone: 'ok', text: `已终止实例 #${instanceID}` })
      await load()
    } catch (error) {
      setNotice(errorToNotice(error, '终止实例失败。'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="admin-traffic-view">
      <section className="panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">Instances</p>
            <h2>动态实例</h2>
            <p className="panel-subtitle">用于赛中故障处理：查看运行状态、过期时间、强制终止。</p>
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
            <input value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="username / challenge / status" />
          </label>
          <div className="board-filter-meta">{loading ? '加载中…' : `${filtered.length}/${items.length}`}</div>
        </div>
      </section>

      <section className="panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">List</p>
            <h2>实例列表</h2>
          </div>
        </header>

        {loading ? <div className="empty-state">加载中…</div> : null}
        {!loading && filtered.length === 0 ? <div className="empty-state">暂无实例</div> : null}
        {!loading && filtered.length ? (
          <div className="table-stack">
            {filtered.map((item) => (
              <article key={item.id} className="entry-card">
                <div className="badge-row">
                  <span className="badge">#{item.id}</span>
                  <span className="badge">{item.status}</span>
                  <span className="badge">{item.challenge_slug}</span>
                  <span className="badge">@{item.username}</span>
                  <span className="badge">:{item.host_port}</span>
                </div>
                <strong>{item.container_id}</strong>
                <div className="hint-text">expires_at {item.expires_at}{item.terminated_at ? ` · terminated_at ${item.terminated_at}` : ''}</div>
                <div className="wrap-actions" style={{ marginTop: 12 }}>
                  <button className="ghost-button danger-button" type="button" disabled={saving} onClick={() => void terminate(item.id)}>
                    终止
                  </button>
                </div>
              </article>
            ))}
          </div>
        ) : null}
      </section>
    </section>
  )
}

