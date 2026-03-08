import React, { useMemo, useState } from 'react'
import ReactDOM from 'react-dom/client'
import './styles.css'

type View = 'hall' | 'challenges' | 'runtime' | 'scoreboard' | 'admin'
type CategoryKey = 'Web' | 'Misc' | 'Crypto'
type SectionKey = 'statement' | 'attachments' | 'submit'
type AdminSectionKey = 'challenges' | 'announcements' | 'submissions' | 'instances'

type NavItem = {
  id: View
  label: string
  compactLabel: string
  note: string
}

type Challenge = {
  id: number
  title: string
  category: CategoryKey
  points: number
  difficulty: 'Easy' | 'Normal' | 'Hard'
  dynamic: boolean
  solved: boolean
  summary: string
  objective: string
  attachments: string[]
}

type Announcement = {
  title: string
  time: string
  content: string
}

type SolvedChallenge = {
  title: string
  category: CategoryKey
  difficulty: Challenge['difficulty']
}

type Ranking = {
  rank: string
  name: string
  score: number
  solved: number
  solvedChallenges: SolvedChallenge[]
}

type AdminChallengeRecord = {
  id: number
  slug: string
  title: string
  category: CategoryKey
  points: number
  visible: boolean
  dynamic: boolean
  difficulty: Challenge['difficulty']
  updatedAt: string
}

type AdminAnnouncementRecord = {
  id: number
  title: string
  pinned: boolean
  published: boolean
  author: string
  updatedAt: string
}

type SubmissionRecord = {
  id: number
  challenge: string
  player: string
  status: 'Correct' | 'Wrong'
  submittedAt: string
  source: string
  submittedFlag: string
  resultMessage: string
}

type InstanceRecord = {
  id: number
  challenge: string
  player: string
  status: 'running' | 'creating' | 'terminated'
  expiresIn: string
  actionLabel: string
}

const navItems: NavItem[] = [
  { id: 'hall', label: '比赛大厅', compactLabel: '大厅', note: 'Overview' },
  { id: 'challenges', label: '题目面板', compactLabel: '题目', note: 'Challenges' },
  { id: 'runtime', label: '实例中心', compactLabel: '实例', note: 'Runtime' },
  { id: 'scoreboard', label: '排行榜', compactLabel: '排行', note: 'Board' },
  { id: 'admin', label: '管理台', compactLabel: '管理', note: 'Admin' },
]

const challenges: Challenge[] = [
  {
    id: 1,
    title: 'Welcome Panel',
    category: 'Web',
    points: 100,
    difficulty: 'Easy',
    dynamic: true,
    solved: true,
    summary: '入口题，适合选手首次体验实例拉起、访问链接和 Flag 提交流程。',
    objective: '阅读题面并访问独立实例，确认页面入口和实例回收链路均工作正常。',
    attachments: ['statement.pdf'],
  },
  {
    id: 2,
    title: 'Packet Etiquette',
    category: 'Misc',
    points: 150,
    difficulty: 'Normal',
    dynamic: false,
    solved: false,
    summary: '通过附件恢复一次完整的认证行为，强调观察与信息筛选能力。',
    objective: '从抓包中定位异常认证字段，并还原出最终提交内容。',
    attachments: ['traffic.pcapng', 'readme.txt'],
  },
  {
    id: 3,
    title: 'Cipher Note',
    category: 'Crypto',
    points: 200,
    difficulty: 'Hard',
    dynamic: false,
    solved: false,
    summary: '轻量古典密码题，适合作为招新赛中段拉开差距。',
    objective: '分析题面给出的替换规则，并恢复原始信息。',
    attachments: ['cipher.txt'],
  },
]

const announcements: Announcement[] = [
  {
    title: '比赛环境已开放',
    time: '09:00',
    content: '动态题实例已经开放分配，首次启动通常在数秒内完成。',
  },
  {
    title: '附件下载规范',
    time: '09:12',
    content: '抓包、样本与附件统一在题目详情页下载，不在公告区重复发布。',
  },
  {
    title: '计分规则说明',
    time: '09:30',
    content: '总分相同的情况下，按达到该分数的时间先后进行排序。',
  },
]

const ranking: Ranking[] = [
  {
    rank: '01',
    name: 'alice',
    score: 550,
    solved: 4,
    solvedChallenges: [
      { title: 'Welcome Panel', category: 'Web', difficulty: 'Easy' },
      { title: 'Packet Etiquette', category: 'Misc', difficulty: 'Normal' },
      { title: 'Cipher Note', category: 'Crypto', difficulty: 'Hard' },
      { title: 'Vault Echo', category: 'Web', difficulty: 'Normal' },
    ],
  },
  {
    rank: '02',
    name: 'miko',
    score: 450,
    solved: 3,
    solvedChallenges: [
      { title: 'Welcome Panel', category: 'Web', difficulty: 'Easy' },
      { title: 'Packet Etiquette', category: 'Misc', difficulty: 'Normal' },
      { title: 'Cipher Note', category: 'Crypto', difficulty: 'Hard' },
    ],
  },
  {
    rank: '03',
    name: 'raven',
    score: 400,
    solved: 3,
    solvedChallenges: [
      { title: 'Welcome Panel', category: 'Web', difficulty: 'Easy' },
      { title: 'Cipher Note', category: 'Crypto', difficulty: 'Hard' },
      { title: 'Signal Trace', category: 'Misc', difficulty: 'Normal' },
    ],
  },
  {
    rank: '04',
    name: 'lin',
    score: 250,
    solved: 2,
    solvedChallenges: [
      { title: 'Welcome Panel', category: 'Web', difficulty: 'Easy' },
      { title: 'Packet Etiquette', category: 'Misc', difficulty: 'Normal' },
    ],
  },
  {
    rank: '05',
    name: 'zhou',
    score: 150,
    solved: 1,
    solvedChallenges: [{ title: 'Welcome Panel', category: 'Web', difficulty: 'Easy' }],
  },
]

