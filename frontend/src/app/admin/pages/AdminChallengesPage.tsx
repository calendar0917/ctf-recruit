import React, { useEffect, useMemo, useState } from 'react'

import { api, type AdminChallengeInput, type AdminChallengeSummary } from '../../../api'
import { NoticeBanner } from '../../components/NoticeBanner'
import type { Notice } from '../../utils/errors'
import { errorToNotice } from '../../utils/errors'

const CHALLENGE_STATUSES = ['draft', 'review', 'ready', 'published'] as const
const DIFFICULTIES = ['easy', 'normal', 'hard'] as const

function defaultChallengeInput(): AdminChallengeInput {
  return {
    slug: '',
    title: '',
    category_slug: 'web',
    description: '',
    points: 100,
    difficulty: 'normal',
    flag_type: 'static',
    flag_value: 'flag{...}',
    dynamic_enabled: false,
    status: 'draft',
    visible: false,
    sort_order: 10,
    runtime_config: {
      enabled: false,
      image_name: '',
      exposed_protocol: 'http',
      container_port: 80,
      default_ttl_seconds: 1800,
      max_renew_count: 1,
      memory_limit_mb: 256,
      cpu_limit_millicores: 500,
      max_active_instances: 100,
      user_cooldown_seconds: 30,
      env: {},
      command: [],
    },
  }
}

