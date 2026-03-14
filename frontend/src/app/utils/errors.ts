export type AppError = Error & {
  code?: string
  status?: number
}

export type NoticeTone = 'neutral' | 'ok' | 'danger'

export type Notice = {
  tone: NoticeTone
  text: string
}

export function describeError(error: unknown, fallback: string): string {
  const typed = error as AppError | undefined
  return typed?.message?.trim() ? typed.message : fallback
}

export function readErrorCode(error: unknown): string {
  const typed = error as AppError | undefined
  return typed?.code ?? ''
}

export function isUnauthorized(error: unknown): boolean {
  const typed = error as AppError | undefined
  return typed?.status === 401
}

export function errorToNotice(error: unknown, fallback: string): Notice {
  const code = readErrorCode(error)
  if (code === 'contest_not_public') {
    return { tone: 'neutral', text: '比赛尚未公开或当前内容未开放。' }
  }
  if (code === 'scoreboard_not_public') {
    return { tone: 'neutral', text: '当前阶段未开放排行榜。' }
  }
  if (code === 'submission_closed') {
    return { tone: 'neutral', text: '当前阶段不允许提交 Flag。' }
  }
  if (code === 'runtime_closed') {
    return { tone: 'neutral', text: '当前阶段未开放动态实例。' }
  }
  if (code === 'registration_closed') {
    return { tone: 'neutral', text: '当前阶段未开放注册入口。' }
  }
  if (code === 'submission_rate_limited') {
    return { tone: 'danger', text: '提交过于频繁，请稍后再试。' }
  }
  if (code === 'login_rate_limited') {
    return { tone: 'danger', text: '登录尝试过于频繁，请稍后再试。' }
  }
  if (code === 'register_rate_limited') {
    return { tone: 'danger', text: '注册尝试过于频繁，请稍后再试。' }
  }
  if (code === 'instance_capacity_reached') {
    return { tone: 'danger', text: '当前题目实例配额已满，请稍后再试。' }
  }
  if (code === 'instance_cooldown_active') {
    return { tone: 'danger', text: '刚刚创建过实例，冷却中，请稍后再试。' }
  }
  if (code === 'instance_port_exhausted') {
    return { tone: 'danger', text: '当前实例端口资源已耗尽，请稍后再试或联系管理员。' }
  }
  if (code === 'instance_renew_limit_reached') {
    return { tone: 'danger', text: '实例已达到最大续期次数。' }
  }
  if (code === 'runtime_config_missing') {
    return { tone: 'danger', text: '运行配置不完整，无法拉起实例。' }
  }
  return { tone: 'danger', text: describeError(error, fallback) }
}