const adminChallenges: AdminChallengeRecord[] = [
  {
    id: 1,
    slug: 'web-welcome',
    title: 'Welcome Panel',
    category: 'Web',
    points: 100,
    visible: true,
    dynamic: true,
    difficulty: 'Easy',
    updatedAt: '今天 10:12',
  },
  {
    id: 2,
    slug: 'packet-etiquette',
    title: 'Packet Etiquette',
    category: 'Misc',
    points: 150,
    visible: true,
    dynamic: false,
    difficulty: 'Normal',
    updatedAt: '今天 09:42',
  },
  {
    id: 3,
    slug: 'cipher-note',
    title: 'Cipher Note',
    category: 'Crypto',
    points: 200,
    visible: false,
    dynamic: false,
    difficulty: 'Hard',
    updatedAt: '昨天 20:18',
  },
]

const adminAnnouncements: AdminAnnouncementRecord[] = [
  {
    id: 1,
    title: '比赛环境已开放',
    pinned: true,
    published: true,
    author: 'admin',
    updatedAt: '今天 09:00',
  },
  {
    id: 2,
    title: '附件下载规范',
    pinned: false,
    published: true,
    author: 'admin',
    updatedAt: '今天 09:12',
  },
  {
    id: 3,
    title: '第二阶段题目预告',
    pinned: false,
    published: false,
    author: 'ops',
    updatedAt: '草稿',
  },
]

const submissionRecords: SubmissionRecord[] = [
  {
    id: 3012,
    challenge: 'Welcome Panel',
    player: 'alice',
    status: 'Correct',
    submittedAt: '10:14:21',
    source: '127.0.0.1',
    submittedFlag: 'flag{welcome_runtime_ok}',
    resultMessage: 'flag accepted, awarded 100 points',
  },
  {
    id: 3011,
    challenge: 'Cipher Note',
    player: 'miko',
    status: 'Wrong',
    submittedAt: '10:13:40',
    source: '127.0.0.1',
    submittedFlag: 'flag{cipher_guess_v2}',
    resultMessage: 'flag rejected, challenge remains unsolved',
  },
  {
    id: 3010,
    challenge: 'Packet Etiquette',
    player: 'lin',
    status: 'Correct',
    submittedAt: '10:11:06',
    source: '127.0.0.1',
    submittedFlag: 'flag{packet_reassembled}',
    resultMessage: 'flag accepted, awarded 150 points',
  },
  {
    id: 3009,
    challenge: 'Welcome Panel',
    player: 'zhou',
    status: 'Wrong',
    submittedAt: '10:09:31',
    source: '127.0.0.1',
    submittedFlag: 'flag{hello_world}',
    resultMessage: 'flag rejected, no points awarded',
  },
]

const instanceRecords: InstanceRecord[] = [
  { id: 901, challenge: 'Welcome Panel', player: 'alice', status: 'running', expiresIn: '22m', actionLabel: '终止实例' },
  { id: 902, challenge: 'Packet Etiquette', player: 'lin', status: 'creating', expiresIn: '启动中', actionLabel: '查看日志' },
  { id: 903, challenge: 'Vault Echo', player: 'raven', status: 'terminated', expiresIn: '已结束', actionLabel: '查看记录' },
]

