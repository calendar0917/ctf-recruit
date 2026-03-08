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

type AttachmentRecord = {
  name: string
  scope: 'public' | 'admin' | 'staff'
}

type AdminChallengeRecord = {
  id: number
  slug: string
  title: string
  category: CategoryKey
  points: number
  dynamic: boolean
  difficulty: Challenge['difficulty']
  releaseState: 'published' | 'draft' | 'hidden'
  runtimeHealth: 'healthy' | 'review' | 'offline'
  updatedAt: string
  owner: string
  updatedBy: string
  solveCount: number
  wrongCount: number
  summary: string
  attachments: AttachmentRecord[]
  flag: string
  judgeMode: string
  retryPolicy: string
  runtimeImage: string
  runtimePort: string
  runtimeTimeout: string
  runtimeLimit: string
  notes: string
}

type AdminAnnouncementRecord = {
  id: number
  title: string
  pinned: boolean
  status: 'published' | 'scheduled' | 'draft'
  author: string
  updatedAt: string
  updatedBy: string
  scope: string
  channel: string
  scheduledAt: string
  summary: string
  content: string
  surfaces: string[]
}

type SubmissionRecord = {
  id: number
  challengeId: number
  challenge: string
  player: string
  status: 'Correct' | 'Wrong'
  submittedAt: string
  source: string
  submittedFlag: string
  resultMessage: string
  reviewState: 'clear' | 'watch' | 'blocked'
  latency: string
  matchedPolicy: string
  note: string
}

type InstanceEvent = {
  time: string
  text: string
}

type InstanceRecord = {
  id: number
  challengeId: number
  challenge: string
  player: string
  status: 'running' | 'creating' | 'terminated'
  expiresIn: string
  expiresInMin: number
  actionLabel: string
  endpoint: string
  image: string
  region: string
  owner: string
  risk: 'stable' | 'expiring' | 'stuck'
  uptime: string
  lastEvent: string
  events: InstanceEvent[]
}

type OpsAlert = {
  id: string
  module: AdminSectionKey
  severity: 'notice' | 'watch' | 'critical'
  title: string
  detail: string
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
    dynamic: true,
    difficulty: 'Easy',
    releaseState: 'published',
    runtimeHealth: 'healthy',
    updatedAt: '今天 10:12',
    owner: 'yushu',
    updatedBy: 'admin',
    solveCount: 47,
    wrongCount: 16,
    summary: '入口题，适合选手首次体验实例拉起、访问链接和 Flag 提交流程。',
    attachments: [
      { name: 'statement.pdf', scope: 'public' },
      { name: 'docker-compose.yml', scope: 'admin' },
    ],
    flag: 'flag{welcome_runtime_ok}',
    judgeMode: 'Exact Match',
    retryPolicy: '允许重复提交，不重复计分',
    runtimeImage: 'yulin/web-welcome:latest',
    runtimePort: '8080 -> public http',
    runtimeTimeout: '30 min / 可续期',
    runtimeLimit: '256MB / 0.5 CPU',
    notes: '实例健康稳定，适合保留为新手入口；赛时应优先观察实例回收是否及时。',
  },
  {
    id: 2,
    slug: 'packet-etiquette',
    title: 'Packet Etiquette',
    category: 'Misc',
    points: 150,
    dynamic: false,
    difficulty: 'Normal',
    releaseState: 'published',
    runtimeHealth: 'healthy',
    updatedAt: '今天 09:42',
    owner: 'lin',
    updatedBy: 'ops',
    solveCount: 29,
    wrongCount: 33,
    summary: '通过附件恢复一次完整的认证行为，强调观察与信息筛选能力。',
    attachments: [
      { name: 'traffic.pcapng', scope: 'public' },
      { name: 'readme.txt', scope: 'public' },
    ],
    flag: 'flag{packet_reassembled}',
    judgeMode: 'Exact Match',
    retryPolicy: '允许重复提交，保留历史记录',
    runtimeImage: 'static-assets / none',
    runtimePort: '无动态端口',
    runtimeTimeout: '不适用',
    runtimeLimit: '不适用',
    notes: '错误提交偏多，建议在比赛中段复核题面提示是否足够明确。',
  },
  {
    id: 3,
    slug: 'cipher-note',
    title: 'Cipher Note',
    category: 'Crypto',
    points: 200,
    dynamic: false,
    difficulty: 'Hard',
    releaseState: 'hidden',
    runtimeHealth: 'review',
    updatedAt: '昨天 20:18',
    owner: 'miko',
    updatedBy: 'miko',
    solveCount: 0,
    wrongCount: 7,
    summary: '轻量古典密码题，适合作为招新赛中段拉开差距。',
    attachments: [{ name: 'cipher.txt', scope: 'public' }],
    flag: 'flag{columnar_note_restored}',
    judgeMode: 'Regex + normalized whitespace',
    retryPolicy: '允许重复提交，开启频率观察',
    runtimeImage: 'static-assets / none',
    runtimePort: '无动态端口',
    runtimeTimeout: '不适用',
    runtimeLimit: '不适用',
    notes: '当前保持隐藏，等题面复核完成后再开放；需留意选手是否已通过其它渠道获取附件。',
  },
]

const adminAnnouncements: AdminAnnouncementRecord[] = [
  {
    id: 1,
    title: '比赛环境已开放',
    pinned: true,
    status: 'published',
    author: 'admin',
    updatedAt: '今天 09:00',
    updatedBy: 'admin',
    scope: '全部选手',
    channel: '大厅横幅 + 公告栏',
    scheduledAt: '已发布',
    summary: '告知所有选手动态环境已就绪，可立即开始实例分配。',
    content: '动态题实例已经开放分配，首次启动通常在数秒内完成。若实例长时间未就绪，请先查看实例中心状态。',
    surfaces: ['顶部公告栏', '比赛大厅', '实例中心提示'],
  },
  {
    id: 2,
    title: '附件下载规范',
    pinned: false,
    status: 'published',
    author: 'admin',
    updatedAt: '今天 09:12',
    updatedBy: 'ops',
    scope: '全部选手',
    channel: '公告栏',
    scheduledAt: '已发布',
    summary: '统一附件入口，降低选手在大厅与题面之间来回寻找的成本。',
    content: '抓包、样本与附件统一在题目详情页下载，不在公告区重复发布；如附件更新，将在题面和管理台同步标记版本。',
    surfaces: ['公告栏', '题目详情页'],
  },
  {
    id: 3,
    title: '排行榜冻结提醒',
    pinned: false,
    status: 'scheduled',
    author: 'ops',
    updatedAt: '今天 11:20',
    updatedBy: 'ops',
    scope: '全部选手',
    channel: '大厅横幅',
    scheduledAt: '12:30 自动发布',
    summary: '在比赛尾段提示排名冻结，减少争议。',
    content: '比赛最后 30 分钟将进入排行榜冻结阶段，提交仍正常计分，最终结果在闭赛后统一揭晓。',
    surfaces: ['顶部横幅', '排行榜顶部'],
  },
  {
    id: 4,
    title: '第二阶段题目预告',
    pinned: false,
    status: 'draft',
    author: 'ops',
    updatedAt: '草稿',
    updatedBy: 'yushu',
    scope: '指定分组',
    channel: '大厅横幅',
    scheduledAt: '待定',
    summary: '等待题面全部解锁后再放出预告，避免提前泄露。',
    content: '第二阶段将开放更高分值题目与一条额外的动态链路，请留意公告更新时间。',
    surfaces: ['大厅横幅'],
  },
]

