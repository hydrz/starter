import { useGetHealth } from "./api/generated/enterprise";
import { Announcements } from "./features/announcements/Announcements";

const modules = [
  { name: "契约中心", detail: "TypeSpec 驱动的 API 设计", state: "已接入" },
  { name: "数据访问", detail: "PostgreSQL · sqlc · pgx", state: "已接入" },
  { name: "交付流水线", detail: "单二进制 · Docker Compose", state: "待接入" },
];

function App() {
  const health = useGetHealth({
    query: {
      refetchInterval: 30_000,
      retry: 1,
    },
  });
  const serviceAvailable = health.data?.data.status === "ok";

  return (
    <div className="shell">
      <aside className="sidebar">
        <a className="brand" href="/" aria-label="Enterprise Console 首页">
          <span className="brand-mark">E</span>
          <span>Enterprise</span>
        </a>
        <nav aria-label="主导航">
          <a className="nav-item active" href="#overview">
            <span>概览</span>
            <span className="nav-dot" />
          </a>
          <a className="nav-item" href="#architecture">
            架构
          </a>
          <a className="nav-item" href="#workflow">
            工作流
          </a>
          <a
            className="nav-item"
            href="/api/docs"
            target="_blank"
            rel="noreferrer"
          >
            API 文档
          </a>
        </nav>
        <div className="sidebar-footer">
          <span className={serviceAvailable ? "pulse" : "pulse unavailable"} />
          {serviceAvailable ? "API 服务运行正常" : "正在连接 API 服务"}
        </div>
      </aside>

      <main>
        <header className="topbar">
          <div>
            <span className="eyebrow">OPERATIONS PLATFORM</span>
            <h1>工程控制台</h1>
          </div>
          <span className="phase">Phase 05</span>
        </header>

        <section className="hero" id="overview">
          <div>
            <p className="kicker">Foundation established</p>
            <h2>
              从清晰的边界开始，
              <br />
              持续交付可靠的软件。
            </h2>
            <p className="hero-copy">
              统一 Go 与 React
              的开发入口，为契约、数据和自动化流水线预留稳定边界。
            </p>
          </div>
          <div className="metric">
            <span>当前阶段</span>
            <strong>05</strong>
            <small>纵向业务切片</small>
          </div>
        </section>

        <Announcements />

        <section className="section" id="architecture">
          <div className="section-heading">
            <div>
              <span className="eyebrow">SYSTEM FOUNDATION</span>
              <h3>能力模块</h3>
            </div>
            <span className="section-meta">3 PLANNED</span>
          </div>
          <div className="cards">
            {modules.map((module, index) => (
              <article className="card" key={module.name}>
                <span className="card-number">0{index + 1}</span>
                <div>
                  <h4>{module.name}</h4>
                  <p>{module.detail}</p>
                </div>
                <span className="card-state">{module.state}</span>
              </article>
            ))}
          </div>
        </section>

        <section className="workflow" id="workflow">
          <span className="eyebrow">DEVELOPMENT LOOP</span>
          <div className="workflow-row">
            {["设计", "实现", "验证", "交付"].map((step, index) => (
              <div className="workflow-step" key={step}>
                <span>{index + 1}</span>
                <strong>{step}</strong>
              </div>
            ))}
          </div>
        </section>
      </main>
    </div>
  );
}

export default App;