function App() {
  const [activeView, setActiveView] = useState<View>('challenges')
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [selectedChallengeId, setSelectedChallengeId] = useState<number>(1)
  const [flagValue, setFlagValue] = useState('')
  const [submitState, setSubmitState] = useState<'idle' | 'sent'>('idle')

  const selectedChallenge = useMemo(
    () => challenges.find((item) => item.id === selectedChallengeId) ?? challenges[0],
    [selectedChallengeId],
  )

  const solvedCount = useMemo(() => challenges.filter((item) => item.solved).length, [])

  const pageTitle = useMemo(() => {
    switch (activeView) {
      case 'hall':
        return '比赛大厅'
      case 'challenges':
        return '题目面板'
      case 'runtime':
        return '实例中心'
      case 'scoreboard':
        return '排行榜'
      case 'admin':
        return '管理台'
    }
  }, [activeView])

  const pageDescription = useMemo(() => {
    switch (activeView) {
      case 'hall':
        return '先看清比赛状态、公告和当前重点入口，再进入题目。'
      case 'challenges':
        return '做题页现在支持分类折叠、完成标记和题面区块折叠，但题面主线仍保持稳定。'
      case 'runtime':
        return '查看你当前的动态实例状态、访问入口和剩余时间。'
      case 'scoreboard':
        return '按总分和达到时间查看当前比赛排名，并可按方向悬浮查看各选手已解题目。'
      case 'admin':
        return '把题目、公告、提交与实例管理统一收进一个高密度但清晰的运营工作台。'
    }
  }, [activeView])

  function handleSelectChallenge(id: number) {
    setSelectedChallengeId(id)
    setFlagValue('')
    setSubmitState('idle')
  }

  function handleFlagChange(value: string) {
    setFlagValue(value)
    if (submitState === 'sent') {
      setSubmitState('idle')
    }
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!flagValue.trim()) return
    setSubmitState('sent')
  }

  return (
    <main className={sidebarCollapsed ? 'workspace-shell sidebar-collapsed' : 'workspace-shell'}>
      <aside className={sidebarCollapsed ? 'sidebar collapsed' : 'sidebar'}>
        <div className="sidebar-top">
          <div className="studio-mark">
            <div className="crest">御</div>
            <div className="studio-copy">
              <p className="studio-name">御林工作室</p>
              <h1>Yulin Contest Console</h1>
            </div>
          </div>

          <button
            aria-expanded={!sidebarCollapsed}
            aria-label={sidebarCollapsed ? '展开侧边栏' : '折叠侧边栏'}
            className={sidebarCollapsed ? 'sidebar-toggle collapsed' : 'sidebar-toggle'}
            onClick={() => setSidebarCollapsed((current) => !current)}
            type="button"
          >
            <span aria-hidden="true">{sidebarCollapsed ? '>>' : '<<'}</span>
          </button>
        </div>

        <nav className="sidebar-nav" aria-label="Workspace Navigation">
          {navItems.map((item) => {
            const isActive = item.id === activeView
            return (
              <button
                key={item.id}
                className={isActive ? 'nav-item active' : 'nav-item'}
                onClick={() => setActiveView(item.id)}
                title={item.label}
                type="button"
              >
                <span>{sidebarCollapsed ? item.compactLabel : item.label}</span>
                {!sidebarCollapsed && <small>{item.note}</small>}
              </button>
            )
          })}
        </nav>

        {!sidebarCollapsed && (
          <div className="sidebar-note">
            <p className="note-label">Studio Direction</p>
            <p>以“御林”意象建立秩序感和守卫感，但界面优先服务比赛操作，不用装饰压住功能。</p>
          </div>
        )}
      </aside>

      <section className="workspace-main">
        <header className="workspace-header">
          <div>
            <p className="header-kicker">Recruit 2025</p>
            <h2>{pageTitle}</h2>
            <p>{pageDescription}</p>
          </div>
          <div className="header-status">
            <div>
              <span>当前阶段</span>
              <strong>Round Open</strong>
            </div>
            <div>
              <span>登录身份</span>
              <strong>player / admin</strong>
            </div>
          </div>
        </header>

        {activeView === 'hall' && <HallView />}
        {activeView === 'challenges' && (
          <ChallengesView
            flagValue={flagValue}
            onFlagChange={handleFlagChange}
            onSelectChallenge={handleSelectChallenge}
            onSubmit={handleSubmit}
            selectedChallenge={selectedChallenge}
            selectedChallengeId={selectedChallengeId}
            solvedCount={solvedCount}
            submitState={submitState}
          />
        )}
        {activeView === 'runtime' && <RuntimeView selectedChallenge={selectedChallenge} />}
        {activeView === 'scoreboard' && <ScoreboardView />}
        {activeView === 'admin' && <AdminView />}
      </section>
    </main>
  )
}

function HallView() {
  return (
    <div className="view-stack">
      <section className="overview-grid">
        <article className="panel major-panel">
          <div className="panel-head">
            <div>
              <p className="section-kicker">Overview</p>
              <h3>比赛概览</h3>
            </div>
          </div>
          <div className="metric-grid">
            <div className="metric-card">
              <span>开放题目</span>
              <strong>12</strong>
            </div>
            <div className="metric-card">
              <span>动态题</span>
              <strong>3</strong>
            </div>
            <div className="metric-card">
              <span>在线实例</span>
              <strong>7</strong>
            </div>
            <div className="metric-card">
              <span>当前第一</span>
              <strong>alice</strong>
            </div>
          </div>
        </article>

        <article className="panel minor-panel">
          <div className="panel-head compact-head">
            <div>
              <p className="section-kicker">Quick Entry</p>
              <h3>快速入口</h3>
            </div>
          </div>
          <div className="entry-list">
            <button className="entry-card" type="button">
              <strong>进入题目面板</strong>
              <span>浏览题目、下载附件、提交 Flag</span>
            </button>
            <button className="entry-card" type="button">
              <strong>查看动态实例</strong>
              <span>启动、查看和销毁当前实例</span>
            </button>
          </div>
        </article>
      </section>

      <section className="two-column-layout">
        <article className="panel">
          <div className="panel-head compact-head">
            <div>
              <p className="section-kicker">Announcements</p>
              <h3>最新公告</h3>
            </div>
          </div>
          <div className="announcement-list">
            {announcements.map((item) => (
              <article className="announcement-card" key={item.title}>
                <div className="announcement-head">
                  <strong>{item.title}</strong>
                  <span>{item.time}</span>
                </div>
                <p>{item.content}</p>
              </article>
            ))}
          </div>
        </article>

        <article className="panel">
          <div className="panel-head compact-head">
            <div>
              <p className="section-kicker">Focus</p>
              <h3>今日重点题目</h3>
            </div>
          </div>
          <div className="focus-list">
            {challenges.slice(0, 2).map((item) => (
              <article className="focus-row" key={item.id}>
                <div>
                  <strong>{item.title}</strong>
                  <p>{item.summary}</p>
                </div>
                <span>{item.points} pts</span>
              </article>
            ))}
          </div>
        </article>
      </section>
    </div>
  )
}

