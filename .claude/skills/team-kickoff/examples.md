# Examples

## Example 1: Performance Phase

### kickoff.yaml (input)

```yaml
phase: "Performance"

goal: >
  Define how we measure, generate load, and evaluate performance
  before any implementation or optimization work begins.

system_context: |
  Kafka-backed Pub/Sub router in Go.
  Admin API manages Topics + Subscriptions.
  Ingest API accepts generic events and routes to Kafka.
  Delivery workers consume Kafka and push to subscriber gRPC targets with retries + DLQ.
  Prometheus + Grafana + docker-compose for local observability.

dependencies:
  - README.md
  - docs/arch/ARCH.md
  - docs/backend/INGEST_DELIVERY.md
  - docs/ops/METRICS.md

teammates:
  - role: "Load Test Engineer"
    doc: docs/perf/LOADTEST_PLAN.md
    focus: |
      - workload models (steady state, burst, soak)
      - publish-only and end-to-end test scenarios
      - scenario matrix: events/sec, payload sizes, subscription counts, target failure rate
      - exact commands for smoke vs sustained vs soak tests
      - pass/fail thresholds and safety guardrails for local runs

  - role: "Perf Telemetry Engineer"
    doc: docs/perf/TELEMETRY_PLAN.md
    focus: |
      - performance-critical metrics and label conventions
      - Grafana dashboards for throughput, latency, error rate, and Kafka lag
      - Go runtime telemetry (goroutines, GC, memory)
      - pprof plan: when and how to capture CPU/heap profiles during runs

  - role: "Performance Analyst"
    doc: docs/perf/PERF_PLAN.md
    focus: |
      - SLOs and success criteria (latency, throughput, error rate targets)
      - baseline measurement plan
      - regression detection criteria
      - go/no-go criteria for optimization work
```

### Resulting spawn prompt for "Load Test Engineer" (what gets sent to the teammate)

```
ROLE: Load Test Engineer

Write ONLY to: docs/perf/LOADTEST_PLAN.md
Do NOT edit any other files.
Do NOT write code unless and until the plan is approved.

## System Context
Kafka-backed Pub/Sub router in Go.
Admin API manages Topics + Subscriptions.
Ingest API accepts generic events and routes to Kafka.
Delivery workers consume Kafka and push to subscriber gRPC targets with retries + DLQ.
Prometheus + Grafana + docker-compose for local observability.

## Goal for this phase
Define how we measure, generate load, and evaluate performance
before any implementation or optimization work begins.

## Your Focus Areas
- workload models (steady state, burst, soak)
- publish-only and end-to-end test scenarios
- scenario matrix: events/sec, payload sizes, subscription counts, target failure rate
- exact commands for smoke vs sustained vs soak tests
- pass/fail thresholds and safety guardrails for local runs

## Inputs — read all that exist, treat as truth
- README.md
- docs/arch/ARCH.md
- docs/backend/INGEST_DELIVERY.md
- docs/ops/METRICS.md

---

## Master Plan Template

You must follow this template VERBATIM. Fill in every section for your role.
Sections marked "as relevant to your role" should be scoped to your focus area.
Sections not applicable should say "N/A — not in scope for this role" with a brief reason.

[... full TEMPLATE.md contents embedded here ...]

---

## IMPORTANT PROCESS RULES
- You are in PLAN MODE. Stay in PLAN MODE until the plan is explicitly approved.
- Follow the template above verbatim — do not skip or rename sections.
- Keep the "Tighten the plan into 4–7 small tasks" section strictly to 4–7 tasks.
- Do not write any code.
- End your document with the exact line: READY FOR APPROVAL
```

---

## Example 2: Auth Phase (different project)

### kickoff.yaml (input)

```yaml
phase: "Auth"

goal: >
  Design the authentication and authorization system before any implementation begins.

system_context: |
  Python/FastAPI monolith with PostgreSQL.
  Currently no auth — all endpoints are open.
  Need JWT-based auth with role-based access control (RBAC).
  Deployed on Railway, frontend is Next.js.

dependencies:
  - README.md
  - docs/api/OPENAPI.yaml

teammates:
  - role: "Auth Architect"
    doc: docs/auth/ARCH_PLAN.md
    focus: |
      - JWT vs session tradeoffs for this stack
      - token storage strategy (httpOnly cookie vs Authorization header)
      - refresh token rotation plan
      - RBAC model: roles, permissions, enforcement layer

  - role: "Backend Auth Engineer"
    doc: docs/auth/BACKEND_PLAN.md
    focus: |
      - FastAPI middleware and dependency injection for auth
      - user/role/permission data model and migrations
      - password hashing, login, logout, refresh endpoints
      - unit + integration test plan

  - role: "Frontend Auth Engineer"
    doc: docs/auth/FRONTEND_PLAN.md
    focus: |
      - Next.js auth state management (context vs library)
      - protected route pattern
      - token storage and refresh handling on the client
      - login/logout UI flow
```
