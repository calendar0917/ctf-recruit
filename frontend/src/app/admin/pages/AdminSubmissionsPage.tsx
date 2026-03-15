import React, { useEffect, useMemo, useState } from 'react'

import { api, type AdminSubmission } from '../../../api'
import { NoticeBanner } from '../../components/NoticeBanner'
import type { Notice } from '../../utils/errors'
import { errorToNotice } from '../../utils/errors'

export function AdminSubmissionsPage(props: { token: string }): React.JSX.Element {
  const [loading, setLoading] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)

  const [items, setItems] = useState<AdminSubmission[]>([])
  const [filter, setFilter] = useState('')

  const load = async (): Promise<void> => {
    setLoading(true)
    setNotice(null)
    try {
      const response = await api.adminSubmissions(props.token)
      setItems(response.items)
    } catch (error) {
      setNotice(errorToNotice(error, '提交记录加载失败。'))
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
    return items.filter((it) => it.username.toLowerCase().includes(q) || it.challenge_slug.toLowerCase().includes(q) || it.source_ip.toLowerCase().includes(q))
  }, [filter, items])

  return (
    <section className="admin-traffic-view">
      <section className="panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">Submissions</p>
            <h2>提交记录</h2>
            <p className="panel-subtitle">用于赛中排查：频繁错误、异常 IP、flag 爆破迹象等。</p>
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
            <input value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="username / challenge / ip" />
          </label>
          <div className="board-filter-meta">{loading ? '加载中…' : `${filtered.length}/${items.length}`}</div>
        </div>
      </section>

      <section className="panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">List</p>
            <h2>最近提交</h2>
          </div>
        </header>

        {loading ? <div className="empty-state">加载中…</div> : null}
        {!loading && filtered.length === 0 ? <div className="empty-state">暂无数据</div> : null}
        {!loading && filtered.length ? (
          <div className="table-stack">
            {filtered.slice(0, 300).map((item) => (
              <article key={item.id} className="entry-card">
                <div className="badge-row">
                  <span className="badge">#{item.id}</span>
                  <span className={`badge ${item.correct ? 'badge-solid' : ''}`}>{item.correct ? 'Correct' : 'Wrong'}</span>
                  <span className="badge">{item.challenge_slug}</span>
                  <span className="badge">@{item.username}</span>
                  <span className="badge">{item.source_ip}</span>
                </div>
                <strong>{item.submitted_at}</strong>
                <div className="hint-text">challenge_id {item.challenge_id}</div>
              </article>
            ))}
          </div>
        ) : null}
      </section>
    </section>
  )
}