function ChallengesView(props: {
  selectedChallengeId: number
  selectedChallenge: Challenge
  flagValue: string
  solvedCount: number
  submitState: 'idle' | 'sent'
  onSelectChallenge: (id: number) => void
  onFlagChange: (value: string) => void
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void
}) {
  const {
    selectedChallengeId,
    selectedChallenge,
    flagValue,
    solvedCount,
    submitState,
    onSelectChallenge,
    onFlagChange,
    onSubmit,
  } = props

  const [openCategories, setOpenCategories] = useState<Record<CategoryKey, boolean>>({
    Web: true,
    Misc: true,
    Crypto: true,
  })
  const [openSections, setOpenSections] = useState<Record<SectionKey, boolean>>({
    statement: true,
    attachments: true,
    submit: true,
  })

  const groupedChallenges = useMemo(() => {
    const categories: CategoryKey[] = ['Web', 'Misc', 'Crypto']
    return categories.map((category) => {
      const items = challenges.filter((challenge) => challenge.category === category)
      const solved = items.filter((challenge) => challenge.solved).length
      return { category, items, solved }
    })
  }, [])

  function toggleCategory(category: CategoryKey) {
    setOpenCategories((current) => ({ ...current, [category]: !current[category] }))
  }

  function toggleSection(section: SectionKey) {
    setOpenSections((current) => ({ ...current, [section]: !current[section] }))
  }

  return (
    <div className="challenge-workspace focused-workspace">
      <aside className="panel challenge-rail categorized-rail">
        <div className="panel-head compact-head rail-head">
          <div>
            <p className="section-kicker">Challenge List</p>
            <h3>题目</h3>
          </div>
          <div className="rail-progress">
            <span>已完成</span>
            <strong>
              {solvedCount}/{challenges.length}
            </strong>
          </div>
        </div>

        <div className="category-stack">
          {groupedChallenges.map((group) => {
            const isOpen = openCategories[group.category]
            const categoryListId = `category-${group.category.toLowerCase()}-list`
            return (
              <section className="category-group" key={group.category}>
                <button
                  aria-controls={categoryListId}
                  aria-expanded={isOpen}
                  className={isOpen ? 'category-toggle open' : 'category-toggle'}
                  onClick={() => toggleCategory(group.category)}
                  type="button"
                >
                  <div className="category-toggle-main">
                    <span>{group.category}</span>
                    <small>
                      {group.solved}/{group.items.length} 已完成
                    </small>
                  </div>
                  <div className="category-toggle-side">
                    <small>{group.items.length} 题</small>
                    <strong className={isOpen ? 'toggle-indicator open' : 'toggle-indicator'} aria-hidden="true" />
                  </div>
                </button>
                {isOpen && (
                  <div className="challenge-menu" id={categoryListId}>
                    {group.items.map((challenge) => {
                      const active = challenge.id === selectedChallengeId
                      const itemClassName = [
                        'challenge-menu-item',
                        `menu-item-${challenge.difficulty.toLowerCase()}`,
                        active ? 'active' : '',
                      ]
                        .filter(Boolean)
                        .join(' ')

                      return (
                        <button
                          aria-pressed={active}
                          className={itemClassName}
                          key={challenge.id}
                          onClick={() => onSelectChallenge(challenge.id)}
                          type="button"
                        >
                          <div className="challenge-menu-main">
                            <div className="challenge-menu-heading">
                              <strong>{challenge.title}</strong>
                              <span>{challenge.points} pts</span>
                            </div>
                            <div className="menu-status">
                              <small className="menu-difficulty-dot" aria-hidden="true" />
                              <small className={`difficulty-chip difficulty-${challenge.difficulty.toLowerCase()}`}>
                                {challenge.difficulty}
                              </small>
                              {challenge.dynamic && <small className="dynamic-chip">Dynamic</small>}
                              {challenge.solved && <small className="done-chip">已完成</small>}
                            </div>
                          </div>
                        </button>
                      )
                    })}
                  </div>
                )}
              </section>
            )
          })}
        </div>
      </aside>

      <section className="panel challenge-detail focused-detail">
        <div className="panel-head focused-head">
          <div>
            <p className="section-kicker">Statement</p>
            <h3>{selectedChallenge.title}</h3>
          </div>
          <div className="detail-badges">
            <span>{selectedChallenge.category}</span>
            <span>{selectedChallenge.points} pts</span>
            <span className={`difficulty-badge difficulty-${selectedChallenge.difficulty.toLowerCase()}`}>
              {selectedChallenge.difficulty}
            </span>
            {selectedChallenge.dynamic && <span className="dynamic-badge">Dynamic</span>}
            {selectedChallenge.solved && <span className="done-badge">Solved</span>}
          </div>
        </div>

        <div className="detail-status-bar">
          <span>当前状态</span>
          <strong>
            {selectedChallenge.solved
              ? '这道题已经完成，可以复盘或继续查看附件。'
              : '这道题尚未完成，建议先阅读题面再决定是否启动实例。'}
          </strong>
        </div>

        <section className={openSections.statement ? 'foldable-block open' : 'foldable-block'}>
          <button
            aria-controls="challenge-statement-panel"
            aria-expanded={openSections.statement}
            className="fold-toggle"
            onClick={() => toggleSection('statement')}
            type="button"
          >
            <div>
              <span className="subhead">题目说明</span>
              <strong>阅读题面</strong>
            </div>
            <span className={openSections.statement ? 'fold-toggle-state open' : 'fold-toggle-state'}>
              {openSections.statement ? '收起' : '展开'}
            </span>
          </button>
          {openSections.statement && (
            <div className="fold-content statement-block" id="challenge-statement-panel">
              <p>{selectedChallenge.summary}</p>
              <p>{selectedChallenge.objective}</p>
            </div>
          )}
        </section>

        <section className={openSections.attachments ? 'foldable-block elevated-block open' : 'foldable-block elevated-block'}>
          <button
            aria-controls="challenge-attachments-panel"
            aria-expanded={openSections.attachments}
            className="fold-toggle"
            onClick={() => toggleSection('attachments')}
            type="button"
          >
            <div>
              <span className="subhead">附件</span>
              <strong>下载材料</strong>
            </div>
            <span className={openSections.attachments ? 'fold-toggle-state open' : 'fold-toggle-state'}>
              {openSections.attachments ? '收起' : '展开'}
            </span>
          </button>
          {openSections.attachments && (
            <div className="fold-content vertical-list" id="challenge-attachments-panel">
              {selectedChallenge.attachments.map((file) => (
                <button className="attachment-item wide-item" key={file} type="button">
                  <span>{file}</span>
                  <strong>下载</strong>
                </button>
              ))}
            </div>
          )}
        </section>

        <section className={openSections.submit ? 'foldable-block prominent-form open' : 'foldable-block prominent-form'}>
          <button
            aria-controls="challenge-submit-panel"
            aria-expanded={openSections.submit}
            className="fold-toggle"
            onClick={() => toggleSection('submit')}
            type="button"
          >
            <div>
              <span className="subhead">提交 Flag</span>
              <strong>提交答案</strong>
            </div>
            <span className={openSections.submit ? 'fold-toggle-state open' : 'fold-toggle-state'}>
              {openSections.submit ? '收起' : '展开'}
            </span>
          </button>
          {openSections.submit && (
            <form className="fold-content flag-form" id="challenge-submit-panel" onSubmit={onSubmit}>
              <div className="flag-row">
                <input
                  id="flag-input"
                  onChange={(event) => onFlagChange(event.target.value)}
                  placeholder="flag{...}"
                  type="text"
                  value={flagValue}
                />
                <button className="primary-button slim" type="submit">
                  提交答案
                </button>
              </div>
              <p className={submitState === 'sent' ? 'submit-hint success-text' : 'submit-hint'}>
                {submitState === 'sent'
                  ? '演示界面已记录一次提交动作，后续可直接接真实提交 API。'
                  : '分类、难度和完成状态现在可扫读，同时支持分类与区块折叠。'}
              </p>
            </form>
          )}
        </section>
      </section>
    </div>
  )
}

