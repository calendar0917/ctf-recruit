import React from 'react'

export function Card(props: {
  title: string
  subtitle?: string
  actions?: React.ReactNode
  children: React.ReactNode
}): React.JSX.Element {
  return (
    <section className="ds-card">
      <header className="ds-card-header">
        <div style={{ display: 'flex', gap: 12, alignItems: 'baseline', justifyContent: 'space-between', flexWrap: 'wrap' }}>
          <div>
            <h2>{props.title}</h2>
            {props.subtitle ? <p>{props.subtitle}</p> : null}
          </div>
          {props.actions ? <div>{props.actions}</div> : null}
        </div>
      </header>
      <div className="ds-card-body">{props.children}</div>
    </section>
  )
}