export function AdminChallengesPage(props: { token: string }): React.JSX.Element {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)

  const [items, setItems] = useState<AdminChallengeSummary[]>([])
  const [activeID, setActiveID] = useState<number | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [draft, setDraft] = useState<AdminChallengeInput>(defaultChallengeInput)

  const loadList = async (): Promise<void> => {
    setLoading(true)
    setNotice(null)
    try {
      const response = await api.adminChallenges(props.token)
      setItems(response.items)
      if (!activeID && response.items.length) {
        setActiveID(response.items[0].id)
      }
    } catch (error) {
      setNotice(errorToNotice(error, '题目列表加载失败。'))
    } finally {
      setLoading(false)
    }
  }

  const loadDetail = async (id: number): Promise<void> => {
    setDetailLoading(true)
    setNotice(null)
    try {
      const response = await api.adminChallenge(props.token, id)
      const ch = response.challenge
      setDraft({
        slug: ch.slug,
        title: ch.title,
        category_slug: ch.category,
        description: ch.description,
        points: ch.points,
        difficulty: ch.difficulty,
        flag_type: ch.flag_type,
        flag_value: ch.flag_value,
        dynamic_enabled: ch.dynamic_enabled,
        status: ch.status,
        visible: ch.visible,
        sort_order: ch.sort_order,
        runtime_config: ch.runtime_config,
      })
    } catch (error) {
      setNotice(errorToNotice(error, '题目详情加载失败。'))
    } finally {
      setDetailLoading(false)
    }
  }

  useEffect(() => {
    void loadList()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (activeID) void loadDetail(activeID)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeID])

  const activeSummary = useMemo(() => items.find((item) => item.id === activeID) ?? null, [activeID, items])

  const save = async (): Promise<void> => {
    if (!activeID) return
    setSaving(true)
    setNotice(null)
    try {
      await api.updateAdminChallenge(props.token, activeID, draft)
      setNotice({ tone: 'ok', text: '已保存题目。' })
      await loadList()
    } catch (error) {
      setNotice(errorToNotice(error, '保存题目失败。'))
    } finally {
      setSaving(false)
    }
  }

  const create = async (): Promise<void> => {
    setSaving(true)
    setNotice(null)
    try {
      const response = await api.createAdminChallenge(props.token, draft)
      setNotice({ tone: 'ok', text: '已创建题目。' })
      await loadList()
      setActiveID(response.challenge.id)
    } catch (error) {
      setNotice(errorToNotice(error, '创建题目失败。'))
    } finally {
      setSaving(false)
    }
  }

  const uploadAttachment = async (file: File): Promise<void> => {
    if (!activeID) return
    setSaving(true)
    setNotice(null)
    try {
      await api.uploadAdminAttachment(props.token, activeID, file)
      setNotice({ tone: 'ok', text: `已上传附件：${file.name}` })
      await loadDetail(activeID)
    } catch (error) {
      setNotice(errorToNotice(error, '上传附件失败。'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="player-board-shell workspace-grid admin-grid">
      <aside className="panel rail-panel">
        <header className="panel-head">
          <div>
            <p className="eyebrow">Challenges</p>
            <h2>题目</h2>
            <p className="panel-subtitle">列表 + 编辑（后端会按角色限制 author 可见范围）。</p>
          </div>
        </header>

        <NoticeBanner notice={notice} />

        <div className="board-list-toolbar" style={{ marginTop: 8 }}>
          <div className="board-filter-meta">{loading ? '加载中…' : `${items.length} items`}</div>
          <div className="wrap-actions">
            <button className="ghost-button" type="button" onClick={() => void loadList()} disabled={loading || saving}>
              刷新
            </button>
            <button
              className="ghost-button"
              type="button"
              onClick={() => {
                setActiveID(null)
                setDraft(defaultChallengeInput())
              }}
              disabled={saving}
            >
              新建
            </button>
          </div>
        </div>

        <div className="challenge-card-list" style={{ marginTop: 12 }}>
          {items.map((item) => (
            <button
              key={item.id}
              type="button"
              className={`challenge-card challenge-card-player ${activeID === item.id ? 'active' : ''}`}
              onClick={() => setActiveID(item.id)}
            >
              <strong>{item.title}</strong>
              <div className="challenge-card-subline">
                <small>
                  {item.category} · {item.points} pts · {item.dynamic_enabled ? 'Dyn' : 'Static'}
                </small>
                <small>{item.status}</small>
              </div>
            </button>
          ))}
        </div>
      </aside>

      <section className="content-stack">
        <section className="panel">
          <header className="panel-head">
            <div>
              <p className="eyebrow">Editor</p>
              <h2>{activeSummary ? `#${activeSummary.id} ${activeSummary.title}` : '新建题目'}</h2>
              <p className="panel-subtitle">保存后即写入后端；points 与 status 影响公开列表与得分。</p>
            </div>
            <div className="inline-actions">
              {activeID ? (
                <button className="primary-button" type="button" disabled={saving || detailLoading} onClick={() => void save()}>
                  {saving ? '保存中…' : '保存'}
                </button>
              ) : (
                <button className="primary-button" type="button" disabled={saving} onClick={() => void create()}>
                  {saving ? '创建中…' : '创建'}
                </button>
              )}
            </div>
          </header>

          {detailLoading ? <div className="empty-state">加载详情中…</div> : null}

          <div className="form-grid">
            <label className="field">
              <span>slug</span>
              <input value={draft.slug} onChange={(e) => setDraft((v) => ({ ...v, slug: e.target.value }))} placeholder="web-welcome" />
            </label>
            <label className="field">
              <span>title</span>
              <input value={draft.title} onChange={(e) => setDraft((v) => ({ ...v, title: e.target.value }))} placeholder="Welcome" />
            </label>

            <label className="field">
              <span>category</span>
              <input value={draft.category_slug} onChange={(e) => setDraft((v) => ({ ...v, category_slug: e.target.value }))} placeholder="web" />
            </label>

            <label className="field">
              <span>difficulty</span>
              <select value={draft.difficulty} onChange={(e) => setDraft((v) => ({ ...v, difficulty: e.target.value }))}>
                {DIFFICULTIES.map((d) => (
                  <option key={d} value={d}>
                    {d}
                  </option>
                ))}
              </select>
            </label>

            <label className="field">
              <span>points</span>
              <input
                type="number"
                value={draft.points}
                onChange={(e) => setDraft((v) => ({ ...v, points: Number(e.target.value) }))}
                min={0}
              />
            </label>

            <label className="field">
              <span>status</span>
              <select value={draft.status} onChange={(e) => setDraft((v) => ({ ...v, status: e.target.value }))}>
                {CHALLENGE_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            </label>

            <label className="field" style={{ gridColumn: '1 / -1' }}>
              <span>description</span>
              <textarea value={draft.description} onChange={(e) => setDraft((v) => ({ ...v, description: e.target.value }))} rows={6} />
            </label>

            <label className="field">
              <span>flag_type</span>
              <select value={draft.flag_type} onChange={(e) => setDraft((v) => ({ ...v, flag_type: e.target.value }))}>
                <option value="static">static</option>
                <option value="case_insensitive">case_insensitive</option>
                <option value="regex">regex</option>
              </select>
            </label>
            <label className="field">
              <span>flag_value</span>
              <input value={draft.flag_value} onChange={(e) => setDraft((v) => ({ ...v, flag_value: e.target.value }))} />
            </label>

            <label className="field">
              <span>dynamic_enabled</span>
              <select value={String(draft.dynamic_enabled)} onChange={(e) => setDraft((v) => ({ ...v, dynamic_enabled: e.target.value === 'true' }))}>
                <option value="false">false</option>
                <option value="true">true</option>
              </select>
            </label>
            <label className="field">
              <span>visible</span>
              <select value={String(draft.visible)} onChange={(e) => setDraft((v) => ({ ...v, visible: e.target.value === 'true' }))}>
                <option value="false">false</option>
                <option value="true">true</option>
              </select>
            </label>
          </div>
        </section>

        {activeID ? (
          <section className="panel">
            <header className="panel-head">
              <div>
                <p className="eyebrow">Attachments</p>
                <h2>附件</h2>
                <p className="panel-subtitle">上传后会在选手端按 phase 控制显示。</p>
              </div>
              <div className="inline-actions">
                <label className="ghost-button" style={{ display: 'inline-flex', alignItems: 'center', gap: 10 }}>
                  <input
                    type="file"
                    style={{ display: 'none' }}
                    onChange={(e) => {
                      const file = e.target.files?.[0]
                      if (!file) return
                      void uploadAttachment(file)
                      e.currentTarget.value = ''
                    }}
                  />
                  上传
                </label>
              </div>
            </header>

            <div className="hint-text">上传能力已接通；附件列表刷新会在后端返回详情后显示。</div>
          </section>
        ) : null}
      </section>
    </section>
  )
}