const submissionRecords: SubmissionRecord[] = [
  {
    id: 3012,
    challengeId: 1,
    challenge: 'Welcome Panel',
    player: 'alice',
    status: 'Correct',
    submittedAt: '10:14:21',
    source: '127.0.0.1',
    submittedFlag: 'flag{welcome_runtime_ok}',
    resultMessage: 'flag accepted, awarded 100 points',
    reviewState: 'clear',
    latency: '42 ms',
    matchedPolicy: 'exact-match',
    note: '命中标准判题链路，无需额外介入。',
  },
  {
    id: 3011,
    challengeId: 3,
    challenge: 'Cipher Note',
    player: 'miko',
    status: 'Wrong',
    submittedAt: '10:13:40',
    source: '127.0.0.1',
    submittedFlag: 'flag{cipher_guess_v2}',
    resultMessage: 'flag rejected, challenge remains unsolved',
    reviewState: 'watch',
    latency: '37 ms',
    matchedPolicy: 'normalized-regex',
    note: '同一选手短时间内连续第 4 次猜测，建议关注是否需要追加提示。',
  },
  {
    id: 3010,
    challengeId: 2,
    challenge: 'Packet Etiquette',
    player: 'lin',
    status: 'Correct',
    submittedAt: '10:11:06',
    source: '127.0.0.1',
    submittedFlag: 'flag{packet_reassembled}',
    resultMessage: 'flag accepted, awarded 150 points',
    reviewState: 'clear',
    latency: '49 ms',
    matchedPolicy: 'exact-match',
    note: '附件路径完整，判题结果稳定。',
  },
  {
    id: 3009,
    challengeId: 1,
    challenge: 'Welcome Panel',
    player: 'zhou',
    status: 'Wrong',
    submittedAt: '10:09:31',
    source: '127.0.0.1',
    submittedFlag: 'flag{hello_world}',
    resultMessage: 'flag rejected, no points awarded',
    reviewState: 'clear',
    latency: '35 ms',
    matchedPolicy: 'exact-match',
    note: '常见错误格式，暂无异常。',
  },
  {
    id: 3008,
    challengeId: 3,
    challenge: 'Cipher Note',
    player: 'raven',
    status: 'Wrong',
    submittedAt: '10:07:18',
    source: '10.0.0.24',
    submittedFlag: 'flag{column_shift_maybe}',
    resultMessage: 'flag rejected, frequency threshold exceeded',
    reviewState: 'blocked',
    latency: '18 ms',
    matchedPolicy: 'normalized-regex',
    note: '同源快速错误提交过多，已建议暂时拦截并人工确认。',
  },
]

