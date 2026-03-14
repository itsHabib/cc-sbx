# teams-sbx

A sandbox for exploring [Claude Code](https://claude.ai/code) — with a focus on subagents and multi-agent teams. Each sub-project explores prompting patterns, team structures, and workflows for using Claude Code agents to design and build non-trivial systems from scratch.

## Custom Skills

This repo includes [Claude Code skills](https://docs.anthropic.com/en/docs/claude-code) in `.claude/skills/` that power the team-based workflows used across all projects. They're automatically available when you open the repo in Claude Code.

| Skill | Description |
|-------|-------------|
| `/team-code-project` | Scaffold a complete project — generates `PROJECT.md` and phase kickoff YAMLs from a few questions. |
| `/init-team-project` | Interactively define a new project from scratch into a `PROJECT.md` source-of-truth doc. |
| `/new-team-phase` | Scaffold a new phase config YAML (teammates, roles, focus areas, dependencies). |
| `/team-kickoff` | Spawn a multi-agent planning team from a phase config — each teammate writes a plan doc in parallel. |
| `/team-execute` | Re-spawn teammates to write full deliverables from their approved plan docs. |
| `/team-code-execute` | Like `/team-execute` but for code — teammates write code and self-verify with build/test commands. |
| `/team-to-orchestra` | Convert `PROJECT.md` + kickoff YAMLs into an `orchestra.yaml` for the [agent-orchestra](./agent-orchestra) CLI. |
| `/hackathon-init` | Scaffold a complete agent hackathon — teams, phases, output directories, and judging criteria. |

## Table of Contents

- [Projects](#projects)
  - [pubsub-router](#pubsub-router)
    - [Agent Workflow](#agent-workflow)
    - [Prompts](#prompts)
    - [Plans / Docs](#plans--docs)
    - [Learnings](#learnings)
  - [course-builder](#course-builder)
    - [Agent Workflow](#agent-workflow-1)
    - [Phases](#phases)
    - [Plans / Docs](#plans--docs-1)
  - [wellness-team](#wellness-team)
    - [Agent Workflow](#agent-workflow-2)
    - [Team Members](#team-members)
    - [Plans / Docs](#plans--docs-2)
  - [agent-hackathon](#agent-hackathon)
    - [Agent Workflow](#agent-workflow-3)
    - [Plans / Docs](#plans--docs-3)
  - [agent-orchestra](#agent-orchestra)
    - [Architecture](#architecture)
  - [ingest](#ingest)
    - [Agent Workflow](#agent-workflow-4)
    - [Phases](#phases-1)
    - [Plans / Docs](#plans--docs-4)
  - [skill-sync](#skill-sync)
    - [Agent Workflow](#agent-workflow-5)
    - [Commands](#commands)
  - [babel-protocol](#babel-protocol)
    - [Agent Workflow](#agent-workflow-6)
    - [A/B Test](#ab-test)

---

## Projects

### pubsub-router

A Kafka-backed Pub/Sub Router with an Admin UI and API. Events are ingested over HTTP, routed through Kafka topics, and delivered to subscriber gRPC endpoints with retries, a dead-letter queue, and full observability (Prometheus + Grafana).

#### Agent Workflow

**First Pass: "Big Team" (didn't work well)**

The initial approach was to spawn a single large team covering all disciplines at once — frontend, backend, infra, SRE, etc. — and have them coordinate on the whole system simultaneously.

**Problem:** Too much coordination overhead, context bleed between roles, and agents stepping on each other's work. The scope was too broad for a single team to handle coherently in one pass.

**Second Pass: Dedicated Teams per Phase (worked well)**

Scrapped the big team and ran focused, purpose-built teams one phase at a time. Each team was small (usually 2 agents) with a clearly scoped output and a strict no-code / plan-first rule enforced via `/docs/TEMPLATE.md`.

| Phase | Team / Agents | Output |
|---|---|---|
| **1. Architecture** | Architect + Devil's Advocate | `docs/arch/ARCH.md`, `docs/arch/DECISIONS.md` |
| **2. Backend — Admin API** | Backend agent (Admin API) | `docs/backend/ADMIN_API.md` |
| **3. Backend — Ingest** | Backend agent (Ingest) | `docs/backend/INGEST_DELIVERY.md` |
| **4. Backend — Delivery** | Backend agent (Delivery) | `docs/backend/DELIVERY_WORKER.md` |
| **5. Frontend** | Frontend agent | `docs/ui/FRONTEND_PLAN.md` |
| **6. Testing / SDET** | SDET lead + Test infra | `docs/test/E2E_PLAN.md`, `docs/test/INFRA_PLAN.md` |
| **7. Performance** | Perf analyst + Telemetry eng + Load test eng | `docs/perf/PERF_PLAN.md`, `docs/perf/TELEMETRY_PLAN.md`, `docs/perf/LOADTEST_PLAN.md` |
| **8. Review** | 6 reviewers (frontend, admin-api, ingest, delivery, security, infra) | `docs/review/*.md`, `docs/review/REVIEW_SUMMARY.md` |

Agent roles used: Architect, Devil's Advocate, Backend (Admin API), Backend (Ingest), Backend (Delivery), Frontend, Perf Analyst, Telemetry Engineer, Load Test Engineer, SDET Lead, Test Infra, Reviewers (x6).

#### Prompts

All agent kickoff prompts live in [`pubsub-router/prompts/`](./pubsub-router/prompts/):

| File | Purpose |
|---|---|
| `teams.md` | Master plan template all agents follow |
| `arch.md` | Architect kickoff |
| `backend-kickoff.md` | Backend phase kickoff (Admin API + Ingest/Delivery) |
| `admin-api.md` | Admin API agent |
| `ingest.md` | Ingest agent |
| `delivery-worker.md` | Delivery worker agent |
| `frontend.md` | Frontend agent |
| `devilsadv.md` | Devil's Advocate agent |
| `perf-kickoff.md` | Performance phase kickoff |
| `perf-analyst.md` | Perf analyst agent |
| `perf-tel-eng.md` | Telemetry engineer agent |
| `load-test-eng.md` | Load test engineer agent |
| `sdet-team-kickoff.md` | SDET phase kickoff |
| `sdet-lead.md` | SDET lead agent |
| `test-infra.md` | Test infra agent |
| `review-kickoff.md` | Review phase kickoff |

#### Plans / Docs

All plans (no code) are in [`pubsub-router/docs/`](./pubsub-router/docs/):

```
docs/
  TEMPLATE.md          # Master template all agents must follow
  arch/                # Architecture decisions and system design
  backend/             # Backend service plans (admin, ingest, delivery)
  ui/                  # Frontend plan
  perf/                # Performance, telemetry, load test plans
  test/                # E2E and test infra plans
  review/              # Per-domain code review docs + summary
```

#### Learnings

- **A big single team doesn't work.** Doing frontend + backend + infra all at once is too much — agents lose focus and context bleeds between roles.
- **Small focused teams work well.** A team of 2 per domain with a clear scope and a single output file per agent keeps things clean.
- **Infra/platform is the hardest area.** Agents tend to make mistakes with Docker setups and platform-level config. Expect more iteration there.
- **Clear context between phases matters.** Reset/clear context after each team run so agents start fresh without carrying stale state.
- **Plan-first enforced by template.** A strict `TEMPLATE.md` that all agents follow (plan mode, no code, fixed output file, end with `READY FOR APPROVAL`) dramatically improves consistency and reviewability.

---

### course-builder

A demonstration of using Claude Code agent teams to produce a complete, learner-ready technical course from scratch. The output is a project-based course that teaches backend developers to build a caching proxy in Go.

#### Agent Workflow

Three skills drove the entire workflow:

- **`/init-team-project`** — interviewed the user and produced `PROJECT.md`, the source-of-truth doc that all teammates read before writing anything.
- **`/new-team-phase`** — scaffolded each phase's `kickoff.yaml` config (teammates, roles, output doc paths, focus areas, and dependencies).
- **`/team-kickoff`** — read the kickoff config, spawned all teammates in parallel (each in plan mode), waited for all plans to land, then synthesized cross-team risks and asked which plan to approve first.

Each phase followed the same loop:

```
/new-team-phase  →  edit kickoff.yaml  →  /team-kickoff
      ↓
teammates write plans in parallel (plan mode)
      ↓
team lead synthesizes, user approves plans
      ↓
teammates implement (write docs / reports)
      ↓
team lead reviews, commits artifacts
```

#### Phases

| Phase | Teammates | Output |
|---|---|---|
| **1. Curriculum Architecture** | Curriculum Architect, Subject Matter Researcher, Instructional Designer, Assessment Designer, QA Skeptic, Publishing Planner | Phase 1 planning docs in `docs/phases/curriculum-architecture/` |
| **2. Content Production** | Module Writers (Foundation / Core / Advanced), Exercise Designer, Quiz Writer, Support Docs Writer | All learner-facing content in `docs/course/` |
| **3. QA & Editorial Review** | QA Reviewer, Editor, Skeptic | Review reports in `docs/phases/qa-editorial/` |

#### Plans / Docs

```
PROJECT.md                          # source of truth — read by all teammates
docs/
  course/                           # all learner-facing content (final artifacts)
    00-overview.md
    modules/                        # m00–m07 lesson files
    exercises/
    assessments/
    glossary-and-resources.md
  phases/                           # planning artifacts and review reports
    curriculum-architecture/        # Phase 1 plans
    content-production/             # Phase 2 (kickoff config only; output → docs/course/)
    qa-editorial/                   # Phase 3 review reports + fix history
```

---

### wellness-team

A multi-disciplinary coaching team that builds a comprehensive 4-month (16-week) training plan to prepare an athlete for the CrossFit Open with a Quarterfinals qualification (top 25%) as the target. All deliverables are markdown documents — no code or infrastructure.

#### Agent Workflow

Four skills drove the entire workflow end-to-end:

1. **`/init-team-project`** — interactive interview that produced `PROJECT.md` (athlete profile, coaching domains, constraints, non-goals).
2. **`/new-team-phase`** — scaffolded `kickoff.yaml` with teammates, roles, output paths, focus areas, and dependencies.
3. **`/team-kickoff`** — spawned all 8 teammates in parallel (plan mode), each writing their domain plan. Head Coach reviewed all plans and issued revision directives.
4. **`/team-execute`** — re-spawned all 8 teammates to write full deliverables from their approved plans, followed by a cross-domain consistency review and targeted fixes.

```
/init-team-project  →  PROJECT.md
        ↓
/new-team-phase  →  kickoff.yaml
        ↓
/team-kickoff  →  8 plans written in parallel
        ↓
/team-execute  →  8 deliverables written in parallel  →  consistency review  →  fixes
```

#### Team Members

| Role | Focus |
|------|-------|
| Head Coach | Cross-domain coordination, master schedule, phase structure, conflict resolution |
| Nutritionist | Macros, meal timing, hydration, supplementation by phase |
| Strength Coach | Resistance training, progressive overload, strength benchmarks |
| Endurance Coach | Aerobic/anaerobic conditioning, engine building, intervals |
| Mobility Coach | Flexibility, joint health, warm-ups, cool-downs, movement screening |
| Recovery & Sleep Coach | Sleep hygiene, rest protocols, deloads, readiness scoring |
| Mental Performance Coach | Visualization, stress inoculation, competition-day routines |
| Competition Strategy Coach | Open/QF tactics, pacing models, score optimization, mock competitions |

#### Plans / Docs

```
wellness-team/
  PROJECT.md                        # source of truth — read by all teammates
  docs/
    crossfit-open-prep/
      kickoff.yaml                  # phase config for /team-kickoff and /team-execute
    plans/                          # approved plans (output of /team-kickoff)
    content/                        # full deliverables (output of /team-execute)
```

---

### agent-hackathon

A mini hackathon run entirely by AI agents — from ideation to code to judging. Two teams of Claude Code subagents competed to build the best AI-powered CLI tool, while a commentator documented the process and AI judges scored the results. No humans wrote any code, pitches, logs, or scorecards.

#### Agent Workflow

9 agents across 3 sequential phases, orchestrated via `/team-kickoff` and `/team-execute`. Engineers on the same team communicated through shared log files; teams were isolated from each other. Judges read source code, built both tools, and ran test suites before scoring.

| Phase | Teammates | Output |
|-------|-----------|--------|
| **1. Ideation** | 4 engineers (2 per team) + 1 commentator | Pitches, team logs, recap |
| **2. Build** | 4 engineers (2 per team) + 1 commentator | Working Go CLI tools with tests |
| **3. Judging** | 3 specialist judges + 1 head judge | Independent scorecards + final results |

#### Plans / Docs

```
agent-hackathon/
  PROJECT.md                            # hackathon rules and format
  docs/
    phase-1-ideation/kickoff.yaml       # ideation phase config
    phase-2-build/kickoff.yaml          # build phase config
    phase-3-judging/kickoff.yaml        # judging phase config
    phase-*/plans/                      # individual agent plan docs
  output/
    team-alpha/                         # config-doctor source code, pitch, logs
    team-beta/                          # git-wtf source code, pitch, logs
    recap.md                            # commentator observations
    scorecard.md                        # final judging results
```

---

### agent-orchestra

A Go CLI that orchestrates large software projects across multiple AI agent teams. Define teams, tasks, and dependencies in an `orchestra.yaml`, then run `orchestra run`. It builds a DAG (Kahn's algorithm), executes teams tier-by-tier (parallel within a tier, sequential across tiers), and flows results forward so downstream teams get full context.

#### Architecture

Each team is spawned as a `claude -p` subprocess with a constructed prompt containing its role, context, tasks, and completed dependency results. Teams can run in solo mode (lead works directly) or team-lead mode (lead spawns teammates via `TeamCreate`).

```
orchestra.yaml  →  validate  →  DAG (Kahn's)  →  tier-by-tier execution
                                                    Tier 0: [backend, auth]    (parallel)
                                                    Tier 1: [frontend, devops] (parallel)
                                                    Tier 2: [integration]
```

---

### ingest

A multi-tenant, Kafka-backed distributed ingestion and query engine in Go. Uses Kafka as the durability layer instead of a traditional Write-Ahead Log (WAL), making ingesters completely stateless. Built from scratch to deeply understand distributed ingestion design — stateless ingesters, consistent hashing, backpressure propagation, at-least-once delivery guarantees, and time-series query execution.

#### Agent Workflow

Built entirely using `/team-kickoff` and `/team-code-execute` across 11 phases, each with 3 specialized AI engineers. Every phase followed the same loop: plan in parallel → approve → code in parallel → self-verify (`go build`, `go test`, `go vet`) → human review between phases.

#### Phases

| Phase | Engineers | Focus |
|-------|-----------|-------|
| **1. Foundation** | infra, model, gateway | Docker Compose (Kafka KRaft), data types, HTTP push endpoint |
| **2. Core Engine** | ring, storage, ingester | Consistent hash ring, object storage, Kafka consumer + batcher |
| **3. Production** | tenancy, observability, integration | Rate limiting, Prometheus/Grafana, S3/MinIO, load generator |
| **4. Quality** | unit-test, integration-test, chaos | Unit coverage, E2E pipeline tests, crash recovery + failure injection |
| **5. Performance** | benchmark, load-test, profiling | Micro-benchmarks, load profiles, pprof optimization |
| **6. Query Foundation** | index, query-api, block-reader | Block index, HTTP query API, parallel block reading |
| **7. Compaction & Caching** | compactor, block-format, cache | Background compaction, structured block format, LRU cache |
| **8. Query Performance** | query-planner, fanout, query-bench | Query planning, parallel fan-out, read-path benchmarks |
| **9. K8s Foundation** | k8s-manifest, kafka-cluster, config | K8s manifests, Strimzi 3-broker Kafka, Kustomize overlays |
| **10. Scaling & Observability** | autoscaling, tracing, resilience | HPA/KEDA autoscaling, OpenTelemetry tracing, PDBs |
| **11. CI/CD & Operations** | cicd, gitops, operations | GitHub Actions, ArgoCD GitOps, alerting rules, runbooks, SLOs |

#### Plans / Docs

```
ingest/
  PROJECT.md                          # source of truth
  AGENTS.md                           # full agent development process
  docs/
    foundation/                       # Phase 1 plans
    core-engine/                      # Phase 2 plans
    production/                       # Phase 3 plans
    quality/                          # Phase 4 plans
    performance/                      # Phase 5 plans + report
    query-foundation/                 # Phase 6 plans
    compaction/                       # Phase 7 plans
    query-performance/                # Phase 8 plans
    k8s-foundation/                   # Phase 9 plans
    k8s-scaling/                      # Phase 10 plans
    k8s-operations/                   # Phase 11 plans
```

---

### skill-sync

A Go CLI that syncs custom skills across AI coding assistant providers. All major providers (Claude Code, GitHub Copilot, Gemini CLI, Factory AI) use the same [Agent Skills open standard](https://docs.github.com/en/copilot/concepts/agents/about-agent-skills) (`SKILL.md` files), but each stores them in a different directory. skill-sync lets you declare one provider as the source of truth and copies skills to all configured targets, with `status` and `diff` commands to catch drift.

#### Agent Workflow

Built using `/team-kickoff` and `/team-code-execute` across two phases with specialized AI engineers. Each phase followed the same plan → approve → code → self-verify loop.

#### Commands

| Command | Purpose |
|---------|---------|
| `skill-sync init` | Create `.skill-sync.yaml` config |
| `skill-sync sync` | Copy skills from source to all targets |
| `skill-sync status` | Report drift across targets (exit 1 if drift detected) |
| `skill-sync diff` | Show unified diffs for modified skills |

---

### babel-protocol

An experiment where Claude Code agent teams invented their own language from scratch for agent-to-agent communication, then compressed it over 3 rounds until it hit the theoretical floor. The result: **88% message compression, 82% all-in savings, 100% task success**, learnable by a fresh agent in 15 minutes from a 104-token spec.

The question: **can AI agents do better than English when talking to each other?** The answer: yes, dramatically.

#### Agent Workflow

Built using `/team-kickoff` and `/team-execute` across 7 phases with ~40 agent spawns. Each phase followed the standard plan → approve → execute loop. Phases 5-7 ran back-to-back with zero manual intervention (~45 minutes wall clock).

| Phase | Focus | Result |
|-------|-------|--------|
| **1. Experiment Design** | Framework, 10 tasks, metrics definitions | Task library + evaluation criteria |
| **2. Baseline** | English encoding of all 10 tasks | 1,676 tokens, 90% success rate |
| **3. Language Genesis** | Agents invent Babel v1 from scratch | 88% message compression, but 2,611-token spec (net worse) |
| **4. Compression Round 1** | Slash the spec | Spec 2,611 → 317 tokens. AICR crosses to net win |
| **5. Compression Round 2** | Hyphenation trick | Spec 317 → 104 tokens. Free lunch |
| **6. Compression Round 3** | Attempt further compression | Floor confirmed. vFinal = v3. STOP |
| **7. Final Analysis** | Fresh agent test, evolution report | 10/10 encode, 10/10 decode, 0 divergences |

#### A/B Test

Ran the same multi-agent task (conference scheduling — 3 agents, 8 speakers, 4 rooms, 6 slots, 11 constraints) twice: once in English, once in Babel. Both produced the identical correct schedule.

| Metric | English | Babel | Delta |
|--------|---------|-------|-------|
| Inter-agent payload | 2,886 words | 459 words | **-84%** |
| Total API tokens | 30,518 | 29,324 | **-4%** |
| Wall clock time | 118s | 98s | **-17%** |
| Accuracy | 11/11 constraints | 11/11 constraints | Same |

API-level savings are modest because inter-agent payloads are a fraction of total session tokens (system prompts, tool overhead, reasoning). In systems with many agents and handoffs, the savings compound.

#### Plans / Docs

```
babel-protocol/
  PROJECT.md                           # Project definition
  docs/
    experiment-design/content/         # Phase 1: framework, task library, metrics
    baseline/content/                  # Phase 2: English baseline transcripts
    language-genesis/content/          # Phase 3: v1 spec, architect, tester
    compress-v1/content/               # Phase 4: v2 spec, round 1 encode/decode
    compress-v2/content/               # Phase 5: v3 spec, round 2 encode/decode
    compress-v3/content/               # Phase 6: vFinal spec, evolution summary
    final-analysis/content/            # Phase 7: fresh agent, evolution, theory, report
  test/                                # A/B test: Babel vs English
    with-babel/                        # Agent outputs using Babel
    without-babel/                     # Agent outputs using English
```

