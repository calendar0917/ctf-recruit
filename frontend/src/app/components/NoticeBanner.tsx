import React from 'react'

import type { Notice } from '../utils/errors'

export function NoticeBanner({ notice }: { notice: Notice | null }): React.JSX.Element | null {
  if (!notice) {
    return null
  }
  const tone = notice.tone === 'ok' ? 'ok' : notice.tone === 'danger' ? 'danger' : 'neutral'
  return (
    <div aria-live="polite" className="ds-notice" data-tone={tone}>
      {notice.text}
    </div>
  )
}