const instanceRecords: InstanceRecord[] = [
  {
    id: 901,
    challengeId: 1,
    challenge: 'Welcome Panel',
    player: 'alice',
    status: 'running',
    expiresIn: '22m',
    expiresInMin: 22,
    actionLabel: '终止实例',
    endpoint: 'http://localhost:18081',
    image: 'yulin/web-welcome:latest',
    region: 'node-a / sandbox-01',
    owner: 'runtime-bot',
    risk: 'stable',
    uptime: '07m 14s',
    lastEvent: '14 秒前收到健康检查',
    events: [
      { time: '10:08', text: '容器创建完成，开始暴露访问入口。' },
      { time: '10:09', text: '选手首次访问首页，实例状态稳定。' },
      { time: '10:14', text: '最近一次健康检查通过。' },
    ],
  },
  {
    id: 902,
    challengeId: 2,
    challenge: 'Packet Etiquette',
    player: 'lin',
    status: 'creating',
    expiresIn: '启动中',
    expiresInMin: 0,
    actionLabel: '查看日志',
    endpoint: 'pending://runtime',
    image: 'static-assets / none',
    region: 'node-b / queue-03',
    owner: 'runtime-bot',
    risk: 'stuck',
    uptime: '00m 41s',
    lastEvent: '镜像准备完成，等待资源锁释放',
    events: [
      { time: '10:10', text: '实例请求入队，等待资源调度。' },
      { time: '10:10', text: '静态资源挂载成功，尚未生成访问链接。' },
      { time: '10:11', text: '当前处于启动等待，建议关注资源池。' },
    ],
  },
  {
    id: 903,
    challengeId: 99,
    challenge: 'Vault Echo',
    player: 'raven',
    status: 'terminated',
    expiresIn: '已结束',
    expiresInMin: 0,
    actionLabel: '查看记录',
    endpoint: 'terminated://vault-echo',
    image: 'yulin/vault-echo:2025.08',
    region: 'node-c / archive',
    owner: 'ops',
    risk: 'stable',
    uptime: '31m 09s',
    lastEvent: '实例已被管理员回收',
    events: [
      { time: '09:21', text: '实例启动并开始对外服务。' },
      { time: '09:52', text: '达到回收条件，开始清理资源。' },
      { time: '09:53', text: '实例终止完成，保留日志记录。' },
    ],
  },
  {
    id: 904,
    challengeId: 1,
    challenge: 'Welcome Panel',
    player: 'zhou',
    status: 'running',
    expiresIn: '06m',
    expiresInMin: 6,
    actionLabel: '延长 10 分钟',
    endpoint: 'http://localhost:18084',
    image: 'yulin/web-welcome:latest',
    region: 'node-a / sandbox-07',
    owner: 'runtime-bot',
    risk: 'expiring',
    uptime: '23m 06s',
    lastEvent: '实例接近过期阈值，等待续期或回收',
    events: [
      { time: '09:51', text: '实例创建完成，分配独立入口。' },
      { time: '10:03', text: '选手提交一次错误 Flag，实例继续保留。' },
      { time: '10:15', text: '剩余时间不足 10 分钟，进入提醒窗口。' },
    ],
  },
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

  const activeNavItem = useMemo(() => navItems.find((item) => item.id === activeView) ?? navItems[0], [activeView])

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
        <header className="workspace-header workspace-status-bar">
          <div>
            <p className="header-kicker">Recruit 2025</p>
            <p className="workspace-route-note">当前工作区 / {activeNavItem.label}</p>
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
            <div>
              <span>视图</span>
              <strong>{activeNavItem.note}</strong>
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

function getModuleLabel(section: AdminSectionKey) {
  switch (section) {
    case 'challenges':
      return '题目管理'
    case 'announcements':
      return '公告管理'
    case 'submissions':
      return '提交记录'
    case 'instances':
      return '实例处置'
  }
}

function getAlertSeverityMeta(severity: OpsAlert['severity']) {
  switch (severity) {
    case 'notice':
      return { label: 'Notice', className: 'admin-info-chip' }
    case 'watch':
      return { label: 'Watch', className: 'admin-warn-chip' }
    case 'critical':
      return { label: 'Critical', className: 'admin-critical-chip' }
  }
}

function getChallengeReleaseMeta(releaseState: AdminChallengeRecord['releaseState']) {
  switch (releaseState) {
    case 'published':
      return { label: 'Published', className: 'done-chip' }
    case 'draft':
      return { label: 'Draft', className: 'admin-info-chip' }
    case 'hidden':
      return { label: 'Hidden', className: 'admin-muted-chip' }
  }
}

function getRuntimeHealthMeta(runtimeHealth: AdminChallengeRecord['runtimeHealth']) {
  switch (runtimeHealth) {
    case 'healthy':
      return { label: 'Healthy', className: 'done-chip' }
    case 'review':
      return { label: 'Review', className: 'admin-warn-chip' }
    case 'offline':
      return { label: 'Offline', className: 'admin-critical-chip' }
  }
}

function getAnnouncementStatusMeta(status: AdminAnnouncementRecord['status']) {
  switch (status) {
    case 'published':
      return { label: 'Published', className: 'done-chip' }
    case 'scheduled':
      return { label: 'Scheduled', className: 'admin-info-chip' }
    case 'draft':
      return { label: 'Draft', className: 'admin-muted-chip' }
  }
}

function getSubmissionReviewMeta(reviewState: SubmissionRecord['reviewState']) {
  switch (reviewState) {
    case 'clear':
      return { label: 'Clear', className: 'done-chip' }
    case 'watch':
      return { label: 'Watch', className: 'admin-warn-chip' }
    case 'blocked':
      return { label: 'Blocked', className: 'admin-critical-chip' }
  }
}

function getSubmissionStatusMeta(status: SubmissionRecord['status']) {
  return status === 'Correct'
    ? { label: 'Correct', className: 'done-chip' }
    : { label: 'Wrong', className: 'admin-warn-chip' }
}

function getInstanceRiskMeta(risk: InstanceRecord['risk']) {
  switch (risk) {
    case 'stable':
      return { label: 'Stable', className: 'done-chip' }
    case 'expiring':
      return { label: 'Expiring', className: 'admin-warn-chip' }
    case 'stuck':
      return { label: 'Stuck', className: 'admin-critical-chip' }
  }
}

function getAttachmentScopeMeta(scope: AttachmentRecord['scope']) {
  switch (scope) {
    case 'public':
      return { label: 'public', className: 'done-chip' }
    case 'admin':
      return { label: 'admin', className: 'admin-muted-chip' }
    case 'staff':
      return { label: 'staff', className: 'admin-info-chip' }
  }
}

function AdminView() {
  const [activeSection, setActiveSection] = useState<AdminSectionKey>('challenges')
  const [selectedAdminChallengeId, setSelectedAdminChallengeId] = useState<number>(adminChallenges[0].id)
  const [selectedAnnouncementId, setSelectedAnnouncementId] = useState<number>(adminAnnouncements[0].id)
  const [selectedSubmissionId, setSelectedSubmissionId] = useState<number>(submissionRecords[0].id)
  const [selectedInstanceId, setSelectedInstanceId] = useState<number>(instanceRecords[0].id)

  const pendingCounts = useMemo(
    () => ({
      challenges: adminChallenges.filter((item) => item.releaseState !== 'published' || item.runtimeHealth !== 'healthy').length,
      announcements: adminAnnouncements.filter((item) => item.status !== 'published').length,
      submissions: submissionRecords.filter((item) => item.reviewState !== 'clear').length,
      instances: instanceRecords.filter((item) => item.risk !== 'stable').length,
    }),
    [],
  )

  const adminSections: Array<{ id: AdminSectionKey; label: string; note: string; pending: number }> = [
    { id: 'challenges', label: '题目管理', note: '题面、Flag 与运行配置', pending: pendingCounts.challenges },
    { id: 'announcements', label: '公告管理', note: '发布、排程与投放面', pending: pendingCounts.announcements },
    { id: 'submissions', label: '提交记录', note: '筛选、复核与异常响应', pending: pendingCounts.submissions },
    { id: 'instances', label: '实例处置', note: '运行状态、续期与回收', pending: pendingCounts.instances },
  ]

  const opsAlerts: OpsAlert[] = [
    {
      id: 'challenge-hidden-review',
      module: 'challenges',
      severity: 'watch',
      title: 'Cipher Note 仍处于隐藏复核',
      detail: '已有错误提交出现，建议在正式开放前再次确认题面与提示链路。',
    },
    {
      id: 'announcement-freeze-schedule',
      module: 'announcements',
      severity: 'notice',
      title: '排行榜冻结公告已排程',
      detail: '12:30 自动发布，投放到顶部横幅和排行榜顶部。',
    },
    {
      id: 'submission-frequency-block',
      module: 'submissions',
      severity: 'critical',
      title: '1 条提交触发频率阈值',
      detail: '同源快速错误提交建议人工介入，避免误伤正常选手。',
    },
    {
      id: 'instance-runtime-risk',
      module: 'instances',
      severity: 'watch',
      title: '2 个实例需要持续关注',
      detail: '其中 1 个即将过期，另 1 个仍停在启动等待状态。',
    },
  ]

  const activeSectionMeta = adminSections.find((section) => section.id === activeSection) ?? adminSections[0]
  const activeAlerts = opsAlerts.filter((item) => item.module === activeSection)
  const totalPending = adminSections.reduce((sum, section) => sum + section.pending, 0)

  return (
    <div className="admin-layout admin-workspace">
      <section className="admin-topbar panel admin-control-deck">
        <div className="admin-deck-copy">
          <p className="section-kicker">Admin Workbench</p>
          <h3>{activeSectionMeta.label}</h3>
          <p className="admin-lead">左侧是模块目录，当前区域只保留你正在处理的事务、优先级和少量核心指标。</p>
        </div>

        <div className="admin-deck-overview">
          <div className="admin-focus-ribbon admin-focus-ribbon-tight">
            <span>当前处理流</span>
            <strong>{activeSectionMeta.note}</strong>
            <small>{activeSectionMeta.pending} 项待处理</small>
          </div>

          <div className="admin-priority-list admin-priority-list-tight">
            {activeAlerts.length > 0 ? (
              activeAlerts.map((alert) => {
                const severityMeta = getAlertSeverityMeta(alert.severity)
                return (
                  <article className={`admin-priority-item ${alert.severity}`} key={alert.id}>
                    <div className="admin-related-meta">
                      <small className={severityMeta.className}>{severityMeta.label}</small>
                      <strong>{alert.title}</strong>
                    </div>
                    <span>{alert.detail}</span>
                  </article>
                )
              })
            ) : (
              <article className="admin-priority-item notice">
                <div className="admin-related-meta">
                  <small className="done-chip">Clear</small>
                  <strong>当前模块没有高优先级异常</strong>
                </div>
                <span>可以继续按常规流程维护，不需要额外切换观察视角。</span>
              </article>
            )}
          </div>
        </div>

        <div className="admin-summary-strip admin-summary-strip-compact">
          <div className="metric-card admin-metric compact-metric">
            <span>已发布题目</span>
            <strong>{adminChallenges.filter((item) => item.releaseState === 'published').length}</strong>
          </div>
          <div className="metric-card admin-metric compact-metric">
            <span>待发布公告</span>
            <strong>{adminAnnouncements.filter((item) => item.status !== 'published').length}</strong>
          </div>
          <div className="metric-card admin-metric compact-metric">
            <span>异常提交</span>
            <strong>{submissionRecords.filter((item) => item.reviewState !== 'clear').length}</strong>
          </div>
          <div className="metric-card admin-metric compact-metric">
            <span>总待处理</span>
            <strong>{totalPending}</strong>
          </div>
        </div>
      </section>

      <section className="admin-grid admin-directory-layout">
        <aside className="panel admin-sidebar-panel admin-directory-panel">
          <div className="admin-directory-head">
            <p className="section-kicker">Directory</p>
            <h3>模块目录</h3>
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
                  <div className="admin-module-meta">
                    <strong>{section.label}</strong>
                    <small className="admin-module-count">{section.pending}</small>
                  </div>
                  <span>{section.note}</span>
                </button>
              )
            })}
          </div>
          <div className="admin-directory-note">
            <div className="detail-list-row">
              <span>当前目录</span>
              <small>{activeSectionMeta.label}</small>
            </div>
            <div className="detail-list-row">
              <span>活跃告警</span>
              <small>{activeAlerts.length} 条</small>
            </div>
          </div>
        </aside>

        <section className="admin-main-panel">
          {activeSection === 'challenges' && (
            <AdminChallengesSection selectedChallengeId={selectedAdminChallengeId} onSelectChallenge={setSelectedAdminChallengeId} />
          )}
          {activeSection === 'announcements' && (
            <AdminAnnouncementsSection
              selectedAnnouncementId={selectedAnnouncementId}
              onSelectAnnouncement={setSelectedAnnouncementId}
            />
          )}
          {activeSection === 'submissions' && (
            <AdminSubmissionsSection selectedSubmissionId={selectedSubmissionId} onSelectSubmission={setSelectedSubmissionId} />
          )}
          {activeSection === 'instances' && (
            <AdminInstancesSection selectedInstanceId={selectedInstanceId} onSelectInstance={setSelectedInstanceId} />
          )}
        </section>
      </section>
    </div>
  )
}
function AdminChallengesSection(props: { selectedChallengeId: number; onSelectChallenge: (id: number) => void }) {
  const [searchValue, setSearchValue] = useState('')
  const [releaseFilter, setReleaseFilter] = useState<'all' | AdminChallengeRecord['releaseState']>('all')
  const [healthFilter, setHealthFilter] = useState<'all' | AdminChallengeRecord['runtimeHealth']>('all')
  const [openCategories, setOpenCategories] = useState<Record<CategoryKey, boolean>>({
    Web: true,
    Misc: true,
    Crypto: true,
  })

  const filteredChallenges = useMemo(
    () =>
      adminChallenges.filter((item) => {
        const keyword = searchValue.trim().toLowerCase()
        const matchesKeyword =
          keyword.length === 0 ||
          [item.title, item.slug, item.owner, item.updatedBy].some((part) => part.toLowerCase().includes(keyword))
        const matchesRelease = releaseFilter === 'all' || item.releaseState === releaseFilter
        const matchesHealth = healthFilter === 'all' || item.runtimeHealth === healthFilter
        return matchesKeyword && matchesRelease && matchesHealth
      }),
    [healthFilter, releaseFilter, searchValue],
  )

  const groupedChallenges = useMemo(() => {
    const categories: CategoryKey[] = ['Web', 'Misc', 'Crypto']
    return categories
      .map((category) => {
        const items = filteredChallenges.filter((item) => item.category === category)
        return {
          category,
          items,
          published: items.filter((item) => item.releaseState === 'published').length,
          attention: items.filter((item) => item.releaseState !== 'published' || item.runtimeHealth !== 'healthy').length,
        }
      })
      .filter((group) => group.items.length > 0)
  }, [filteredChallenges])

  const selectedChallenge =
    filteredChallenges.find((item) => item.id === props.selectedChallengeId) ??
    filteredChallenges[0] ??
    adminChallenges.find((item) => item.id === props.selectedChallengeId) ??
    adminChallenges[0]

  const relatedSubmissions = submissionRecords.filter((item) => item.challengeId === selectedChallenge.id)
  const relatedInstances = instanceRecords.filter((item) => item.challengeId === selectedChallenge.id)
  const totalAttempts = selectedChallenge.solveCount + selectedChallenge.wrongCount
  const solveRate = totalAttempts > 0 ? Math.round((selectedChallenge.solveCount / totalAttempts) * 100) : 0
  const releaseMeta = getChallengeReleaseMeta(selectedChallenge.releaseState)
  const runtimeMeta = getRuntimeHealthMeta(selectedChallenge.runtimeHealth)

  function toggleCategory(category: CategoryKey) {
    setOpenCategories((current) => ({ ...current, [category]: !current[category] }))
  }

  return (
    <div className="admin-section-stack">
      <section className="panel admin-filter-panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Challenge Operations</p>
            <h3>题目管理</h3>
          </div>
          <button className="primary-button slim" type="button">
            新建题目
          </button>
        </div>

        <div className="admin-filter-stack">
          <div className="admin-filter-grid">
            <label className="form-field">
              <span>搜索题目</span>
              <input
                onChange={(event) => setSearchValue(event.target.value)}
                placeholder="标题 / slug / 维护人"
                type="text"
                value={searchValue}
              />
            </label>
            <label className="form-field">
              <span>发布状态</span>
              <select
                onChange={(event) => setReleaseFilter(event.target.value as 'all' | AdminChallengeRecord['releaseState'])}
                value={releaseFilter}
              >
                <option value="all">全部</option>
                <option value="published">Published</option>
                <option value="draft">Draft</option>
                <option value="hidden">Hidden</option>
              </select>
            </label>
            <label className="form-field">
              <span>运行健康</span>
              <select
                onChange={(event) => setHealthFilter(event.target.value as 'all' | AdminChallengeRecord['runtimeHealth'])}
                value={healthFilter}
              >
                <option value="all">全部</option>
                <option value="healthy">Healthy</option>
                <option value="review">Review</option>
                <option value="offline">Offline</option>
              </select>
            </label>
            <article className="detail-list-row stacked admin-focus-card">
              <span>当前聚焦</span>
              <strong>{selectedChallenge.title}</strong>
              <small>
                {selectedChallenge.points} pts / {selectedChallenge.owner}
              </small>
            </article>
          </div>

          <div className="admin-mini-metrics">
            <div className="metric-card admin-mini-metric">
              <span>可见题目</span>
              <strong>{adminChallenges.filter((item) => item.releaseState === 'published').length}</strong>
            </div>
            <div className="metric-card admin-mini-metric">
              <span>隐藏 / 草稿</span>
              <strong>{adminChallenges.filter((item) => item.releaseState !== 'published').length}</strong>
            </div>
            <div className="metric-card admin-mini-metric">
              <span>动态链路</span>
              <strong>{adminChallenges.filter((item) => item.dynamic).length}</strong>
            </div>
            <div className="metric-card admin-mini-metric">
              <span>当前解出率</span>
              <strong>{solveRate}%</strong>
            </div>
          </div>
        </div>
      </section>

      <section className="admin-ops-layout admin-challenge-layout">
        <article className="panel admin-record-panel admin-category-panel">
          <div className="panel-head compact-head">
            <div>
              <p className="section-kicker">Challenge Queue</p>
              <h3>分类题目列表</h3>
            </div>
            <small>{filteredChallenges.length} 项</small>
          </div>
          <div className="admin-record-list admin-category-records">
            {groupedChallenges.length > 0 ? (
              groupedChallenges.map((group) => {
                const isOpen = openCategories[group.category]
                const categoryListId = `admin-category-${group.category.toLowerCase()}-list`
                return (
                  <section className="admin-category-group" key={group.category}>
                    <button
                      aria-controls={categoryListId}
                      aria-expanded={isOpen}
                      className={isOpen ? 'admin-category-toggle open' : 'admin-category-toggle'}
                      onClick={() => toggleCategory(group.category)}
                      type="button"
                    >
                      <div className="admin-category-main">
                        <strong>{group.category}</strong>
                        <small>{group.items.length} 题</small>
                      </div>
                      <div className="admin-category-side">
                        <small>{group.published} published</small>
                        {group.attention > 0 && <small className="admin-warn-chip">{group.attention} attention</small>}
                        <strong className={isOpen ? 'toggle-indicator open' : 'toggle-indicator'} aria-hidden="true" />
                      </div>
                    </button>

                    {isOpen && (
                      <div className="admin-category-list" id={categoryListId}>
                        {group.items.map((item) => {
                          const itemReleaseMeta = getChallengeReleaseMeta(item.releaseState)
                          const itemRuntimeMeta = getRuntimeHealthMeta(item.runtimeHealth)
                          const active = item.id === selectedChallenge.id
                          return (
                            <button
                              className={active ? 'admin-table-row admin-select-row active' : 'admin-table-row admin-select-row'}
                              key={item.id}
                              onClick={() => props.onSelectChallenge(item.id)}
                              type="button"
                            >
                              <div className="admin-select-head">
                                <strong>{item.title}</strong>
                                <small>{item.points} pts</small>
                              </div>
                              <p>{item.slug}</p>
                              <div className="admin-row-meta">
                                <small className={`difficulty-chip difficulty-${item.difficulty.toLowerCase()}`}>{item.difficulty}</small>
                                <small className={itemReleaseMeta.className}>{itemReleaseMeta.label}</small>
                                <small className={itemRuntimeMeta.className}>{itemRuntimeMeta.label}</small>
                                {item.dynamic && <small className="dynamic-chip">Dynamic</small>}
                              </div>
                              <div className="admin-row-actions">
                                <span>
                                  {item.owner} / {item.solveCount} solve / {item.wrongCount} wrong
                                </span>
                                <small>{item.updatedAt}</small>
                              </div>
                            </button>
                          )
                        })}
                      </div>
                    )}
                  </section>
                )
              })
            ) : (
              <div className="admin-empty-state">没有匹配当前筛选条件的题目。</div>
            )}
          </div>
        </article>

        <div className="admin-section-stack">
          <section className="panel admin-form-panel admin-challenge-workbench" key={`challenge-${selectedChallenge.id}`}>
            <div className="panel-head compact-head">
              <div>
                <p className="section-kicker">Challenge Editor</p>
                <h3>{selectedChallenge.title}</h3>
              </div>
              <div className="admin-row-meta">
                <small className={releaseMeta.className}>{releaseMeta.label}</small>
                <small className={runtimeMeta.className}>{runtimeMeta.label}</small>
                {selectedChallenge.dynamic && <small className="dynamic-chip">Dynamic</small>}
              </div>
            </div>

            <div className="admin-workbench-layout">
              <div className="admin-workbench-main">
                <div className="admin-workbench-status">
                  <div className="detail-list-row">
                    <span>维护责任</span>
                    <small>
                      {selectedChallenge.owner} / 最近由 {selectedChallenge.updatedBy} 更新
                    </small>
                  </div>
                  <div className="detail-list-row">
                    <span>提交反馈</span>
                    <small>
                      {selectedChallenge.solveCount} solve / {selectedChallenge.wrongCount} wrong / 解出率 {solveRate}%
                    </small>
                  </div>
                </div>

                <div className="admin-workbench-form">
                  <div className="admin-quick-edit-grid admin-quick-edit-grid-tight">
                    <label className="form-field full-span">
                      <span>题目标题</span>
                      <input defaultValue={selectedChallenge.title} type="text" />
                    </label>
                    <label className="form-field full-span">
                      <span>题目标识</span>
                      <input defaultValue={selectedChallenge.slug} type="text" />
                    </label>
                    <label className="form-field">
                      <span>分类</span>
                      <select defaultValue={selectedChallenge.category}>
                        <option>Web</option>
                        <option>Misc</option>
                        <option>Crypto</option>
                      </select>
                    </label>
                    <label className="form-field">
                      <span>难度</span>
                      <select defaultValue={selectedChallenge.difficulty}>
                        <option>Easy</option>
                        <option>Normal</option>
                        <option>Hard</option>
                      </select>
                    </label>
                    <label className="form-field">
                      <span>分值</span>
                      <input defaultValue={selectedChallenge.points} type="number" />
                    </label>
                    <label className="form-field">
                      <span>维护人</span>
                      <input defaultValue={selectedChallenge.owner} type="text" />
                    </label>
                    <label className="form-field full-span">
                      <span>题目摘要</span>
                      <textarea defaultValue={selectedChallenge.summary} rows={4} />
                    </label>
                  </div>
                </div>
              </div>

              <aside className="admin-workbench-side">
                <article className="admin-detail-card admin-judge-card">
                  <div className="admin-detail-head">
                    <strong>Flag 与判题</strong>
                    <button className="ghost-button slim" type="button">
                      更新校验
                    </button>
                  </div>
                  <div className="detail-list">
                    <div className="detail-list-row stacked code-safe-row">
                      <span>当前 Flag</span>
                      <code>{selectedChallenge.flag}</code>
                    </div>
                    <div className="detail-list-row">
                      <span>判题模式</span>
                      <small>{selectedChallenge.judgeMode}</small>
                    </div>
                    <div className="detail-list-row stacked">
                      <span>重试策略</span>
                      <small>{selectedChallenge.retryPolicy}</small>
                    </div>
                  </div>
                </article>

                <div className="admin-workbench-side-actions">
                  <button className="ghost-button" type="button">
                    保存草稿
                  </button>
                  <button className="primary-button" type="button">
                    发布更新
                  </button>
                </div>
              </aside>
            </div>
          </section>

          <section className="admin-detail-grid admin-challenge-detail-grid">
            <article className="admin-detail-card">
              <div className="admin-detail-head">
                <strong>附件管理</strong>
                <button className="ghost-button slim" type="button">
                  上传附件
                </button>
              </div>
              <div className="detail-list">
                {selectedChallenge.attachments.map((attachment) => {
                  const scopeMeta = getAttachmentScopeMeta(attachment.scope)
                  return (
                    <div className="detail-list-row" key={attachment.name}>
                      <span>{attachment.name}</span>
                      <small className={scopeMeta.className}>{scopeMeta.label}</small>
                    </div>
                  )
                })}
              </div>
            </article>

            <article className="admin-detail-card full-span-card">
              <div className="admin-detail-head">
                <strong>运行配置</strong>
                <button className="ghost-button slim" type="button">
                  编辑运行配置
                </button>
              </div>
              <div className="runtime-config-grid">
                <div className="detail-list-row stacked code-safe-row">
                  <span>镜像</span>
                  <code>{selectedChallenge.runtimeImage}</code>
                </div>
                <div className="detail-list-row">
                  <span>端口</span>
                  <small>{selectedChallenge.runtimePort}</small>
                </div>
                <div className="detail-list-row">
                  <span>时限</span>
                  <small>{selectedChallenge.runtimeTimeout}</small>
                </div>
                <div className="detail-list-row">
                  <span>资源限制</span>
                  <small>{selectedChallenge.runtimeLimit}</small>
                </div>
              </div>
              <div className="admin-note-block">{selectedChallenge.notes}</div>
            </article>
          </section>

          <section className="panel admin-associated-panel">
            <div className="panel-head compact-head">
              <div>
                <p className="section-kicker">Challenge Context</p>
                <h3>相关活动</h3>
              </div>
            </div>
            <div className="admin-associated-grid">
              <article className="admin-detail-card">
                <div className="admin-detail-head">
                  <strong>最近提交</strong>
                  <small>{relatedSubmissions.length} 条</small>
                </div>
                <div className="detail-list">
                  {relatedSubmissions.length > 0 ? (
                    relatedSubmissions.map((item) => {
                      const reviewMeta = getSubmissionReviewMeta(item.reviewState)
                      const statusMeta = getSubmissionStatusMeta(item.status)
                      return (
                        <div className="detail-list-row stacked" key={item.id}>
                          <div className="admin-related-meta">
                            <strong>#{item.id}</strong>
                            <small>{item.player}</small>
                            <small className={statusMeta.className}>{statusMeta.label}</small>
                            <small className={reviewMeta.className}>{reviewMeta.label}</small>
                          </div>
                          <small>{item.note}</small>
                        </div>
                      )
                    })
                  ) : (
                    <div className="admin-empty-state">当前题目还没有提交记录。</div>
                  )}
                </div>
              </article>

              <article className="admin-detail-card">
                <div className="admin-detail-head">
                  <strong>运行实例</strong>
                  <small>{relatedInstances.length} 个</small>
                </div>
                <div className="detail-list">
                  {relatedInstances.length > 0 ? (
                    relatedInstances.map((item) => {
                      const riskMeta = getInstanceRiskMeta(item.risk)
                      return (
                        <div className="detail-list-row stacked" key={item.id}>
                          <div className="admin-related-meta">
                            <strong>实例 #{item.id}</strong>
                            <small>{item.player}</small>
                            <small className={`instance-chip instance-${item.status}`}>{item.status}</small>
                            <small className={riskMeta.className}>{riskMeta.label}</small>
                          </div>
                          <small>
                            {item.endpoint} / {item.lastEvent}
                          </small>
                        </div>
                      )
                    })
                  ) : (
                    <div className="admin-empty-state">当前题目没有活跃实例。</div>
                  )}
                </div>
              </article>
            </div>
          </section>
        </div>
      </section>
    </div>
  )
}
function AdminAnnouncementsSection(props: {
  selectedAnnouncementId: number
  onSelectAnnouncement: (id: number) => void
}) {
  const [searchValue, setSearchValue] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | AdminAnnouncementRecord['status']>('all')

  const filteredAnnouncements = useMemo(
    () =>
      adminAnnouncements.filter((item) => {
        const keyword = searchValue.trim().toLowerCase()
        const matchesKeyword =
          keyword.length === 0 ||
          [item.title, item.author, item.updatedBy, item.scope, item.channel].some((part) => part.toLowerCase().includes(keyword))
        const matchesStatus = statusFilter === 'all' || item.status === statusFilter
        return matchesKeyword && matchesStatus
      }),
    [searchValue, statusFilter],
  )

  const selectedAnnouncement =
    filteredAnnouncements.find((item) => item.id === props.selectedAnnouncementId) ??
    filteredAnnouncements[0] ??
    adminAnnouncements.find((item) => item.id === props.selectedAnnouncementId) ??
    adminAnnouncements[0]

  const statusMeta = getAnnouncementStatusMeta(selectedAnnouncement.status)

  return (
    <div className="admin-section-stack">
      <section className="panel admin-filter-panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Announcement Operations</p>
            <h3>公告管理</h3>
          </div>
          <button className="primary-button slim" type="button">
            新建公告
          </button>
        </div>

        <div className="admin-filter-stack">
          <div className="admin-filter-grid admin-filter-grid-relaxed">
            <label className="form-field">
              <span>搜索公告</span>
              <input
                onChange={(event) => setSearchValue(event.target.value)}
                placeholder="标题 / 作者 / 范围"
                type="text"
                value={searchValue}
              />
            </label>
            <label className="form-field">
              <span>状态</span>
              <select
                onChange={(event) => setStatusFilter(event.target.value as 'all' | AdminAnnouncementRecord['status'])}
                value={statusFilter}
              >
                <option value="all">全部</option>
                <option value="published">Published</option>
                <option value="scheduled">Scheduled</option>
                <option value="draft">Draft</option>
              </select>
            </label>
            <article className="detail-list-row stacked admin-focus-card">
              <span>当前聚焦</span>
              <strong>{selectedAnnouncement.title}</strong>
              <small>{selectedAnnouncement.scope}</small>
            </article>
          </div>

          <div className="admin-mini-metrics admin-mini-metrics-relaxed">
            <div className="metric-card admin-mini-metric">
              <span>已发布</span>
              <strong>{adminAnnouncements.filter((item) => item.status === 'published').length}</strong>
            </div>
            <div className="metric-card admin-mini-metric">
              <span>排程中</span>
              <strong>{adminAnnouncements.filter((item) => item.status === 'scheduled').length}</strong>
            </div>
            <div className="metric-card admin-mini-metric">
              <span>草稿箱</span>
              <strong>{adminAnnouncements.filter((item) => item.status === 'draft').length}</strong>
            </div>
          </div>
        </div>
      </section>

      <section className="admin-ops-layout admin-ops-layout-relaxed">
        <article className="panel admin-record-panel admin-list-panel">
          <div className="panel-head compact-head">
            <div>
              <p className="section-kicker">Announcement Queue</p>
              <h3>公告队列</h3>
            </div>
            <small>{filteredAnnouncements.length} 项</small>
          </div>
          <div className="admin-record-list">
            {filteredAnnouncements.length > 0 ? (
              filteredAnnouncements.map((item) => {
                const itemStatusMeta = getAnnouncementStatusMeta(item.status)
                const active = item.id === selectedAnnouncement.id
                return (
                  <button
                    className={active ? 'admin-table-row admin-select-row admin-select-row-compact active' : 'admin-table-row admin-select-row admin-select-row-compact'}
                    key={item.id}
                    onClick={() => props.onSelectAnnouncement(item.id)}
                    type="button"
                  >
                    <div className="admin-select-head">
                      <strong>{item.title}</strong>
                      <small>{item.scheduledAt}</small>
                    </div>
                    <p>{item.author}</p>
                    <div className="admin-row-meta">
                      <small className={itemStatusMeta.className}>{itemStatusMeta.label}</small>
                      {item.pinned && <small className="dynamic-chip">Pinned</small>}
                      <small>{item.scope}</small>
                    </div>
                    <div className="admin-row-actions">
                      <span>{item.channel}</span>
                      <small>{item.updatedAt}</small>
                    </div>
                  </button>
                )
              })
            ) : (
              <div className="admin-empty-state">没有匹配当前筛选条件的公告。</div>
            )}
          </div>
        </article>

        <div className="admin-section-stack admin-editor-stack">
          <section className="panel admin-form-panel" key={`announcement-${selectedAnnouncement.id}`}>
            <div className="panel-head compact-head">
              <div>
                <p className="section-kicker">Announcement Composer</p>
                <h3>{selectedAnnouncement.title}</h3>
              </div>
              <div className="admin-row-meta">
                <small className={statusMeta.className}>{statusMeta.label}</small>
                {selectedAnnouncement.pinned && <small className="dynamic-chip">Pinned</small>}
              </div>
            </div>

            <div className="admin-selection-banner admin-selection-banner-relaxed">
              <div className="detail-list-row">
                <span>发布时间线</span>
                <small>{selectedAnnouncement.scheduledAt}</small>
              </div>
              <div className="detail-list-row">
                <span>最后编辑</span>
                <small>
                  {selectedAnnouncement.updatedBy} / {selectedAnnouncement.updatedAt}
                </small>
              </div>
            </div>

            <div className="admin-form-grid admin-form-grid-relaxed">
              <label className="form-field full-span">
                <span>公告标题</span>
                <input defaultValue={selectedAnnouncement.title} type="text" />
              </label>
              <label className="form-field">
                <span>可见范围</span>
                <input defaultValue={selectedAnnouncement.scope} type="text" />
              </label>
              <label className="form-field">
                <span>投放通道</span>
                <input defaultValue={selectedAnnouncement.channel} type="text" />
              </label>
              <label className="form-field toggle-field">
                <span>置顶</span>
                <button className="ghost-button slim" type="button">
                  {selectedAnnouncement.pinned ? '已置顶' : '普通公告'}
                </button>
              </label>
              <label className="form-field toggle-field">
                <span>状态</span>
                <button className="ghost-button slim" type="button">
                  {statusMeta.label}
                </button>
              </label>
              <label className="form-field full-span">
                <span>公告摘要</span>
                <textarea defaultValue={selectedAnnouncement.summary} rows={3} />
              </label>
              <label className="form-field full-span">
                <span>公告内容</span>
                <textarea defaultValue={selectedAnnouncement.content} rows={5} />
              </label>
            </div>

            <div className="admin-form-actions">
              <button className="ghost-button" type="button">
                保存草稿
              </button>
              <button className="primary-button" type="button">
                应用发布
              </button>
            </div>
          </section>

          <section className="admin-associated-grid admin-associated-grid-relaxed">
            <article className="panel admin-detail-panel">
              <div className="panel-head compact-head">
                <div>
                  <p className="section-kicker">Delivery Surface</p>
                  <h3>投放面</h3>
                </div>
              </div>
              <div className="admin-surface-list">
                {selectedAnnouncement.surfaces.map((surface) => (
                  <span className="admin-surface-chip" key={surface}>
                    {surface}
                  </span>
                ))}
              </div>
              <div className="admin-note-block">{selectedAnnouncement.summary}</div>
            </article>

            <article className="panel admin-detail-panel">
              <div className="panel-head compact-head">
                <div>
                  <p className="section-kicker">Publishing Notes</p>
                  <h3>发布备注</h3>
                </div>
              </div>
              <div className="detail-list detail-stack-list">
                <div className="detail-list-row">
                  <span>作者</span>
                  <small>{selectedAnnouncement.author}</small>
                </div>
                <div className="detail-list-row">
                  <span>编辑人</span>
                  <small>{selectedAnnouncement.updatedBy}</small>
                </div>
                <div className="detail-list-row stacked">
                  <span>排程</span>
                  <small>{selectedAnnouncement.scheduledAt}</small>
                </div>
                <div className="detail-list-row stacked">
                  <span>发布建议</span>
                  <small>
                    {selectedAnnouncement.status === 'draft'
                      ? '建议在题面全部确认后再进入排程，避免公告先于内容解锁。'
                      : '当前公告已具备明确的投放范围和发布时间线，可继续沿既定计划执行。'}
                  </small>
                </div>
              </div>
            </article>
          </section>
        </div>
      </section>
    </div>
  )
}

