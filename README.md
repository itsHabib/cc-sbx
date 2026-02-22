# teams-sbx

A sandbox for exploring [Claude Code](https://claude.ai/code) — with a focus on subagents and multi-agent teams. Each sub-project explores prompting patterns, team structures, and workflows for using Claude Code agents to design and build non-trivial systems from scratch.

---

## Projects

### [`pubsub-router`](./pubsub-router)

A Kafka-backed Pub/Sub Router with an Admin UI and API. Events are ingested over HTTP, routed through Kafka topics, and delivered to subscriber gRPC endpoints with retries, a dead-letter queue, and full observability (Prometheus + Grafana).

---

## Agent Workflow — pubsub-router

### First Pass: "Big Team" (didn't work well)

The initial approach was to spawn a single large team covering all disciplines at once — frontend, backend, infra, SRE, etc. — and have them coordinate on the whole system simultaneously.

**Problem:** Too much coordination overhead, context bleed between roles, and agents stepping on each other's work. The scope was too broad for a single team to handle coherently in one pass.

### Second Pass: Dedicated Teams per Phase (worked well)

Scrapped the big team and ran focused, purpose-built teams one phase at a time. Each team was small (usually 2 agents) with a clearly scoped output and a strict no-code / plan-first rule enforced via `/docs/TEMPLATE.md`.

#### Phase Sequence

| Phase | Team / Agents | Output |
|---|---|---|
| **Architecture** | Architect + Devil's Advocate | `docs/arch/ARCH.md`, `docs/arch/DECISIONS.md` |
| **Backend — Admin API** | Backend agent (Admin API) | `docs/backend/ADMIN_API.md` |
| **Backend — Ingest + Delivery** | Backend agent (Ingest/Delivery) | `docs/backend/INGEST_DELIVERY.md`, `docs/backend/DELIVERY_WORKER.md` |
| **Frontend** | Frontend agent | `docs/ui/FRONTEND_PLAN.md` |
| **Performance** | Perf analyst + Telemetry eng + Load test eng | `docs/perf/PERF_PLAN.md`, `docs/perf/TELEMETRY_PLAN.md`, `docs/perf/LOADTEST_PLAN.md` |
| **Testing / SDET** | SDET lead + Test infra | `docs/test/E2E_PLAN.md`, `docs/test/INFRA_PLAN.md` |
| **Review** | 6 reviewers (frontend, admin-api, ingest, delivery, security, infra) | `docs/review/*.md`, `docs/review/REVIEW_SUMMARY.md` |

#### Agent Roles Used

- **Architect / System Designer** — overall component boundaries, data model, API contracts, delivery semantics, folder structure
- **Devil's Advocate** — challenges assumptions, surfaces risks and edge cases in the architecture
- **Backend (Admin API)** — admin HTTP API and storage layer plan
- **Backend (Ingest + Delivery)** — Kafka ingest, delivery workers, retry/DLQ logic
- **Frontend** — Admin UI component plan
- **SRE / DevOps** — docker-compose, infra, observability setup
- **Perf Analyst** — performance benchmarks and success criteria
- **Telemetry Engineer** — Prometheus metrics and Grafana dashboards
- **Load Test Engineer** — load test scenarios and tooling
- **SDET Lead** — end-to-end test strategy
- **Test Infra** — test infrastructure and CI setup
- **Reviewers (x6)** — code review and simplification suggestions per domain

---

## Prompts

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

---

## Plans / Docs

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

---

## Learnings

See [`learnings.md`](./learnings.md) for notes. Summary:

- **A big single team doesn't work.** Doing frontend + backend + infra all at once in one team is too much — agents lose focus and context bleeds between roles.
- **Small focused teams work well.** A team of 2 per domain (e.g., two backend agents) with a clear scope and a single output file per agent keeps things clean.
- **Infra/platform is the hardest area.** Agents tend to make mistakes with Docker setups and platform-level config. Expect more iteration there.
- **Clear context between phases matters.** Reset/clear context after each team run so agents start fresh without carrying stale state from prior phases.
- **Plan-first enforced by template.** Having a strict `TEMPLATE.md` that all agents follow (plan mode, no code, fixed output file, end with `READY FOR APPROVAL`) dramatically improves consistency and reviewability.