function RuntimeView(props: { selectedChallenge: Challenge }) {
  return (
    <section className="runtime-layout">
      <article className="panel runtime-monitor">
        <div className="panel-head">
          <div>
            <p className="section-kicker">Runtime</p>
            <h3>我的实例</h3>
          </div>
          <span className="status-pill">Running</span>
        </div>
        <div className="runtime-shell">
          <div className="runtime-line">
            <span>题目</span>
            <strong>{props.selectedChallenge.title}</strong>
          </div>
          <div className="runtime-line">
            <span>访问地址</span>
            <strong>http://localhost:18081</strong>
          </div>
          <div className="runtime-line">
            <span>剩余时间</span>
            <strong>22m 14s</strong>
          </div>
          <div className="runtime-line">
            <span>实例状态</span>
            <strong>运行中，可继续访问与调试</strong>
          </div>
        </div>
      </article>

      <article className="panel runtime-log">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Lifecycle</p>
            <h3>实例记录</h3>
          </div>
        </div>
        <div className="timeline-list">
          <div className="timeline-item">
            <span>09:08</span>
            <p>实例已创建，端口映射为 `18081`。</p>
          </div>
          <div className="timeline-item">
            <span>09:09</span>
            <p>你已访问题目首页，实例当前保持可用。</p>
          </div>
          <div className="timeline-item">
            <span>09:38</span>
            <p>若超时未续期，平台将自动回收当前实例。</p>
          </div>
        </div>
      </article>
    </section>
  )
}