function AdminSubmissionsSection(props: { selectedSubmissionId: number; onSelectSubmission: (id: number) => void }) {
  const [searchValue, setSearchValue] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | SubmissionRecord['status']>('all')
  const [reviewFilter, setReviewFilter] = useState<'all' | SubmissionRecord['reviewState']>('all')

  const filteredSubmissions = useMemo(
    () =>
      submissionRecords.filter((item) => {
        const keyword = searchValue.trim().toLowerCase()
        const matchesKeyword =
          keyword.length === 0 ||
          [item.player, item.challenge, item.source, item.submittedFlag].some((part) => part.toLowerCase().includes(keyword))
        const matchesStatus = statusFilter === 'all' || item.status === statusFilter
        const matchesReview = reviewFilter === 'all' || item.reviewState === reviewFilter
        return matchesKeyword && matchesStatus && matchesReview
      }),
    [reviewFilter, searchValue, statusFilter],
  )

  const selectedSubmission =
    filteredSubmissions.find((item) => item.id === props.selectedSubmissionId) ??
    filteredSubmissions[0] ??
    submissionRecords.find((item) => item.id === props.selectedSubmissionId) ??
    submissionRecords[0]

  const reviewMeta = getSubmissionReviewMeta(selectedSubmission.reviewState)
  const statusMeta = getSubmissionStatusMeta(selectedSubmission.status)
  const challengeContext = adminChallenges.find((item) => item.id === selectedSubmission.challengeId)
  const playerHistory = submissionRecords.filter((item) => item.player === selectedSubmission.player)
  const relatedInstances = instanceRecords.filter(
    (item) => item.challengeId === selectedSubmission.challengeId && item.player === selectedSubmission.player,
  )

  return (
    <div className="admin-section-stack submissions-workspace">
      <section className="panel admin-filter-panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Submission Operations</p>
            <h3>提交记录</h3>
          </div>
        </div>

        <div className="admin-filter-stack">
          <div className="admin-filter-grid admin-filter-grid-relaxed">
            <label className="form-field">
              <span>搜索</span>
              <input
                onChange={(event) => setSearchValue(event.target.value)}
                placeholder="选手 / 题目 / 来源 / 提交内容"
                type="text"
                value={searchValue}
              />
            </label>
            <label className="form-field">
              <span>结果</span>
              <select
                onChange={(event) => setStatusFilter(event.target.value as 'all' | SubmissionRecord['status'])}
                value={statusFilter}
              >
                <option value="all">全部</option>
                <option value="Correct">Correct</option>
                <option value="Wrong">Wrong</option>
              </select>
            </label>
            <label className="form-field">
              <span>复核状态</span>
              <select
                onChange={(event) => setReviewFilter(event.target.value as 'all' | SubmissionRecord['reviewState'])}
                value={reviewFilter}
              >
                <option value="all">全部</option>
                <option value="clear">Clear</option>
                <option value="watch">Watch</option>
                <option value="blocked">Blocked</option>
              </select>
            </label>
            <article className="detail-list-row stacked admin-focus-card">
              <span>当前聚焦</span>
              <strong>#{selectedSubmission.id}</strong>
              <small>
                {selectedSubmission.player} / {selectedSubmission.challenge}
              </small>
            </article>
          </div>

          <div className="admin-mini-metrics admin-mini-metrics-relaxed">
            <div className="metric-card admin-mini-metric">
              <span>今日提交</span>
              <strong>{submissionRecords.length}</strong>
            </div>
            <div className="metric-card admin-mini-metric">
              <span>需观察</span>
              <strong>{submissionRecords.filter((item) => item.reviewState === 'watch').length}</strong>
            </div>
            <div className="metric-card admin-mini-metric">
              <span>需拦截</span>
              <strong>{submissionRecords.filter((item) => item.reviewState === 'blocked').length}</strong>
            </div>
          </div>
        </div>
      </section>

      <section className="admin-ops-layout admin-ops-layout-relaxed">
        <article className="panel admin-record-panel admin-list-panel">
          <div className="panel-head compact-head">
            <div>
              <p className="section-kicker">Submission Stream</p>
              <h3>提交流</h3>
            </div>
            <small>{filteredSubmissions.length} 条</small>
          </div>
          <div className="admin-record-list">
            {filteredSubmissions.length > 0 ? (
              filteredSubmissions.map((item) => {
                const itemStatusMeta = getSubmissionStatusMeta(item.status)
                const itemReviewMeta = getSubmissionReviewMeta(item.reviewState)
                const active = item.id === selectedSubmission.id
                return (
                  <button
                    className={active ? 'admin-table-row admin-select-row admin-select-row-compact active' : 'admin-table-row admin-select-row admin-select-row-compact'}
                    key={item.id}
                    onClick={() => props.onSelectSubmission(item.id)}
                    type="button"
                  >
                    <div className="admin-select-head">
                      <strong>{item.challenge}</strong>
                      <small>#{item.id}</small>
                    </div>
                    <p>{item.player}</p>
                    <div className="admin-row-meta">
                      <small className={itemStatusMeta.className}>{itemStatusMeta.label}</small>
                      <small className={itemReviewMeta.className}>{itemReviewMeta.label}</small>
                      <small>{item.source}</small>
                    </div>
                    <div className="admin-row-actions">
                      <span>{item.submittedAt}</span>
                      <small>{item.latency}</small>
                    </div>
                  </button>
                )
              })
            ) : (
              <div className="admin-empty-state">没有匹配当前筛选条件的提交记录。</div>
            )}
          </div>
        </article>

        <div className="admin-section-stack admin-editor-stack">
          <article className="panel admin-detail-panel">
            <div className="panel-head compact-head">
              <div>
                <p className="section-kicker">Submission Detail</p>
                <h3>提交详情</h3>
              </div>
              <div className="admin-row-meta">
                <small className={statusMeta.className}>{statusMeta.label}</small>
                <small className={reviewMeta.className}>{reviewMeta.label}</small>
              </div>
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
              <div className="detail-list-row">
                <span>来源</span>
                <small>{selectedSubmission.source}</small>
              </div>
              <div className="detail-list-row">
                <span>判题链路</span>
                <small>
                  {selectedSubmission.matchedPolicy} / {selectedSubmission.latency}
                </small>
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

            <div className="admin-inline-actions">
              <button className="ghost-button slim" type="button">
                标记已核查
              </button>
              <button className="ghost-button slim" type="button">
                加入观察
              </button>
              <button className="primary-button slim" type="button">
                记录处置意见
              </button>
            </div>
          </article>

          <section className="admin-associated-grid admin-associated-grid-relaxed">
            <article className="panel admin-detail-panel">
              <div className="panel-head compact-head">
                <div>
                  <p className="section-kicker">Review Notes</p>
                  <h3>复核备注</h3>
                </div>
              </div>
              <div className="admin-note-block">{selectedSubmission.note}</div>
              {challengeContext && (
                <div className="detail-list detail-stack-list">
                  <div className="detail-list-row">
                    <span>题目状态</span>
                    <small>{getChallengeReleaseMeta(challengeContext.releaseState).label}</small>
                  </div>
                  <div className="detail-list-row">
                    <span>运行健康</span>
                    <small>{getRuntimeHealthMeta(challengeContext.runtimeHealth).label}</small>
                  </div>
                  <div className="detail-list-row">
                    <span>题目反馈</span>
                    <small>
                      {challengeContext.solveCount} solve / {challengeContext.wrongCount} wrong
                    </small>
                  </div>
                </div>
              )}
            </article>

            <article className="panel admin-detail-panel">
              <div className="panel-head compact-head">
                <div>
                  <p className="section-kicker">Player Context</p>
                  <h3>选手上下文</h3>
                </div>
              </div>
              <div className="detail-list">
                {playerHistory.map((item) => {
                  const itemStatusMeta = getSubmissionStatusMeta(item.status)
                  return (
                    <div className="detail-list-row stacked" key={item.id}>
                      <div className="admin-related-meta">
                        <strong>#{item.id}</strong>
                        <small>{item.challenge}</small>
                        <small className={itemStatusMeta.className}>{itemStatusMeta.label}</small>
                      </div>
                      <small>{item.submittedAt}</small>
                    </div>
                  )
                })}
                {relatedInstances.length > 0 && (
                  <div className="detail-list-row stacked">
                    <span>关联实例</span>
                    <small>
                      {relatedInstances.map((item) => `#${item.id} ${item.status} ${item.expiresIn}`).join(' / ')}
                    </small>
                  </div>
                )}
              </div>
            </article>
          </section>
        </div>
      </section>
    </div>
  )
}

