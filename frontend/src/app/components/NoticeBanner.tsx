import React from 'react'

import type { Notice } from '../utils/errors'

export function NoticeBanner({ notice }: { notice: Notice | null }): React.JSX.Element | null {
  if (!notice) {
    return null
  }

  const tone = notice.tone === 'ok' ? 'ok' : notice.tone === 'danger' ? 'danger' : 'neutral'
  const toneClass = notice.tone === 'ok' ? 'notice-success' : notice.tone === 'danger' ? 'notice-danger' : ''
  return (
    <div aria-live="polite" className={`notice ds-notice ${toneClass}`.trim()} data-tone={tone} role="status">
      {notice.text}
    </div>
  )
}