function ScoreboardView() {
  const categories: CategoryKey[] = ['Web', 'Misc', 'Crypto']

  return (
    <section className="scoreboard-layout">
      <article className="panel scoreboard-panel">
        <div className="panel-head">
          <div>
            <p className="section-kicker">Scoreboard</p>
            <h3>比赛排行</h3>
          </div>
        </div>
        <div className="table-head scoreboard-head">
          <span>排名</span>
          <span>用户</span>
          <span>解题数</span>
          <span>得分</span>
          <span>解题方向</span>
        </div>
        <div className="table-body scoreboard-body">
          {ranking.map((entry) => {
            const groupedSolved = categories
              .map((category) => ({
                category,
                items: entry.solvedChallenges.filter((item) => item.category === category),
              }))
              .filter((group) => group.items.length > 0)

            return (
              <div className="table-row scoreboard-row" key={entry.rank}>
                <span>{entry.rank}</span>
                <strong>{entry.name}</strong>
                <span>{entry.solved}</span>
                <span>{entry.score}</span>
                <div className="solved-groups">
                  {groupedSolved.map((group) => (
                    <button
                      aria-label={`${entry.name} 在 ${group.category} 方向解出 ${group.items.length} 题`}
                      className={`solved-category solved-category-${group.category.toLowerCase()}`}
                      key={group.category}
                      type="button"
                    >
                      <span className="solved-category-label">{group.category}</span>
                      <strong className="solved-category-count">{group.items.length}</strong>
                      <div className="solved-category-panel">
                        <p className="solved-category-title">{group.category} 已解题目</p>
                        <div className="solved-category-list">
                          {group.items.map((item) => (
                            <div className="solved-entry" key={item.title}>
                              <small className="solved-item">{item.title}</small>
                              <small className={`difficulty-chip difficulty-${item.difficulty.toLowerCase()}`}>
                                {item.difficulty}
                              </small>
                            </div>
                          ))}
                        </div>
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      </article>
    </section>
  )
}

function AdminView() {
  const [activeSection, setActiveSection] = useState<AdminSectionKey>('challenges')

  const adminSections: Array<{ id: AdminSectionKey; label: string; note: string }> = [
    { id: 'challenges', label: '题目管理', note: 'Create / Update' },
    { id: 'announcements', label: '公告管理', note: 'Publish / Pin' },
    { id: 'submissions', label: '提交记录', note: 'Review / Filter' },
    { id: 'instances', label: '实例处置', note: 'Observe / Terminate' },
  ]

  return (
    <div className="admin-layout admin-workspace">
      <section className="admin-topbar panel">
        <div>
          <p className="section-kicker">Operations</p>
          <h3>管理操作台</h3>
          <p className="admin-lead">先定题面与分值，再发公告、看提交、查实例，整个运营链路收在同一页里。</p>
        </div>
        <div className="admin-summary-strip">
          <div className="metric-card admin-metric">
            <span>题目总数</span>
            <strong>{adminChallenges.length}</strong>
          </div>
          <div className="metric-card admin-metric">
            <span>已发布公告</span>
            <strong>{adminAnnouncements.filter((item) => item.published).length}</strong>
          </div>
          <div className="metric-card admin-metric">
            <span>今日提交</span>
            <strong>{submissionRecords.length}</strong>
          </div>
          <div className="metric-card admin-metric">
            <span>运行实例</span>
            <strong>{instanceRecords.filter((item) => item.status === 'running').length}</strong>
          </div>
        </div>
      </section>

      <section className="admin-grid">
        <aside className="panel admin-sidebar-panel">
          <div className="panel-head compact-head admin-side-head">
            <div>
              <p className="section-kicker">Modules</p>
              <h3>操作模块</h3>
            </div>
          </div>
          <div className="admin-module-list">
            {adminSections.map((section) => {
              const active = section.id === activeSection
              return (
                <button
                  className={active ? 'admin-module-item active' : 'admin-module-item'}
                  key={section.id}
                  onClick={() => setActiveSection(section.id)}
                  type="button"
                >
                  <strong>{section.label}</strong>
                  <span>{section.note}</span>
                </button>
              )
            })}
          </div>
          <div className="admin-side-note">
            <p className="note-label">Workflow</p>
            <p>建议先维护题目与公告，再在提交记录和实例处置里做赛时响应。</p>
          </div>
        </aside>

        <section className="admin-main-panel">
          {activeSection === 'challenges' && <AdminChallengesSection />}
          {activeSection === 'announcements' && <AdminAnnouncementsSection />}
          {activeSection === 'submissions' && <AdminSubmissionsSection />}
          {activeSection === 'instances' && <AdminInstancesSection />}
        </section>
      </section>
    </div>
  )
}

function AdminChallengesSection() {
  return (
    <div className="admin-section-stack">
      <section className="panel admin-form-panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Challenge Editor</p>
            <h3>题目编辑</h3>
          </div>
          <button className="primary-button slim" type="button">
            新建题目
          </button>
        </div>

        <div className="admin-form-grid">
          <label className="form-field">
            <span>题目标题</span>
            <input defaultValue="Welcome Panel" type="text" />
          </label>
          <label className="form-field">
            <span>题目标识</span>
            <input defaultValue="web-welcome" type="text" />
          </label>
          <label className="form-field">
            <span>分类</span>
            <select defaultValue="Web">
              <option>Web</option>
              <option>Misc</option>
              <option>Crypto</option>
            </select>
          </label>
          <label className="form-field">
            <span>难度</span>
            <select defaultValue="Easy">
              <option>Easy</option>
              <option>Normal</option>
              <option>Hard</option>
            </select>
          </label>
          <label className="form-field">
            <span>分值</span>
            <input defaultValue="100" type="number" />
          </label>
          <label className="form-field toggle-field">
            <span>动态实例</span>
            <button className="ghost-button slim" type="button">
              已启用
            </button>
          </label>
          <label className="form-field full-span">
            <span>题目摘要</span>
            <textarea defaultValue="入口题，适合选手首次体验实例拉起、访问链接和 Flag 提交流程。" rows={4} />
          </label>
        </div>

        <div className="admin-detail-grid">
          <article className="admin-detail-card">
            <div className="admin-detail-head">
              <strong>附件管理</strong>
              <button className="ghost-button slim" type="button">
                上传附件
              </button>
            </div>
            <div className="detail-list">
              <div className="detail-list-row">
                <span>statement.pdf</span>
                <small>下载权限: public</small>
              </div>
              <div className="detail-list-row">
                <span>docker-compose.yml</span>
                <small>下载权限: admin</small>
              </div>
            </div>
          </article>

          <article className="admin-detail-card">
            <div className="admin-detail-head">
              <strong>Flag 与判题</strong>
              <button className="ghost-button slim" type="button">
                更新校验
              </button>
            </div>
            <div className="detail-list">
              <div className="detail-list-row stacked">
                <span>当前 Flag</span>
                <code>flag{'{welcome_runtime_ok}'}</code>
              </div>
              <div className="detail-list-row">
                <span>判题模式</span>
                <small>Exact Match</small>
              </div>
              <div className="detail-list-row">
                <span>重复提交</span>
                <small>允许记录 / 不重复计分</small>
              </div>
            </div>
          </article>

          <article className="admin-detail-card full-span-card">
            <div className="admin-detail-head">
              <strong>动态实例配置</strong>
              <button className="ghost-button slim" type="button">
                编辑运行配置
              </button>
            </div>
            <div className="runtime-config-grid">
              <div className="detail-list-row stacked">
                <span>镜像</span>
                <code>yulin/web-welcome:latest</code>
              </div>
              <div className="detail-list-row">
                <span>端口</span>
                <small>8080 -&gt; public http</small>
              </div>
              <div className="detail-list-row">
                <span>时限</span>
                <small>30 min / 可续期</small>
              </div>
              <div className="detail-list-row">
                <span>资源限制</span>
                <small>256MB / 0.5 CPU</small>
              </div>
            </div>
          </article>
        </div>

        <div className="admin-form-actions">
          <button className="ghost-button" type="button">
            保存草稿
          </button>
          <button className="primary-button" type="button">
            发布更新
          </button>
        </div>
      </section>

      <section className="panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Challenge List</p>
            <h3>题目列表</h3>
          </div>
        </div>
        <div className="admin-table-list">
          {adminChallenges.map((item) => (
            <article className="admin-table-row" key={item.id}>
              <div>
                <strong>{item.title}</strong>
                <p>{item.slug}</p>
              </div>
              <div className="admin-row-meta">
                <small>{item.category}</small>
                <small className={`difficulty-chip difficulty-${item.difficulty.toLowerCase()}`}>{item.difficulty}</small>
                <small>{item.points} pts</small>
                {item.dynamic && <small className="dynamic-chip">Dynamic</small>}
                {item.visible ? <small className="done-chip">Visible</small> : <small className="admin-muted-chip">Hidden</small>}
              </div>
              <div className="admin-row-actions">
                <span>{item.updatedAt}</span>
                <button className="ghost-button slim" type="button">
                  编辑
                </button>
              </div>
            </article>
          ))}
        </div>
      </section>
    </div>
  )
}

function AdminAnnouncementsSection() {
  return (
    <div className="admin-section-stack">
      <section className="panel admin-form-panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Announcement Composer</p>
            <h3>公告发布</h3>
          </div>
          <button className="primary-button slim" type="button">
            新建公告
          </button>
        </div>

        <div className="admin-form-grid">
          <label className="form-field full-span">
            <span>公告标题</span>
            <input defaultValue="比赛环境已开放" type="text" />
          </label>
          <label className="form-field toggle-field">
            <span>置顶</span>
            <button className="ghost-button slim" type="button">
              已置顶
            </button>
          </label>
          <label className="form-field toggle-field">
            <span>发布状态</span>
            <button className="ghost-button slim" type="button">
              立即发布
            </button>
          </label>
          <label className="form-field">
            <span>发布时间</span>
            <input defaultValue="2025-08-08 09:00" type="text" />
          </label>
          <label className="form-field">
            <span>可见范围</span>
            <select defaultValue="全部选手">
              <option>全部选手</option>
              <option>仅管理员</option>
              <option>指定分组</option>
            </select>
          </label>
          <label className="form-field full-span">
            <span>公告内容</span>
            <textarea defaultValue="动态题实例已经开放分配，首次启动通常在数秒内完成。" rows={5} />
          </label>
        </div>

        <div className="admin-detail-grid compact-detail-grid">
          <article className="admin-detail-card">
            <div className="admin-detail-head">
              <strong>投放策略</strong>
            </div>
            <div className="detail-list">
              <div className="detail-list-row">
                <span>顶部公告栏</span>
                <small>启用</small>
              </div>
              <div className="detail-list-row">
                <span>比赛大厅推送</span>
                <small>启用</small>
              </div>
              <div className="detail-list-row">
                <span>历史保留</span>
                <small>赛后可见</small>
              </div>
            </div>
          </article>

          <article className="admin-detail-card">
            <div className="admin-detail-head">
              <strong>草稿说明</strong>
            </div>
            <div className="detail-list">
              <div className="detail-list-row stacked">
                <span>当前草稿</span>
                <small>第二阶段题目预告，等待题面全部解锁后发布。</small>
              </div>
            </div>
          </article>
        </div>

        <div className="admin-form-actions">
          <button className="ghost-button" type="button">
            保存草稿
          </button>
          <button className="primary-button" type="button">
            发布公告
          </button>
        </div>
      </section>

      <section className="panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Announcement Queue</p>
            <h3>公告队列</h3>
          </div>
        </div>
        <div className="admin-split-list">
          <div className="admin-split-column">
            <p className="subhead">已发布</p>
            <div className="admin-table-list compact-table-list">
              {adminAnnouncements
                .filter((item) => item.published)
                .map((item) => (
                  <article className="admin-table-row compact-admin-row" key={item.id}>
                    <div>
                      <strong>{item.title}</strong>
                      <p>{item.author}</p>
                    </div>
                    <div className="admin-row-meta">
                      {item.pinned && <small className="dynamic-chip">Pinned</small>}
                      <small className="done-chip">Published</small>
                    </div>
                  </article>
                ))}
            </div>
          </div>

          <div className="admin-split-column">
            <p className="subhead">草稿箱</p>
            <div className="admin-table-list compact-table-list">
              {adminAnnouncements
                .filter((item) => !item.published)
                .map((item) => (
                  <article className="admin-table-row compact-admin-row" key={item.id}>
                    <div>
                      <strong>{item.title}</strong>
                      <p>{item.author}</p>
                    </div>
                    <div className="admin-row-meta">
                      <small className="admin-muted-chip">Draft</small>
                      <small>{item.updatedAt}</small>
                    </div>
                  </article>
                ))}
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}

function AdminSubmissionsSection() {
  const selectedSubmission = submissionRecords[1]

  return (
    <div className="admin-section-stack submissions-workspace">
      <section className="panel admin-filter-panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Submission Filters</p>
            <h3>提交筛选</h3>
          </div>
        </div>
        <div className="admin-filter-grid">
          <label className="form-field">
            <span>选手</span>
            <input placeholder="alice / miko" type="text" />
          </label>
          <label className="form-field">
            <span>题目</span>
            <input placeholder="Welcome Panel" type="text" />
          </label>
          <label className="form-field">
            <span>结果</span>
            <select defaultValue="全部">
              <option>全部</option>
              <option>Correct</option>
              <option>Wrong</option>
            </select>
          </label>
          <button className="primary-button slim align-end" type="button">
            应用筛选
          </button>
        </div>
      </section>

      <section className="admin-dual-pane">
        <article className="panel">
          <div className="panel-head compact-head">
            <div>
              <p className="section-kicker">Submission Stream</p>
              <h3>提交记录</h3>
            </div>
          </div>
          <div className="admin-table-list">
            {submissionRecords.map((item) => (
              <article className="admin-table-row" key={item.id}>
                <div>
                  <strong>{item.challenge}</strong>
                  <p>#{item.id}</p>
                </div>
                <div className="admin-row-meta">
                  <small>{item.player}</small>
                  {item.status === 'Correct' ? <small className="done-chip">Correct</small> : <small className="admin-warn-chip">Wrong</small>}
                  <small>{item.source}</small>
                </div>
                <div className="admin-row-actions">
                  <span>{item.submittedAt}</span>
                  <button className="ghost-button slim" type="button">
                    查看详情
                  </button>
                </div>
              </article>
            ))}
          </div>
        </article>

        <article className="panel admin-detail-panel">
          <div className="panel-head compact-head">
            <div>
              <p className="section-kicker">Submission Detail</p>
              <h3>提交详情</h3>
            </div>
            <small className={selectedSubmission.status === 'Correct' ? 'done-chip' : 'admin-warn-chip'}>
              {selectedSubmission.status}
            </small>
          </div>
          <div className="detail-list detail-stack-list">
            <div className="detail-list-row">
              <span>选手</span>
              <small>{selectedSubmission.player}</small>
            </div>
            <div className="detail-list-row">
              <span>题目</span>
              <small>{selectedSubmission.challenge}</small>
            </div>
            <div className="detail-list-row">
              <span>提交时间</span>
              <small>{selectedSubmission.submittedAt}</small>
            </div>
            <div className="detail-list-row stacked">
              <span>提交内容</span>
              <code>{selectedSubmission.submittedFlag}</code>
            </div>
            <div className="detail-list-row stacked">
              <span>判题结果</span>
              <small>{selectedSubmission.resultMessage}</small>
            </div>
          </div>
        </article>
      </section>
    </div>
  )
}

function AdminInstancesSection() {
  return (
    <div className="admin-section-stack">
      <section className="panel admin-filter-panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Runtime Actions</p>
            <h3>实例处置</h3>
          </div>
        </div>
        <div className="admin-filter-grid compact-actions">
          <button className="ghost-button slim" type="button">
            刷新实例列表
          </button>
          <button className="ghost-button slim" type="button">
            仅看运行中
          </button>
          <button className="ghost-button slim" type="button">
            导出记录
          </button>
        </div>
      </section>

      <section className="panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Instance List</p>
            <h3>实例列表</h3>
          </div>
        </div>
        <div className="admin-table-list">
          {instanceRecords.map((item) => (
            <article className="admin-table-row" key={item.id}>
              <div>
                <strong>{item.challenge}</strong>
                <p>实例 #{item.id}</p>
              </div>
              <div className="admin-row-meta">
                <small>{item.player}</small>
                <small className={`instance-chip instance-${item.status}`}>{item.status}</small>
                <small>{item.expiresIn}</small>
              </div>
              <div className="admin-row-actions">
                <span>{item.actionLabel}</span>
                <button className={item.status === 'running' ? 'primary-button slim' : 'ghost-button slim'} type="button">
                  {item.actionLabel}
                </button>
              </div>
            </article>
          ))}
        </div>
      </section>
    </div>
  )
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