function AdminInstancesSection(props: { selectedInstanceId: number; onSelectInstance: (id: number) => void }) {
  const [searchValue, setSearchValue] = useState('')
  const [riskFilter, setRiskFilter] = useState<'all' | InstanceRecord['risk']>('all')
  const [runningOnly, setRunningOnly] = useState(false)

  const filteredInstances = useMemo(
    () =>
      instanceRecords.filter((item) => {
        const keyword = searchValue.trim().toLowerCase()
        const matchesKeyword =
          keyword.length === 0 ||
          [item.challenge, item.player, item.region, item.endpoint].some((part) => part.toLowerCase().includes(keyword))
        const matchesRisk = riskFilter === 'all' || item.risk === riskFilter
        const matchesRunning = !runningOnly || item.status === 'running'
        return matchesKeyword && matchesRisk && matchesRunning
      }),
    [riskFilter, runningOnly, searchValue],
  )

  const selectedInstance =
    filteredInstances.find((item) => item.id === props.selectedInstanceId) ??
    filteredInstances[0] ??
    instanceRecords.find((item) => item.id === props.selectedInstanceId) ??
    instanceRecords[0]

  const riskMeta = getInstanceRiskMeta(selectedInstance.risk)
  const relatedChallenge = adminChallenges.find((item) => item.id === selectedInstance.challengeId)

  return (
    <div className="admin-section-stack">
      <section className="panel admin-filter-panel">
        <div className="panel-head compact-head">
          <div>
            <p className="section-kicker">Runtime Actions</p>
            <h3>实例处置</h3>
          </div>
          <div className="admin-inline-actions">
            <button className="ghost-button slim" type="button">
              刷新实例列表
            </button>
            <button className="ghost-button slim" onClick={() => setRunningOnly((current) => !current)} type="button">
              {runningOnly ? '显示全部实例' : '仅看运行中'}
            </button>
            <button className="ghost-button slim" type="button">
              导出记录
            </button>
          </div>
        </div>

        <div className="admin-filter-stack">
          <div className="admin-filter-grid admin-filter-grid-relaxed">
            <label className="form-field">
              <span>搜索实例</span>
              <input
                onChange={(event) => setSearchValue(event.target.value)}
                placeholder="题目 / 选手 / 区域 / 入口"
                type="text"
                value={searchValue}
              />
            </label>
            <label className="form-field">
              <span>风险级别</span>
              <select
                onChange={(event) => setRiskFilter(event.target.value as 'all' | InstanceRecord['risk'])}
                value={riskFilter}
              >
                <option value="all">全部</option>
                <option value="stable">Stable</option>
                <option value="expiring">Expiring</option>
                <option value="stuck">Stuck</option>
              </select>
            </label>
            <article className="detail-list-row stacked admin-focus-card">
              <span>当前聚焦</span>
              <strong>实例 #{selectedInstance.id}</strong>
              <small>
                {selectedInstance.player} / {selectedInstance.challenge}
              </small>
            </article>
          </div>

          <div className="admin-mini-metrics admin-mini-metrics-relaxed">
            <div className="metric-card admin-mini-metric">
              <span>运行中</span>
              <strong>{instanceRecords.filter((item) => item.status === 'running').length}</strong>
            </div>
            <div className="metric-card admin-mini-metric">
              <span>即将过期</span>
              <strong>{instanceRecords.filter((item) => item.risk === 'expiring').length}</strong>
            </div>
            <div className="metric-card admin-mini-metric">
              <span>启动卡住</span>
              <strong>{instanceRecords.filter((item) => item.risk === 'stuck').length}</strong>
            </div>
          </div>
        </div>
      </section>

      <section className="admin-ops-layout admin-ops-layout-relaxed">
        <article className="panel admin-record-panel admin-list-panel">
          <div className="panel-head compact-head">
            <div>
              <p className="section-kicker">Instance Queue</p>
              <h3>实例列表</h3>
            </div>
            <small>{filteredInstances.length} 个</small>
          </div>
          <div className="admin-record-list">
            {filteredInstances.length > 0 ? (
              filteredInstances.map((item) => {
                const itemRiskMeta = getInstanceRiskMeta(item.risk)
                const active = item.id === selectedInstance.id
                return (
                  <button
                    className={active ? 'admin-table-row admin-select-row admin-select-row-compact active' : 'admin-table-row admin-select-row admin-select-row-compact'}
                    key={item.id}
                    onClick={() => props.onSelectInstance(item.id)}
                    type="button"
                  >
                    <div className="admin-select-head">
                      <strong>{item.challenge}</strong>
                      <small>实例 #{item.id}</small>
                    </div>
                    <p>{item.player}</p>
                    <div className="admin-row-meta">
                      <small className={`instance-chip instance-${item.status}`}>{item.status}</small>
                      <small className={itemRiskMeta.className}>{itemRiskMeta.label}</small>
                      <small>{item.region}</small>
                    </div>
                    <div className="admin-row-actions">
                      <span>{item.expiresIn}</span>
                      <small>{item.actionLabel}</small>
                    </div>
                  </button>
                )
              })
            ) : (
              <div className="admin-empty-state">没有匹配当前筛选条件的实例记录。</div>
            )}
          </div>
        </article>

        <div className="admin-section-stack admin-editor-stack">
          <article className="panel admin-detail-panel">
            <div className="panel-head compact-head">
              <div>
                <p className="section-kicker">Instance Detail</p>
                <h3>实例详情</h3>
              </div>
              <div className="admin-row-meta">
                <small className={`instance-chip instance-${selectedInstance.status}`}>{selectedInstance.status}</small>
                <small className={riskMeta.className}>{riskMeta.label}</small>
              </div>
            </div>

            <div className="detail-list detail-stack-list">
              <div className="detail-list-row">
                <span>题目</span>
                <small>{selectedInstance.challenge}</small>
              </div>
              <div className="detail-list-row">
                <span>选手</span>
                <small>{selectedInstance.player}</small>
              </div>
              <div className="detail-list-row stacked">
                <span>访问入口</span>
                <code>{selectedInstance.endpoint}</code>
              </div>
              <div className="detail-list-row">
                <span>镜像</span>
                <small>{selectedInstance.image}</small>
              </div>
              <div className="detail-list-row">
                <span>运行区域</span>
                <small>{selectedInstance.region}</small>
              </div>
              <div className="detail-list-row">
                <span>当前剩余</span>
                <small>{selectedInstance.expiresIn}</small>
              </div>
              <div className="detail-list-row">
                <span>运行时长</span>
                <small>{selectedInstance.uptime}</small>
              </div>
              <div className="detail-list-row">
                <span>托管者</span>
                <small>{selectedInstance.owner}</small>
              </div>
            </div>

            <div className="admin-inline-actions">
              <button className="ghost-button slim" type="button">
                查看容器日志
              </button>
              <button className="ghost-button slim" type="button">
                标记关注
              </button>
              <button className="primary-button slim" type="button">
                {selectedInstance.actionLabel}
              </button>
            </div>
          </article>

          <section className="admin-associated-grid admin-associated-grid-relaxed">
            <article className="panel admin-detail-panel">
              <div className="panel-head compact-head">
                <div>
                  <p className="section-kicker">Lifecycle</p>
                  <h3>事件时间线</h3>
                </div>
              </div>
              <div className="timeline-list admin-timeline">
                {selectedInstance.events.map((event) => (
                  <div className="timeline-item" key={`${selectedInstance.id}-${event.time}`}>
                    <span>{event.time}</span>
                    <p>{event.text}</p>
                  </div>
                ))}
              </div>
            </article>

            <article className="panel admin-detail-panel">
              <div className="panel-head compact-head">
                <div>
                  <p className="section-kicker">Challenge Context</p>
                  <h3>题目上下文</h3>
                </div>
              </div>
              {relatedChallenge ? (
                <div className="detail-list detail-stack-list">
                  <div className="detail-list-row">
                    <span>题目状态</span>
                    <small>{getChallengeReleaseMeta(relatedChallenge.releaseState).label}</small>
                  </div>
                  <div className="detail-list-row">
                    <span>运行健康</span>
                    <small>{getRuntimeHealthMeta(relatedChallenge.runtimeHealth).label}</small>
                  </div>
                  <div className="detail-list-row stacked">
                    <span>维护备注</span>
                    <small>{relatedChallenge.notes}</small>
                  </div>
                </div>
              ) : (
                <div className="admin-empty-state">当前实例未关联到题目配置。</div>
              )}
            </article>
          </section>
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
