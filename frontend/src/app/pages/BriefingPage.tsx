import React from 'react'

import type { AuthUser, ContestInfo, ContestPhase } from '../../api'
import { Card } from '../components/Card'

export function BriefingPage(props: {
  contest: ContestInfo | null
  phase: ContestPhase | null
  user: AuthUser | null
}): React.JSX.Element {
  const title = props.contest?.title ?? 'YulinSec Recruit CTF'
  const status = props.contest?.status ?? 'unknown'

  return (
    <div style={{ display: 'grid', gap: 16 }}>
      <Card
        title={title}
        subtitle={props.phase?.message ?? '等待后台配置比赛状态说明。'}
        actions={<span style={{ fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--primary-dim)' }}>{status}</span>}
      >
        <div style={{ display: 'grid', gap: 10 }}>
          <div style={{ display: 'grid', gridTemplateColumns: '160px 1fr', gap: 10, fontFamily: 'var(--mono)', fontSize: 12 }}>
            <div style={{ color: 'var(--text-dim)' }}>announcement_visible</div>
            <div>{String(props.phase?.announcement_visible ?? false)}</div>

            <div style={{ color: 'var(--text-dim)' }}>challenge_list_visible</div>
            <div>{String(props.phase?.challenge_list_visible ?? false)}</div>

            <div style={{ color: 'var(--text-dim)' }}>submission_allowed</div>
            <div>{String(props.phase?.submission_allowed ?? false)}</div>

            <div style={{ color: 'var(--text-dim)' }}>runtime_allowed</div>
            <div>{String(props.phase?.runtime_allowed ?? false)}</div>

            <div style={{ color: 'var(--text-dim)' }}>scoreboard_visible</div>
            <div>{String(props.phase?.scoreboard_visible ?? false)}</div>
          </div>

          <div style={{ fontSize: 12, color: 'var(--text-dim)' }}>
            {props.user ? `当前用户: ${props.user.display_name || props.user.username} (${props.user.role})` : '未登录。'}
          </div>
        </div>
      </Card>

      <Card title="Next" subtitle="v2 重写推进顺序">
        <ol style={{ margin: 0, paddingLeft: 18, fontSize: 12, color: 'var(--text-dim)' }}>
          <li>Login/Register (phase aware)</li>
          <li>Board: list + detail + submit</li>
          <li>Runtime panel (instance)</li>
          <li>Scoreboard</li>
          <li>Admin sections</li>
        </ol>
      </Card>
    </div>
  )
}
