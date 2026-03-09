# Master Plan Template (Agent Teams)

> Follow this template **verbatim** in your `/docs/<phase>/<role>.md` output.
> - You are in **PLAN MODE**.
> - **No code**.
> - Keep the "Tighten the plan…" section to **4–7 tasks**.
> - End with **READY FOR APPROVAL**.

---

## You are in PLAN MODE.

### Project
I want to do a **<new project / POC / system>**.

**Goal:** build a **<thing>** in which we **<outcome>**.

### Role + Scope (fill in)
- **Role:** <Architect | Backend Admin API | Ingest/Kafka | Delivery | SRE/DevOps | Frontend>
- **Scope:** <what you own; what you explicitly do NOT own>
- **File you will write:** `/docs/<phase>/<role>.md`
- **No-touch zones:** do not edit any other files; do not write code.

---

## Functional Requirements
- <FR1>
- <FR2>
- <FR3>
- <Tests required: unit + integration (where applicable)>
- <Metrics required: Prometheus + Grafana (where applicable)>

## Non-Functional Requirements
- Language/runtime: <Go for backend; frontend TBD>
- Local dev: docker-compose brings up everything
- Observability: `/metrics` + dashboards
- Safety: clear failure behavior, retries/backoff if relevant
- Documentation: README + CLAUDE.md + EXPLAIN.md contributions
- Performance: benchmarks for the critical path (if relevant)

---

## Assumptions / System Model
- Deployment environment: <local docker-compose; later k8s optional>
- Failure modes: <process crash, Kafka unavailable, network timeout, bad inputs>
- Delivery guarantees: <at-least-once, duplicates possible, ordering scope>
- Multi-tenancy: <none for MVP / basic tenant field / future>

---

## Data Model (as relevant to your role)
Describe the entities and required fields.

Example (edit/remove as needed):
- **Topic**
  - id, name, partitions, retention, created_at
- **Subscription**
  - id, topic, filter, target (gRPC endpoint/service), retry policy, DLQ policy, enabled, created_at, updated_at
- **Event Envelope**
  - id, type, source, subject, time, attributes(map), payload(bytes), content_type

Include:
- validation rules
- versioning strategy (if config changes matter)
- minimal persistence approach for MVP (in-memory + snapshot OR sqlite)

---

## APIs (as relevant to your role)
Define API surface precisely (HTTP endpoints and/or gRPC). Include request/response schemas and error semantics.

### Admin API (example shape)
- `POST /topics`
- `GET /topics`
- `POST /subscriptions`
- `GET /subscriptions`
- `PATCH /subscriptions/{id}` enable/disable/update

### Ingest API (example shape)
- `POST /events:publish` (HTTP)
- Optional: `PublishStream` (gRPC streaming ingestion) — decide and justify

### Delivery Target Contract (example shape)
- gRPC method that a subscriber must implement
- expected ack/nack semantics and timeouts

---

## Architecture / Component Boundaries (as relevant)
Describe the components you touch and their responsibilities.

Example:
- **Admin service**: stores topics/subscriptions, serves UI/API
- **Ingest service**: validates envelope, routes to Kafka
- **Delivery workers**: consume Kafka, deliver to subscription targets, retries + DLQ
- **Storage**: config metadata + (optional) delivery attempt state
- **Observability**: Prom metrics + Grafana dashboards

Include:
- how config changes propagate (polling, watch endpoint, event bus)
- concurrency model (goroutines, worker pools, rate limits)
- backpressure strategy (when Kafka or targets are slow)

---

## Correctness Invariants (must be explicit)
List invariants you will protect with tests.

Examples:
- A subscription marked `enabled=false` receives no deliveries.
- Delivery is **at-least-once**: duplicates possible; consumers must be idempotent.
- Retry attempts never exceed `max_attempts`; then message goes to DLQ.
- Invalid events are rejected and/or quarantined with metrics emitted.
- Admin operations are validated and idempotent where appropriate.

---

## Tests
Be specific. List:
- unit tests (which modules)
- integration tests (docker-compose: publish → Kafka → deliver → observe)
- property/fuzz tests (optional but great)
- failure injection tests (timeouts, Kafka down, target 500, etc.)

Include exact commands:
- `go test ./...`
- `go test ./... -run TestX`
- `go test ./... -bench . -benchmem` (if applicable)

---

## Benchmarks + "Success"
If benchmarking is relevant, define:
- what to measure (throughput, latency, allocations)
- target success criteria (e.g., ">= 10k events/s locally", "p95 publish < 20ms")
- benchmark command(s)

If not relevant for your role, say "N/A" and why.

---

## Engineering Decisions & Tradeoffs (REQUIRED)
For every non-trivial design choice in your scope, document:
- **Decision:** what you chose
- **Alternatives considered:** what else you evaluated (at least 1)
- **Why:** the reasoning — performance, simplicity, maintainability, consistency with codebase, etc.
- **Tradeoff acknowledged:** what you gave up by choosing this path

This section must have **at least 2 entries**. Examples of decisions worth documenting:
- Library choice (e.g., gomock vs hand-rolled fakes)
- Data structure or algorithm choice
- API design decisions (naming, error semantics, payload format)
- Test strategy choices (fuzz vs property vs table-driven)
- Concurrency patterns (channels vs mutexes, worker pools vs goroutine-per-request)

---

## Risks & Mitigations (REQUIRED)
List the **top 3–5 risks or unknowns** that could block or derail this work.

For each risk include:
- **Risk:** what could go wrong
- **Impact:** what happens if it materializes (blocked? data loss? flaky tests?)
- **Mitigation:** concrete action to validate or reduce the risk
- **Validation time:** how long to validate (target **< 15 minutes** each)

Do not leave this section empty or write generic risks. Be specific to your role and scope.

---

# Please produce (no code yet):
1) **Recommended API surface** (1–N functions/endpoints) and exact behavior/spec
2) **Folder structure** (what packages/services/modules exist; who owns what)
3) **Step-by-step task plan** in small commits (each: files touched + how to verify)
4) **Benchmark plan** + what "success" looks like

**Stop and ask me to approve before writing code.**

---

# Tighten the plan into 4–7 small tasks (STRICT)
For each task include:
- **Outcome**
- **Files to create/modify**
- **Exact verification command(s)** (tests/bench/run)
- **Suggested commit message**

Do not write code yet.

---

# CLAUDE.md contributions (do NOT write the file; propose content)
Add a section titled:

## From <ROLE>
Include:
- coding style rules
- dev commands
- "before you commit" checklist
- any guardrails (e.g., "no breaking API changes without updating docs/tests")

---

# EXPLAIN.md contributions (do NOT write the file; propose outline bullets)
Provide bullets for:
- flow / architecture explanation
- key engineering decisions + tradeoffs
- limits of MVP + next steps
- how to run locally + how to validate

---

## READY FOR APPROVAL
(Write this exact line at the end.)
