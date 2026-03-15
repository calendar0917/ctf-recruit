import React, { useEffect, useMemo, useState } from 'react'

import { api, type AdminAnnouncement } from '../../../api'
import { NoticeBanner } from '../../components/NoticeBanner'
import type { Notice } from '../../utils/errors'
import { errorToNotice } from '../../utils/errors'

export function AdminAnnouncementsPage(props: { token: string }): React.JSX.Element {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)

  const [items, setItems] = useState<AdminAnnouncement[]>([])

  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [pinned, setPinned] = useState(false)
  const [published, setPublished] = useState(true)

  const load = async (): Promise<void> => {
    setLoading(true)
    setNotice(null)
    try {
      const response = await api.adminAnnouncements(props.token)
      setItems(response.items)
    } catch (error) {
      setNotice(errorToNotice(error, '公告列表加载失败。'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const canCreate = useMemo(() => {
    if (saving) return false
    if (!title.trim()) return false
    if (!content.trim()) return false
    return true
  }, [content, saving, title])

  const create = async (): Promise<void> => {
    if (!canCreate) return
    setSaving(true)
    setNotice(null)
    try {
      await api.createAdminAnnouncement(props.token, {
        title: title.trim(),
        content: content.trim(),
        pinned,
        published,
      })
      setTitle('')
      setContent('')
      setPinned(false)
      setPublished(true)
      setNotice({ tone: 'ok', text: '已发布公告。' })
      await load()
    } catch (error) {
      setNotice(errorToNotice(error, '发布公告失败。'))
    } finally {
      setSaving(false)
    }
  }

  const remove = async (id: number): Promise<void> => {
    if (!confirm('确认删除该公告？')) return
    setSaving(true)
    setNotice(null)
    try {
      await api.deleteAdminAnnouncement(props.token, id)
      setNotice({ tone: 'ok', text: '已删除公告。' })
      await load()
    } catch (error) {
      setNotice(errorToNotice(error, '删除公告失败。'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="admin-announcements-view">
      <section className="panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">Announcements</p>
            <h2>公告</h2>
            <p className="panel-subtitle">简洁优先：标题 + 文本；支持置顶与发布开关。</p>
          </div>
          <div className="inline-actions">
            <button className="ghost-button" type="button" onClick={() => void load()} disabled={loading || saving}>
              刷新
            </button>
            <button className="primary-button" type="button" onClick={() => void create()} disabled={!canCreate}>
              {saving ? '提交中…' : '发布'}
            </button>
          </div>
        </header>

        <NoticeBanner notice={notice} />

        <div className="form-grid">
          <label className="field">
            <span>title</span>
            <input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="公告标题" />
          </label>
          <div className="field" />
          <label className="field" style={{ gridColumn: '1 / -1' }}>
            <span>content</span>
            <textarea value={content} onChange={(e) => setContent(e.target.value)} rows={5} placeholder="公告内容（纯文本）" />
          </label>
          <label className="field">
            <span>pinned</span>
            <select value={String(pinned)} onChange={(e) => setPinned(e.target.value === 'true')}>
              <option value="false">false</option>
              <option value="true">true</option>
            </select>
          </label>
          <label className="field">
            <span>published</span>
            <select value={String(published)} onChange={(e) => setPublished(e.target.value === 'true')}>
              <option value="true">true</option>
              <option value="false">false</option>
            </select>
          </label>
        </div>
      </section>

      <section className="panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">List</p>
            <h2>已存在公告</h2>
            <p className="panel-subtitle">按发布时间倒序。</p>
          </div>
        </header>

        {loading ? <div className="empty-state">加载中…</div> : null}
        {!loading && items.length === 0 ? <div className="empty-state">暂无公告</div> : null}
        {!loading && items.length ? (
          <div className="compact-list">
            {items.map((item) => (
              <article key={item.id} className="entry-card">
                <div className="badge-row">
                  {item.pinned ? <span className="badge badge-accent">Pinned</span> : null}
                  <span className="badge">{item.published ? 'Published' : 'Draft'}</span>
                  {item.published_at ? <span className="badge">{item.published_at}</span> : null}
                </div>
                <strong>{item.title}</strong>
                <p>{item.content}</p>
                <div className="wrap-actions" style={{ marginTop: 12 }}>
                  <button className="ghost-button danger-button" type="button" onClick={() => void remove(item.id)} disabled={saving}>
                    删除
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

