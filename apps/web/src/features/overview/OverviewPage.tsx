import * as m from "../../paraglide/messages";

function getModules() {
  return [
    {
      name: m.overview_module_contracts_name(),
      detail: m.overview_module_contracts_detail(),
    },
    {
      name: m.overview_module_data_name(),
      detail: m.overview_module_data_detail(),
    },
    {
      name: m.overview_module_delivery_name(),
      detail: m.overview_module_delivery_detail(),
    },
  ];
}

function getWorkflowSteps() {
  return [
    m.overview_workflow_step_design(),
    m.overview_workflow_step_build(),
    m.overview_workflow_step_verify(),
    m.overview_workflow_step_ship(),
  ];
}

/** 控制台概览页：迁移自旧 `router.tsx` 的内联 `OverviewPage`（文案已接入 Paraglide）。 */
export function OverviewPage() {
  const modules = getModules();
  const workflowSteps = getWorkflowSteps();

  return (
    <>
      <section className="hero">
        <div>
          <p className="kicker">{m.overview_kicker()}</p>
          <h2>{m.overview_heading()}</h2>
          <p className="hero-copy">{m.overview_copy()}</p>
        </div>

        <div className="metric">
          <span>{m.overview_metric_label()}</span>
          <strong>{m.overview_metric_value()}</strong>
          <small>{m.overview_metric_hint()}</small>
        </div>
      </section>

      <section className="section" aria-labelledby="modules-title">
        <div className="section-heading">
          <div>
            <span className="eyebrow">{m.overview_modules_eyebrow()}</span>
            <h3 id="modules-title">{m.overview_modules_title()}</h3>
          </div>
          <span className="section-meta">{m.overview_modules_meta()}</span>
        </div>

        <div className="cards">
          {modules.map((module, index) => (
            <article key={module.name} className="card">
              <span className="card-number">0{index + 1}</span>
              <div>
                <h4>{module.name}</h4>
                <p>{module.detail}</p>
              </div>
              <span className="card-state">
                {m.overview_module_state_connected()}
              </span>
            </article>
          ))}
        </div>
      </section>

      <section className="workflow" aria-labelledby="workflow-title">
        <div className="section-heading">
          <div>
            <span className="eyebrow">{m.overview_workflow_eyebrow()}</span>
            <h3 id="workflow-title">{m.overview_workflow_title()}</h3>
          </div>
          <span className="section-meta">{m.overview_workflow_meta()}</span>
        </div>

        <div className="workflow-row">
          {workflowSteps.map((step, index) => (
            <div key={step} className="workflow-step">
              <span>0{index + 1}</span>
              <strong>{step}</strong>
            </div>
          ))}
        </div>
      </section>
    </>
  );
}
