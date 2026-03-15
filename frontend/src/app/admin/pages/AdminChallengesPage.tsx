import React, { useEffect, useMemo, useState } from 'react'

import { api, type AdminAttachment, type AdminChallengeAuthor, type AdminChallengeInput, type AdminChallengeSummary } from '../../../api'
import { NoticeBanner } from '../../components/NoticeBanner'
import type { Notice } from '../../utils/errors'
import { errorToNotice } from '../../utils/errors'

import { hasAdminPermission } from '../utils/permissions'

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const idx = Math.min(units.length - 1, Math.max(0, Math.floor(Math.log(bytes) / Math.log(1024))))
  const value = bytes / 1024 ** idx
  const fixed = value >= 10 || idx === 0 ? 0 : 1
  return `${value.toFixed(fixed)} ${units[idx]}`
}

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
      max_renew_count: 0,
      memory_limit_mb: 256,
      cpu_limit_millicores: 500,
      max_active_instances: 0,
      user_cooldown_seconds: 0,
      env: {},
      command: [],
    },
  }
}

export function AdminChallengesPage(props: { token: string }): React.JSX.Element {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)

  const [lastSavedDraft, setLastSavedDraft] = useState<AdminChallengeInput>(defaultChallengeInput)
  const [dirtyFields, setDirtyFields] = useState<Set<string>>(() => new Set())

  const [items, setItems] = useState<AdminChallengeSummary[]>([])
  const [activeID, setActiveID] = useState<number | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [draft, setDraft] = useState<AdminChallengeInput>(defaultChallengeInput)

  const [attachments, setAttachments] = useState<AdminAttachment[]>([])

  const [authors, setAuthors] = useState<AdminChallengeAuthor[]>([])
  const [authorsLoading, setAuthorsLoading] = useState(false)
  const [usersLoading, setUsersLoading] = useState(false)
  const [authorSearch, setAuthorSearch] = useState('')
  const [authorCandidates, setAuthorCandidates] = useState<Array<{ id: number; username: string; display_name: string; email: string; role: string }>>([])

  const [importRoot, setImportRoot] = useState('../challenges')
  const [importAttachmentDir, setImportAttachmentDir] = useState('')

  const [importResult, setImportResult] = useState<null | { imported: number; slugs: string[] }>(null)

  const [buildResult, setBuildResult] = useState<{ stdout: string; stderr: string; exit_code: number; duration_ms: number; command: string[]; error?: string } | null>(null)

  const [myInstance, setMyInstance] = useState<null | {
    status: string
    access_url?: string
    host_port?: number
    renew_count: number
    started_at: string
    expires_at: string
    terminated_at?: string | null
  }>(null)
  const [myInstanceLoading, setMyInstanceLoading] = useState(false)

  const [meRole, setMeRole] = useState('')

  const markDirty = (key: string): void => {
    setDirtyFields((current) => {
      const next = new Set(current)
      next.add(key)
      return next
    })
  }

  const patchDraft = (key: string, update: Partial<AdminChallengeInput>): void => {
    setDraft((current) => ({ ...current, ...update }))
    markDirty(key)
  }

  const patchRuntime = (key: string, update: Partial<NonNullable<AdminChallengeInput['runtime_config']>>): void => {
    setDraft((current) => ({
      ...current,
      runtime_config: {
        ...(current.runtime_config ?? defaultChallengeInput().runtime_config!),
        ...update,
      },
    }))
    markDirty(key)
  }

  const patchRuntimeEnv = (key: string, update: Record<string, string>): void => {
    patchRuntime(key, { env: update })
  }

  const activeSummary = useMemo(() => items.find((item) => item.id === activeID) ?? null, [activeID, items])

  const deriveTemplateFromDraft = useMemo(() => {
    const slug = draft.slug.trim()
    if (slug) return slug
    if (activeSummary?.slug?.trim()) return activeSummary.slug.trim()
    return ''
  }, [activeSummary?.slug, draft.slug])

  const canBuildForDraft = useMemo(() => {
    if (!draft.dynamic_enabled) return false
    if (!deriveTemplateFromDraft) return false
    return true
  }, [deriveTemplateFromDraft, draft.dynamic_enabled])

  const canImportForDraft = useMemo(() => {
    if (!deriveTemplateFromDraft) return false
    return true
  }, [deriveTemplateFromDraft])

  const canDoServerLocal = useMemo(() => {
    // Frontend helper only (backend still enforces).
    return hasAdminPermission({ role: meRole } as any, 'instance:write')
  }, [meRole])

  const canManageAuthors = useMemo(() => {
    // Backend: PUT authors is admin-only.
    return hasAdminPermission({ role: meRole } as any, 'user:write')
  }, [meRole])

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

  const loadMe = async (): Promise<void> => {
    try {
      const response = await api.me(props.token)
      setMeRole(response.user.role)
    } catch {
      setMeRole('')
    }
  }

  const loadDetail = async (id: number): Promise<void> => {
    setDetailLoading(true)
    setNotice(null)
    try {
      const response = await api.adminChallenge(props.token, id)
      const ch = response.challenge
      const nextDraft: AdminChallengeInput = {
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
      }
      setDraft(nextDraft)
      setLastSavedDraft(nextDraft)
      setDirtyFields(new Set())

      // Keep authors in a separate state so admin can edit without having to patch challenge detail schema.
      setAuthors(ch.authors ?? [])

      setAttachments(ch.attachments ?? [])

      setMyInstance(null)
    } catch (error) {
      setNotice(errorToNotice(error, '题目详情加载失败。'))
    } finally {
      setDetailLoading(false)
    }
  }

  const loadMyInstance = async (): Promise<void> => {
    if (!activeID) return
    setMyInstanceLoading(true)
    setNotice(null)
    try {
      const response = await api.adminGetMyInstance(props.token, activeID)
      setMyInstance(response)
    } catch (error) {
      setMyInstance(null)
      setNotice(errorToNotice(error, '实例查询失败。'))
    } finally {
      setMyInstanceLoading(false)
    }
  }

  const startMyInstance = async (): Promise<void> => {
    if (!activeID) return
    setMyInstanceLoading(true)
    setNotice(null)
    try {
      const response = await api.adminStartMyInstance(props.token, activeID)
      setMyInstance(response)
    } catch (error) {
      setNotice(errorToNotice(error, '实例创建失败。'))
    } finally {
      setMyInstanceLoading(false)
    }
  }

  const renewMyInstance = async (): Promise<void> => {
    if (!activeID) return
    setMyInstanceLoading(true)
    setNotice(null)
    try {
      const response = await api.adminRenewMyInstance(props.token, activeID)
      setMyInstance(response)
    } catch (error) {
      setNotice(errorToNotice(error, '实例续期失败。'))
    } finally {
      setMyInstanceLoading(false)
    }
  }

  const deleteMyInstance = async (): Promise<void> => {
    if (!activeID) return
    setMyInstanceLoading(true)
    setNotice(null)
    try {
      const response = await api.adminDeleteMyInstance(props.token, activeID)
      setMyInstance(response)
    } catch (error) {
      setNotice(errorToNotice(error, '实例删除失败。'))
    } finally {
      setMyInstanceLoading(false)
    }
  }

  const loadAuthorCandidates = async (): Promise<void> => {
    if (!canManageAuthors) {
      setAuthorCandidates([])
      return
    }
    setUsersLoading(true)
    setNotice(null)
    try {
      const response = await api.adminUsers(props.token)
      const candidates = response.items
        .filter((u) => u.role === 'admin' || u.role === 'author')
        .map((u) => ({ id: u.id, username: u.username, display_name: u.display_name, email: u.email, role: u.role }))
      setAuthorCandidates(candidates)
    } catch (error) {
      setNotice(errorToNotice(error, '加载作者候选失败。'))
    } finally {
      setUsersLoading(false)
    }
  }

  const loadAuthors = async (): Promise<void> => {
    if (!activeID) return
    setAuthorsLoading(true)
    setNotice(null)
    try {
      const response = await api.adminChallengeAuthors(props.token, activeID)
      setAuthors(response.items)
    } catch (error) {
      setNotice(errorToNotice(error, '加载题目作者失败。'))
    } finally {
      setAuthorsLoading(false)
    }
  }

  const saveAuthors = async (): Promise<void> => {
    if (!activeID) return
    if (!canManageAuthors) {
      setNotice({ tone: 'neutral', text: '当前账号无权限修改作者集合。' })
      return
    }
    setSaving(true)
    setNotice(null)
    try {
      const userIDs = authors.map((a) => a.user_id)
      await api.updateAdminChallengeAuthors(props.token, activeID, userIDs)
      setNotice({ tone: 'ok', text: '已保存题目作者。' })
      await loadAuthors()
      await loadList()
    } catch (error) {
      setNotice(errorToNotice(error, '保存作者失败。'))
    } finally {
      setSaving(false)
    }
  }

  useEffect(() => {
    void loadList()
    void loadMe()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    if (activeID) void loadDetail(activeID)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeID])

  useEffect(() => {
    void loadAuthorCandidates()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canManageAuthors])

  const save = async (): Promise<void> => {
    if (!activeID) return
    setSaving(true)
    setNotice(null)
    try {
      await api.updateAdminChallenge(props.token, activeID, draft)
      setNotice({ tone: 'ok', text: '已保存题目。' })
      setLastSavedDraft(draft)
      setDirtyFields(new Set())
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
      setLastSavedDraft(draft)
      setDirtyFields(new Set())
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

  const importForActiveChallenge = async (): Promise<void> => {
    if (!canDoServerLocal) {
      setNotice({ tone: 'neutral', text: '当前账号无权限执行 server-local 导入（仅 admin/ops）。' })
      return
    }
    const template = deriveTemplateFromDraft
    if (!template.trim()) {
      setNotice({ tone: 'neutral', text: '请先填写 slug（用于定位 templates 目录）。' })
      return
    }
    if (!confirm(`将从 templates/${template}/challenge.yaml 导入并覆盖当前题目信息（含 runtime_config）。继续？`)) return

    setSaving(true)
    setNotice(null)
    setImportResult(null)
    try {
      const root = importRoot.trim() || '../challenges'
      const normalizedRoot = root.replace(/\/+$/, '')
      const response = await api.adminImportChallenges(props.token, {
        root,
        path: `${normalizedRoot}/templates/${template}/challenge.yaml`,
        attachment_dir: importAttachmentDir.trim() || undefined,
      })
      setNotice({ tone: 'ok', text: `已导入：${response.result.slugs.join(', ')}` })
      setImportResult(response.result)
      await loadList()
      if (activeID) {
        await loadDetail(activeID)
      }
    } catch (error) {
      setNotice(errorToNotice(error, '导入失败。'))
    } finally {
      setSaving(false)
    }
  }

  const importAllFromRoot = async (): Promise<void> => {
    if (!canDoServerLocal) {
      setNotice({ tone: 'neutral', text: '当前账号无权限执行 server-local 导入（仅 admin/ops）。' })
      return
    }
    const root = importRoot.trim() || '../challenges'
    if (!confirm(`将扫描 ${root} 下所有 challenge.yaml 并幂等导入/更新题目（含 runtime_config）。继续？`)) return

    setSaving(true)
    setNotice(null)
    setImportResult(null)
    try {
      const response = await api.adminImportChallenges(props.token, {
        root,
        attachment_dir: importAttachmentDir.trim() || undefined,
      })
      setImportResult(response.result)
      setNotice({ tone: 'ok', text: `批量导入完成：${response.result.imported} 题` })
      await loadList()
      if (activeID) {
        await loadDetail(activeID)
      }
    } catch (error) {
      setNotice(errorToNotice(error, '批量导入失败。'))
    } finally {
      setSaving(false)
    }
  }

  const buildImageForActiveChallenge = async (): Promise<void> => {
    if (!canDoServerLocal) {
      setNotice({ tone: 'neutral', text: '当前账号无权限构建镜像。' })
      return
    }
    const template = deriveTemplateFromDraft
    if (!template.trim()) {
      setNotice({ tone: 'neutral', text: '请先填写 slug（用于定位 templates 目录）。' })
      return
    }
    if (!draft.dynamic_enabled) {
      setNotice({ tone: 'neutral', text: '该题未启用 dynamic_enabled；无需构建镜像。' })
      return
    }
    const tag = (draft.runtime_config?.image_name ?? '').trim()
    if (!tag) {
      setNotice({ tone: 'neutral', text: '请先在 Runtime 配置中填写 image_name（用于 docker tag）。' })
      return
    }

    setSaving(true)
    setNotice(null)
    setBuildResult(null)
    try {
      const response = await api.adminBuildChallengeImage(props.token, {
        template,
        tag,
      })
      setBuildResult({ ...response.result, error: response.error })
      if (response.result.exit_code === 0) {
        setNotice({ tone: 'ok', text: `镜像构建完成：${tag}` })
      } else {
        setNotice({ tone: 'danger', text: '镜像构建失败，请查看输出。' })
      }
    } catch (error) {
      setNotice(errorToNotice(error, '镜像构建失败。'))
    } finally {
      setSaving(false)
    }
  }

  const runtimeEnabled = Boolean(draft.runtime_config?.enabled)
  const runtimeEnv = draft.runtime_config?.env ?? {}
  const runtimeCommandText = useMemo(() => {
    const cmd = draft.runtime_config?.command ?? []
    return cmd.join(' ')
  }, [draft.runtime_config?.command])

  const runtimeDirty = useMemo(() => {
    for (const key of dirtyFields) {
      if (key.startsWith('runtime.')) return true
    }
    return false
  }, [dirtyFields])

  const attachmentSummary = useMemo(() => {
    if (!attachments.length) return ''
    const totalBytes = attachments.reduce((sum, item) => sum + (Number.isFinite(item.size_bytes) ? item.size_bytes : 0), 0)
    return `${attachments.length} files · ${formatBytes(totalBytes)}`
  }, [attachments])

  const filteredCandidates = useMemo(() => {
    const q = authorSearch.trim().toLowerCase()
    if (!q) return authorCandidates
    return authorCandidates.filter((u) => {
      return (
        u.username.toLowerCase().includes(q) ||
        u.email.toLowerCase().includes(q) ||
        (u.display_name ?? '').toLowerCase().includes(q)
      )
    })
  }, [authorCandidates, authorSearch])

  const authorIDSet = useMemo(() => new Set(authors.map((a) => a.user_id)), [authors])

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
            <strong>导入 / 构建</strong>
          </summary>

          <div className="hint-text" style={{ marginTop: 10 }}>
            说明：动态题运行依赖 `runtime_config.image_name`（数据库）。导入与构建都默认绑定当前题目的 slug。
          </div>

          <div className="form-grid" style={{ marginTop: 12 }}>
            <label className="field" style={{ gridColumn: '1 / -1' }}>
              <span>root</span>
              <input value={importRoot} onChange={(e) => setImportRoot(e.target.value)} placeholder="../challenges" />
              <small className="hint-text">用于定位 templates 目录（server-local）。</small>
            </label>
            <label className="field" style={{ gridColumn: '1 / -1' }}>
              <span>attachment_dir（可选）</span>
              <input value={importAttachmentDir} onChange={(e) => setImportAttachmentDir(e.target.value)} placeholder="/tmp/ctf-attachments" />
              <small className="hint-text">不填则用后端默认配置。</small>
            </label>
          </div>

          <div className="wrap-actions" style={{ marginTop: 12 }}>
            <button className="primary-button" type="button" disabled={saving || !canDoServerLocal || !canImportForDraft} onClick={() => void importForActiveChallenge()}>
              {saving ? '导入中…' : `导入当前题（templates/${deriveTemplateFromDraft || '…'}/challenge.yaml）`}
            </button>
            <button className="ghost-button" type="button" disabled={saving || !canDoServerLocal} onClick={() => void importAllFromRoot()}>
              {saving ? '导入中…' : '批量导入（扫描 root）'}
            </button>
            <button className="primary-button" type="button" disabled={saving || !canDoServerLocal || !canBuildForDraft} onClick={() => void buildImageForActiveChallenge()}>
              {saving ? '构建中…' : `构建当前题（${(draft.runtime_config?.image_name ?? '').trim() || '请先填 image_name'}）`}
            </button>
          </div>

          {importResult ? (
            <div className="detail-stack" style={{ marginTop: 12 }}>
              <div className="hint-text">imported {importResult.imported}</div>
              <pre className="code-block">{importResult.slugs.join('\n') || '(no slugs)'}</pre>
            </div>
          ) : null}

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
              {dirtyFields.size ? <span className="badge badge-accent">未保存 {dirtyFields.size}</span> : <span className="badge">已同步</span>}
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
              <input value={draft.slug} onChange={(e) => patchDraft('meta.slug', { slug: e.target.value })} placeholder="web-welcome" />
            </label>
            <label className="field">
              <span>title</span>
              <input value={draft.title} onChange={(e) => patchDraft('meta.title', { title: e.target.value })} placeholder="Welcome" />
            </label>

            <label className="field">
              <span>category</span>
              <input value={draft.category_slug} onChange={(e) => patchDraft('meta.category', { category_slug: e.target.value })} placeholder="web" />
            </label>

            <label className="field">
              <span>difficulty</span>
              <select value={draft.difficulty} onChange={(e) => patchDraft('meta.difficulty', { difficulty: e.target.value })}>
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
                onChange={(e) => patchDraft('meta.points', { points: Number(e.target.value) })}
                min={0}
              />
            </label>

            <label className="field">
              <span>status</span>
              <select value={draft.status} onChange={(e) => patchDraft('meta.status', { status: e.target.value })}>
                {CHALLENGE_STATUSES.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            </label>

            <label className="field" style={{ gridColumn: '1 / -1' }}>
              <span>description</span>
              <textarea value={draft.description} onChange={(e) => patchDraft('meta.description', { description: e.target.value })} rows={6} />
            </label>

            <label className="field">
              <span>flag_type</span>
              <select value={draft.flag_type} onChange={(e) => patchDraft('flag.type', { flag_type: e.target.value })}>
                <option value="static">static</option>
                <option value="case_insensitive">case_insensitive</option>
                <option value="regex">regex</option>
              </select>
            </label>
            <label className="field">
              <span>flag_value</span>
              <input value={draft.flag_value} onChange={(e) => patchDraft('flag.value', { flag_value: e.target.value })} />
            </label>

            <label className="field">
              <span>dynamic_enabled</span>
              <select value={String(draft.dynamic_enabled)} onChange={(e) => patchDraft('meta.dynamic_enabled', { dynamic_enabled: e.target.value === 'true' })}>
                <option value="false">false</option>
                <option value="true">true</option>
              </select>
            </label>
            <label className="field">
              <span>visible</span>
              <select value={String(draft.visible)} onChange={(e) => patchDraft('meta.visible', { visible: e.target.value === 'true' })}>
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
                <p className="eyebrow">Runtime</p>
                <h2>动态运行配置</h2>
                <p className="panel-subtitle">一个 dynamic 题目最终对应一个 `image_name`（docker tag）。构建镜像只是在把模板 build 成这个 tag。</p>
              </div>
              <div className="inline-actions">
                {runtimeDirty ? <span className="badge badge-accent">未保存</span> : <span className="badge">已同步</span>}
                <button
                  className="ghost-button"
                  type="button"
                  disabled={saving || detailLoading}
                  onClick={() => {
                    setDraft(lastSavedDraft)
                    setDirtyFields(new Set())
                    setNotice({ tone: 'neutral', text: '已还原到上次保存版本。' })
                  }}
                >
                  撤销
                </button>
                <button className="primary-button" type="button" disabled={saving || detailLoading} onClick={() => void save()}>
                  {saving ? '保存中…' : '保存'}
                </button>
              </div>
            </header>

            <div className="form-grid" style={{ gridTemplateColumns: 'repeat(2, minmax(0, 1fr))' }}>
              <label className="field">
                <span>enabled</span>
                <select value={String(runtimeEnabled)} onChange={(e) => patchRuntime('runtime.enabled', { enabled: e.target.value === 'true' })}>
                  <option value="false">false</option>
                  <option value="true">true</option>
                </select>
              </label>
              <label className="field">
                <span>exposed_protocol</span>
                <select
                  value={draft.runtime_config?.exposed_protocol ?? 'http'}
                  onChange={(e) => patchRuntime('runtime.exposed_protocol', { exposed_protocol: e.target.value })}
                >
                  <option value="http">http</option>
                  <option value="https">https</option>
                  <option value="tcp">tcp</option>
                  <option value="udp">udp</option>
                </select>
              </label>

              <label className="field" style={{ gridColumn: '1 / -1' }}>
                <span>image_name</span>
                <input
                  value={draft.runtime_config?.image_name ?? ''}
                  onChange={(e) => patchRuntime('runtime.image_name', { image_name: e.target.value })}
                  placeholder="ctf/web-welcome:dev"
                />
                <small className="hint-text">建议与 templates 目录同名；动态实例启动时会直接拉取/使用该 tag。</small>
              </label>

              <label className="field">
                <span>container_port</span>
                <input
                  type="number"
                  min={1}
                  value={draft.runtime_config?.container_port ?? 80}
                  onChange={(e) => patchRuntime('runtime.container_port', { container_port: Number(e.target.value) })}
                />
              </label>
              <label className="field">
                <span>default_ttl_seconds</span>
                <input
                  type="number"
                  min={1}
                  value={draft.runtime_config?.default_ttl_seconds ?? 1800}
                  onChange={(e) => patchRuntime('runtime.default_ttl_seconds', { default_ttl_seconds: Number(e.target.value) })}
                />
              </label>

              <label className="field">
                <span>memory_limit_mb</span>
                <input
                  type="number"
                  min={16}
                  value={draft.runtime_config?.memory_limit_mb ?? 256}
                  onChange={(e) => patchRuntime('runtime.memory_limit_mb', { memory_limit_mb: Number(e.target.value) })}
                />
              </label>
              <label className="field">
                <span>cpu_limit_millicores</span>
                <input
                  type="number"
                  min={50}
                  value={draft.runtime_config?.cpu_limit_millicores ?? 500}
                  onChange={(e) => patchRuntime('runtime.cpu_limit_millicores', { cpu_limit_millicores: Number(e.target.value) })}
                />
              </label>

              <label className="field">
                <span>max_renew_count</span>
                <input
                  type="number"
                  min={0}
                  value={draft.runtime_config?.max_renew_count ?? 0}
                  onChange={(e) => patchRuntime('runtime.max_renew_count', { max_renew_count: Number(e.target.value) })}
                />
              </label>
              <label className="field">
                <span>max_active_instances</span>
                <input
                  type="number"
                  min={0}
                  value={draft.runtime_config?.max_active_instances ?? 0}
                  onChange={(e) => patchRuntime('runtime.max_active_instances', { max_active_instances: Number(e.target.value) })}
                />
              </label>

              <label className="field">
                <span>user_cooldown_seconds</span>
                <input
                  type="number"
                  min={0}
                  value={draft.runtime_config?.user_cooldown_seconds ?? 0}
                  onChange={(e) => patchRuntime('runtime.user_cooldown_seconds', { user_cooldown_seconds: Number(e.target.value) })}
                />
              </label>

              <label className="field" style={{ gridColumn: '1 / -1' }}>
                <span>command</span>
                <input
                  value={runtimeCommandText}
                  onChange={(e) => {
                    const nextText = e.target.value
                    const parts = nextText
                      .split(' ')
                      .map((p) => p.trim())
                      .filter(Boolean)
                    patchRuntime('runtime.command', { command: parts })
                  }}
                  placeholder="/app/start.sh --port 80"
                />
                <small className="hint-text">简单按空格切分（不支持引号转义）；复杂 command 建议通过导入 spec 写入。</small>
              </label>
            </div>

            <div className="divider-line" style={{ marginTop: 14 }}>
              <span>Env</span>
            </div>

            <div className="detail-stack" style={{ marginTop: 12 }}>
              {Object.keys(runtimeEnv).length === 0 ? <div className="hint-text">暂无 env（可添加）。</div> : null}
              {Object.entries(runtimeEnv).map(([k, v]) => (
                <div key={k} className="row-card" style={{ gridTemplateColumns: 'minmax(0, 1fr)', padding: 12 }}>
                  <div className="hint-text" style={{ marginBottom: 10 }}>{k}</div>
                  <input
                    value={v}
                    onChange={(e) => patchRuntimeEnv(`runtime.env.${k}`, { ...runtimeEnv, [k]: e.target.value })}
                  />
                  <div className="wrap-actions" style={{ marginTop: 10 }}>
                    <button
                      className="ghost-button danger-button"
                      type="button"
                      onClick={() => {
                        const next = { ...runtimeEnv }
                        delete next[k]
                        patchRuntimeEnv(`runtime.env.${k}`, next)
                      }}
                    >
                      删除
                    </button>
                  </div>
                </div>
              ))}

              <div className="wrap-actions" style={{ marginTop: 12 }}>
                <button
                  className="ghost-button"
                  type="button"
                  onClick={() => {
                    const key = prompt('新增 env key（例如 MODE）')
                    if (!key) return
                    const trimmed = key.trim()
                    if (!trimmed) return
                    if (trimmed in runtimeEnv) {
                      setNotice({ tone: 'neutral', text: '该 key 已存在。' })
                      return
                    }
                    patchRuntimeEnv(`runtime.env.${trimmed}`, { ...runtimeEnv, [trimmed]: '' })
                  }}
                >
                  添加变量
                </button>
                <button
                  className="primary-button"
                  type="button"
                  disabled={saving || !canBuildForDraft}
                  onClick={() => void buildImageForActiveChallenge()}
                  title={!canBuildForDraft ? '需要 dynamic_enabled=true 且 slug 可用' : undefined}
                >
                  构建当前题镜像
                </button>
              </div>
            </div>

            <div className="divider-line" style={{ marginTop: 14 }}>
              <span>验证闭环</span>
            </div>

            <div className="detail-row" style={{ marginTop: 12 }}>
              <div className="badge-row">
                <span className="badge">status {myInstance?.status ?? '—'}</span>
                {myInstance?.host_port ? <span className="badge badge-accent">:{myInstance.host_port}</span> : null}
                <span className="badge">renew {myInstance?.renew_count ?? 0}</span>
              </div>

              {myInstance?.access_url ? (
                <div className="wrap-actions" style={{ marginTop: 12 }}>
                  <a className="primary-button" href={myInstance.access_url} target="_blank" rel="noreferrer">
                    打开实例
                  </a>
                </div>
              ) : null}

              <div className="wrap-actions" style={{ marginTop: 12 }}>
                <button className="ghost-button" type="button" disabled={myInstanceLoading || !canDoServerLocal} onClick={() => void loadMyInstance()}>
                  {myInstanceLoading ? '加载中…' : '查询实例'}
                </button>
                <button className="primary-button" type="button" disabled={myInstanceLoading || !canDoServerLocal} onClick={() => void startMyInstance()}>
                  {myInstanceLoading ? '启动中…' : '启动/复用'}
                </button>
                <button className="ghost-button" type="button" disabled={myInstanceLoading || !canDoServerLocal} onClick={() => void renewMyInstance()}>
                  续期
                </button>
                <button className="ghost-button danger-button" type="button" disabled={myInstanceLoading || !canDoServerLocal} onClick={() => void deleteMyInstance()}>
                  删除
                </button>
              </div>
              <div className="hint-text" style={{ marginTop: 10 }}>
                管理端验证不受比赛 phase 限制：用于赛前检查镜像/端口/TTL/配额是否正常。
              </div>
            </div>
          </section>
        ) : null}

        {activeID ? (
          <section className="panel">
            <header className="panel-head">
              <div>
                <p className="eyebrow">Attachments</p>
                <h2>附件</h2>
                <p className="panel-subtitle">上传后会在选手端按 phase 控制显示。{attachmentSummary ? `当前：${attachmentSummary}` : ''}</p>
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
                <button className="ghost-button" type="button" disabled={saving || detailLoading} onClick={() => activeID && void loadDetail(activeID)}>
                  刷新
                </button>
              </div>
            </header>

            {attachments.length ? (
              <div className="attachment-list" style={{ marginTop: 12 }}>
                {attachments.map((item: any) => (
                  <a
                    key={item.id}
                    className="attachment-row"
                    href={`/api/v1/admin/challenges/${activeID}/attachments/${item.id}`}
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
              <div className="hint-text">暂无附件。上传后会出现在这里，并可直接下载验证。</div>
            )}
          </section>
        ) : null}

        {activeID ? (
          <section className="panel">
            <header className="panel-head">
              <div>
                <p className="eyebrow">Authors</p>
                <h2>题目负责人</h2>
                <p className="panel-subtitle">用于 author 角色可见性与写权限（仅 admin 可修改作者集合）。</p>
              </div>
              <div className="inline-actions">
                <button className="ghost-button" type="button" disabled={saving || authorsLoading} onClick={() => void loadAuthors()}>
                  {authorsLoading ? '加载中…' : '刷新'}
                </button>
                <button className="primary-button" type="button" disabled={saving || !canManageAuthors} onClick={() => void saveAuthors()}>
                  {saving ? '保存中…' : '保存作者'}
                </button>
              </div>
            </header>

            <div className="form-grid" style={{ gridTemplateColumns: 'repeat(2, minmax(0, 1fr))' }}>
              <label className="field">
                <span>搜索候选</span>
                <input value={authorSearch} onChange={(e) => setAuthorSearch(e.target.value)} placeholder="username / email / display_name" />
              </label>
              <div className="field" />
            </div>

            {usersLoading ? <div className="hint-text">加载候选中…</div> : null}

            <div className="card-list" style={{ marginTop: 12, gridTemplateColumns: 'repeat(2, minmax(0, 1fr))' }}>
              {filteredCandidates.slice(0, 12).map((u) => {
                const active = authorIDSet.has(u.id)
                return (
                  <button
                    key={u.id}
                    type="button"
                    className={`challenge-card challenge-card-player ${active ? 'active' : ''}`}
                    onClick={() => {
                      if (active) {
                        setAuthors((list) => list.filter((a) => a.user_id !== u.id))
                        return
                      }
                      setAuthors((list) => [...list, { user_id: u.id, username: u.username, email: u.email, display_name: u.display_name, role: u.role }])
                    }}
                  >
                    <strong>{u.display_name || u.username}</strong>
                    <div className="challenge-card-subline">
                      <small>@{u.username}</small>
                      <small>{u.role}</small>
                    </div>
                  </button>
                )
              })}
            </div>

            <div className="divider-line" style={{ marginTop: 14 }}>
              <span>当前作者</span>
            </div>

            {authors.length ? (
              <div className="compact-list" style={{ marginTop: 12 }}>
                {authors.map((a) => (
                  <article key={a.user_id} className="entry-card" style={{ padding: 12 }}>
                    <div className="badge-row">
                      <span className="badge">#{a.user_id}</span>
                      <span className="badge">{a.role}</span>
                      <span className="badge">@{a.username}</span>
                      <span className="badge">{a.email}</span>
                    </div>
                    <strong>{a.display_name || a.username}</strong>
                    <div className="wrap-actions" style={{ marginTop: 10 }}>
                      <button className="ghost-button danger-button" type="button" onClick={() => setAuthors((list) => list.filter((x) => x.user_id !== a.user_id))}>
                        移除
                      </button>
                    </div>
                  </article>
                ))}
              </div>
            ) : (
              <div className="hint-text" style={{ marginTop: 12 }}>
                尚未设置作者。author 角色将无法看到/编辑该题（会返回 404）。
              </div>
            )}
          </section>
        ) : null}
      </section>
    </section>
  )
}
