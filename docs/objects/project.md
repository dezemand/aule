# Project information model

A **Project** is the top-level context boundary.  
It defines *why work exists*, *what is allowed*, *where artefacts live*, and *how autonomous execution may be*.

Think of a project as a **governed workspace**, not just a container.

Below is a full breakdown of what information a project **needs / should / could / would** contain.

---

## 1. Core identity (must have)

These make a project addressable and stable.

- **Project ID**
- **Project key**
  - Short, human-friendly identifier (used in branches, logs, URLs)
- **Name**
- **Description**
  - What this project is about, in plain language
- **Status**
  - active / paused / archived
- **Created at / updated at**
- **Owner**
  - Person or group accountable for outcomes

---

## 2. Purpose & intent (must have)

Defines *why this project exists*.

- **Primary goal**
  - What success looks like
- **Problem statement**
  - What problem this project is trying to solve
- **Non-goals**
  - Explicit exclusions (critical for autonomy)
- **Expected value**
  - Technical, organisational, learning, strategic
- **Time horizon**
  - Short-lived experiment vs long-running initiative

---

## 3. Scope & boundaries (must have)

Protects against uncontrolled expansion.

- **In-scope domains**
  - Systems, teams, processes, technologies
- **Out-of-scope domains**
- **Assumptions**
  - Things taken as given
- **Known constraints**
  - Budget, time, people, regulations

---

## 4. Stakeholders & roles (must have)

Defines who cares and who decides.

- **Primary stakeholders**
  - Engineers, managers, product, operations
- **Decision makers**
  - Who approves major decisions
- **Reviewers**
  - Who reviews outputs by default
- **Contributors**
  - Humans expected to interact with agents
- **Audience profiles**
  - Technical, non-technical, mixed

---

## 5. Governance & autonomy model (must have)

Controls how autonomous agents are allowed to be.

- **Autonomy level**
  - assistive / supervised / autonomous / overnight-autonomous
- **Human-in-the-loop requirements**
  - Which stages require human approval
- **Review strictness**
  - Light / normal / strict
- **Decision authority**
  - Can agents make decisions or only propose?
- **Escalation rules**
  - When and how humans are notified

---

## 6. Task model configuration (must have)

Defines how work flows inside the project.

- **Allowed TaskTypes**
  - Which types are valid in this project
- **Custom TaskStage overrides**
  - If project deviates from defaults
- **Default task priorities**
- **WIP expectations**
  - Soft limits (even without boards)

---

## 7. Agent configuration (must have)

Defines which agents may operate.

- **Allowed AgentTypes**
- **Agent trust level**
  - Experimental / operational / trusted
- **Runtime permissions**
  - Go-only, Python allowed, etc.
- **Max parallel agents**
- **Agent budget limits**
  - Time, cost, tool usage

---

## 8. Artefact & repository attachments (must have)

Defines where work is read from and written to.

### Git repositories
- Repo ID / URL
- Purpose (code, docs, infra, examples)
- Default branch
- Allowed paths (read/write)
- Branch naming conventions

### Wiki / documentation spaces
- Space ID
- Read/write permissions
- Page prefixes or subsets

### Other artefact stores
- Object storage buckets
- Shared folders
- External systems (later via MCP)

---

## 9. Templates & standards (should have)

Ensures consistency across outputs.

- **Document templates**
  - Architecture, ADR, research memo, briefing
- **Naming conventions**
- **Formatting rules**
- **Diagram standards**
- **Code style references**

These are mounted into agent context as read-only.

---

## 10. Knowledge & context packs (should have)

Lightweight shared understanding.

- **Project brief**
- **Glossary**
- **Known decisions / ADRs**
- **Reference links**
- **Background material**

Initially this can just be files; later it may be a knowledge graph.

---

## 11. Policies & compliance (should have)

Defines what is allowed.

- **Security policies**
- **Data classification**
- **Export / sharing restrictions**
- **Legal/compliance constraints**
- **Internal guidelines**

Used by policy-check tools and reviewers.

---

## 12. Tooling & integration configuration (should have)

Controls external interactions.

- **Allowed tools**
  - Git, wiki, web, MCP endpoints
- **Tool scope defaults**
- **Network egress rules**
- **Credential sources**
- **Audit requirements**

---

## 13. Quality & risk profile (should have)

Guides safety mechanisms.

- **Default risk level**
  - Low / medium / high
- **Required policy checks**
- **Required reviewer roles**
- **Failure tolerance**
  - Fail fast vs best-effort

---

## 14. Communication & visibility (should have)

Defines how noisy or quiet the project is.

- **Notification rules**
  - On failure, on block, on completion
- **Progress visibility**
  - Task-level vs run-level summaries
- **Reporting cadence**
  - Daily brief, weekly summary, on-demand

---

## 15. Cost & resource management (should have)

Important for scaling.

- **Cost budget**
  - Soft / hard
- **Time budget**
- **Agent runtime limits**
- **Overnight run permission**
- **Cost attribution tags**

---

## 16. Failure & recovery strategy (should have)

Defines what happens when things go wrong.

- **Retry rules**
- **Fallback to human**
- **Partial result acceptance**
- **Incident logging**
- **Post-run review requirements**

---

## 17. Lifecycle & evolution (could have)

Controls long-term maintenance.

- **Project phases**
  - discovery / delivery / wrap-up
- **Archival rules**
- **Retention policy**
  - Artefacts, logs, events
- **Versioning of project config**

---

## 18. Cross-project relationships (could have)

Enables scaling later.

- **Upstream/downstream projects**
- **Shared artefacts**
- **Shared knowledge**
- **Dependency constraints**

---

## 19. Observability & audit (system-managed but conceptually part of project)

- Event log retention
- Tool receipts
- Agent execution history
- Decision traceability

---

## 20. Project maturity classification (internal but powerful)

- **Experimental**
  - Loose rules, high learning
- **Operational**
  - Structured, review-heavy
- **Critical**
  - Strict controls, limited autonomy

This gates:
- overnight execution
- sensitive tools
- high-risk tasks

---

## Mental model

A Project defines:
- **Why** we are doing work
- **What kinds** of work are allowed
- **Who/what** may act
- **Where** outputs live
- **How safe** autonomy must be

Everything else (tasks, agents, tools) operates *inside* these boundaries.

---

## Strong guidance

- Projects should be **few and meaningful**, not throwaway.
- Most autonomy failures are **project configuration failures**, not agent failures.
- If humans can’t understand a project’s rules, agents won’t either.
