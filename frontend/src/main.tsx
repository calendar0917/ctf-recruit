import React from 'react'
import ReactDOM from 'react-dom/client'
import './styles.css'

function App() {
  const cards = [
    {
      title: '比赛入口',
      text: '选手端将从这里进入题目列表、实例控制、Flag 提交和排行榜。',
    },
    {
      title: '管理后台',
      text: '管理员在同一个工程中管理题目、公告、提交记录和动态实例。',
    },
    {
      title: '动态实例',
      text: '首期按用户独立实例设计，默认 30 分钟生命周期，由后端自动回收。',
    },
  ]

  return (
    <main className="page">
      <section className="hero">
        <p className="eyebrow">CTF Recruit 2025</p>
        <h1>Recruit Platform</h1>
        <p className="intro">
          单场招新赛平台基线已经建立，当前前端骨架用于承接比赛主页、题目页、实例面板和管理后台。
        </p>
      </section>
      <section className="grid">
        {cards.map((card) => (
          <article className="card" key={card.title}>
            <h2>{card.title}</h2>
            <p>{card.text}</p>
          </article>
        ))}
      </section>
    </main>
  )
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
