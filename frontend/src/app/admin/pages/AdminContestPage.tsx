import React, { useEffect, useMemo, useState } from 'react'

import type { ContestResponse } from '../../../api'
import { api, type AdminContestInput } from '../../../api'
import { NoticeBanner } from '../../components/NoticeBanner'
import type { Notice } from '../../utils/errors'
import { describeError, errorToNotice } from '../../utils/errors'

export function AdminContestPage(props: { token: string }): React.JSX.Element {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)

  const [data, setData] = useState<ContestResponse | null>(null)

  const [status, setStatus] = useState('running')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')

  const phase = data?.phase

  const load = async (): Promise<void> => {
    setLoading(true)
    setNotice(null)
    try {
      const response = await api.adminContest(props.token)
      setData(response)
      setStatus(response.contest.status)
      setStartsAt(response.contest.starts_at ?? '')
      setEndsAt(response.contest.ends_at ?? '')
    } catch (error) {
      setNotice({ tone: 'danger', text: describeError(error, '加载比赛配置失败。') })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const canSave = useMemo(() => {
    if (saving) return false
    if (!status.trim()) return false
    if (!startsAt.trim() || !endsAt.trim()) return false
    return true
  }, [endsAt, saving, startsAt, status])

  const save = async (): Promise<void> => {
    if (!canSave) return
    setSaving(true)
    setNotice(null)
    try {
      const payload: AdminContestInput = {
        status: status.trim(),
        starts_at: startsAt.trim(),
        ends_at: endsAt.trim(),
      }
      const response = await api.updateAdminContest(props.token, payload)
      setData(response)
      setNotice({ tone: 'ok', text: '已保存比赛配置。' })
    } catch (error) {
      setNotice(errorToNotice(error, '保存失败。'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="panel">
      <header className="panel-head">
        <div>
          <p className="eyebrow">Contest</p>
          <h2>比赛状态</h2>
          <p className="panel-subtitle">控制公开开关：题目、提交、实例、排行榜、注册等由后端 phase 自动推导。</p>
        </div>
        <div className="inline-actions">
          <button className="ghost-button" type="button" onClick={() => void load()} disabled={loading || saving}>
            刷新
          </button>
          <button className="primary-button" type="button" onClick={() => void save()} disabled={!canSave}>
            {saving ? '保存中…' : '保存'}
          </button>
        </div>
      </header>

      <NoticeBanner notice={notice} />

      {loading && !data ? <div className="empty-state">加载中…</div> : null}

      <div className="form-grid" style={{ gridTemplateColumns: 'repeat(2, minmax(0, 1fr))' }}>
        <label className="field">
          <span>status</span>
          <select value={status} onChange={(e) => setStatus(e.target.value)}>
            <option value="draft">draft</option>
            <option value="upcoming">upcoming</option>
            <option value="running">running</option>
            <option value="frozen">frozen</option>
            <option value="ended">ended</option>
          </select>
          <small className="hint-text">切换后端 phase 的核心输入。</small>
        </label>

        <div className="field" />

        <label className="field">
          <span>starts_at</span>
          <input value={startsAt} onChange={(e) => setStartsAt(e.target.value)} placeholder="2026-03-01T00:00:00Z" />
          <small className="hint-text">建议 RFC3339（UTC）。</small>
        </label>

        <label className="field">
          <span>ends_at</span>
          <input value={endsAt} onChange={(e) => setEndsAt(e.target.value)} placeholder="2026-03-31T00:00:00Z" />
          <small className="hint-text">建议 RFC3339（UTC）。</small>
        </label>
      </div>

      {phase ? (
        <div className="detail-row" style={{ marginTop: 14 }}>
          <strong>当前 Phase</strong>
          <div className="scoreboard-personal-strip" style={{ marginTop: 12 }}>
            <div className="runtime-metric">
              <span className="eyebrow">Status</span>
              <strong>{phase.status}</strong>
              <small className="hint-text">{phase.message}</small>
            </div>
            <div className="runtime-metric">
              <span className="eyebrow">Public</span>
              <strong>{phase.challenge_list_visible ? 'On' : 'Off'}</strong>
              <small className="hint-text">题目/个人页面</small>
            </div>
            <div className="runtime-metric">
              <span className="eyebrow">Submit</span>
              <strong>{phase.submission_allowed ? 'On' : 'Off'}</strong>
              <small className="hint-text">Flag 提交</small>
            </div>
          </div>
          <div className="hint-text" style={{ marginTop: 10 }}>
            scoreboard_visible={String(phase.scoreboard_visible)} · runtime_allowed={String(phase.runtime_allowed)} · registration_allowed={String(phase.registration_allowed)}
          </div>
        </div>
      ) : null}
    </section>
  )
}

