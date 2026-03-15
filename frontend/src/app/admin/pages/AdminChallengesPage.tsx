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

  const [importRoot, setImportRoot] = useState('../challenges')
  const [importPath, setImportPath] = useState('')
  const [importAttachmentDir, setImportAttachmentDir] = useState('')

  const [buildTemplate, setBuildTemplate] = useState('web-welcome')
  const [buildTag, setBuildTag] = useState('')
  const [buildResult, setBuildResult] = useState<{ stdout: string; stderr: string; exit_code: number; duration_ms: number; command: string[]; error?: string } | null>(null)

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

  const importChallenges = async (): Promise<void> => {
    setSaving(true)
    setNotice(null)
    try {
      const response = await api.adminImportChallenges(props.token, {
        root: importRoot.trim() || undefined,
        path: importPath.trim() || undefined,
        attachment_dir: importAttachmentDir.trim() || undefined,
      })
      setNotice({ tone: 'ok', text: `已导入 ${response.result.imported} 个 challenge：${response.result.slugs.slice(0, 6).join(', ')}${response.result.slugs.length > 6 ? '…' : ''}` })
      await loadList()
    } catch (error) {
      setNotice(errorToNotice(error, '导入失败。'))
    } finally {
      setSaving(false)
    }
  }

  const buildImage = async (): Promise<void> => {
    if (!buildTemplate.trim()) {
      setNotice({ tone: 'neutral', text: '请填写 template。' })
      return
    }
    setSaving(true)
    setNotice(null)
    setBuildResult(null)
    try {
      const response = await api.adminBuildChallengeImage(props.token, {
        template: buildTemplate.trim(),
        tag: buildTag.trim() || undefined,
      })
      setBuildResult({ ...response.result, error: response.error })
      if (response.result.exit_code === 0) {
        setNotice({ tone: 'ok', text: '镜像构建完成。' })
      } else {
        setNotice({ tone: 'danger', text: '镜像构建失败，请查看输出。' })
      }
    } catch (error) {
      setNotice(errorToNotice(error, '镜像构建失败。'))
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

        <details className="detail-row" style={{ marginTop: 12 }}>
          <summary style={{ cursor: 'pointer' }}>
            <strong>导入挑战（server-local）</strong>
          </summary>
          <div className="hint-text" style={{ marginTop: 10 }}>
            用于从仓库内的 `challenge.yaml` 批量导入（类似 `scripts/import-challenges.sh`），不会自动构建镜像。
          </div>
          <div className="form-grid" style={{ marginTop: 12 }}>
            <label className="field" style={{ gridColumn: '1 / -1' }}>
              <span>root</span>
              <input value={importRoot} onChange={(e) => setImportRoot(e.target.value)} placeholder="./challenges" />
              <small className="hint-text">扫描目录，自动发现所有 `challenge.yaml`。</small>
            </label>
            <label className="field" style={{ gridColumn: '1 / -1' }}>
              <span>path（可选）</span>
              <input value={importPath} onChange={(e) => setImportPath(e.target.value)} placeholder="challenges/templates/web-welcome/challenge.yaml" />
              <small className="hint-text">填写则只导入单个 spec，优先级高于 root。</small>
            </label>
            <label className="field" style={{ gridColumn: '1 / -1' }}>
              <span>attachment_dir（可选）</span>
              <input value={importAttachmentDir} onChange={(e) => setImportAttachmentDir(e.target.value)} placeholder="/tmp/ctf-attachments" />
              <small className="hint-text">不填则用后端默认配置。</small>
            </label>
          </div>
          <div className="wrap-actions" style={{ marginTop: 12 }}>
            <button className="primary-button" type="button" disabled={saving} onClick={() => void importChallenges()}>
              {saving ? '导入中…' : '开始导入'}
            </button>
          </div>
        </details>

        <details className="detail-row" style={{ marginTop: 12 }}>
          <summary style={{ cursor: 'pointer' }}>
            <strong>构建镜像（server-exec）</strong>
          </summary>
          <div className="hint-text" style={{ marginTop: 10 }}>
            受限执行：只允许构建 `../challenges/templates/&lt;template&gt;` 下的 Dockerfile。
          </div>
          <div className="form-grid" style={{ marginTop: 12 }}>
            <label className="field" style={{ gridColumn: '1 / -1' }}>
              <span>template</span>
              <input value={buildTemplate} onChange={(e) => setBuildTemplate(e.target.value)} placeholder="web-welcome" />
              <small className="hint-text">对应 templates 目录名。</small>
            </label>
            <label className="field" style={{ gridColumn: '1 / -1' }}>
              <span>tag（可选）</span>
              <input value={buildTag} onChange={(e) => setBuildTag(e.target.value)} placeholder="ctf/web-welcome:dev" />
              <small className="hint-text">不填则默认 `ctf/&lt;template&gt;:dev`。</small>
            </label>
          </div>
          <div className="wrap-actions" style={{ marginTop: 12 }}>
            <button className="primary-button" type="button" disabled={saving} onClick={() => void buildImage()}>
              {saving ? '构建中…' : '开始构建'}
            </button>
          </div>

          {buildResult ? (
            <div className="detail-stack" style={{ marginTop: 12 }}>
              <div className="hint-text">exit_code {buildResult.exit_code} · {buildResult.duration_ms}ms</div>
              <pre className="code-block">{(buildResult.stdout || '').trim() || '(no stdout)'}</pre>
              <pre className="code-block">{(buildResult.stderr || '').trim() || '(no stderr)'}</pre>
            </div>
          ) : null}
        </details>

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
