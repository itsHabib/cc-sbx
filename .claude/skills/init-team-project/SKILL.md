---
name: init-team-project
description: Interactively define a new project from scratch — captures the core idea, architecture, tech stack, non-goals, constraints, and team composition into a PROJECT.md source-of-truth doc. Run this before /new-team-phase or /team-kickoff.
disable-model-invocation: false
---

# Init Project Skill

Help the user create a `PROJECT.md` that serves as the foundational source of truth for all future phase planning.

## Step 1 — Ask all questions in a single message

Ask all at once (do not ask one at a time):

1. **Project name** — what is this project called?
2. **One-line description** — what does it do, in one sentence?
3. **Problem / motivation** — what problem does this solve, and for whom?
4. **Definition of done** — what does a finished, successful project look like? What can users/systems do that they couldn't before?
5. **Key components** — what are the major pieces of the system? (e.g., services, APIs, workers, UIs, data stores). A rough list is fine.
6. **Tech stack** — languages, frameworks, infra, and any must-use tools or platforms.
7. **Non-goals** — what is explicitly out of scope for this project?
8. **Constraints** — any hard constraints? (e.g., must use existing auth system, compliance requirements, can't introduce new infra, etc.). Say "none" if not applicable.
9. **Team** — list the roles/people who will be working on this project (3-5 per phase is the best practice). For each, give:
   - role name (e.g., "Backend Engineer", "Load Test Engineer", "SRE")
   - a short description of what they own or focus on

## Step 2 — Generate PROJECT.md

Once the user answers, produce the following document. Write it to `PROJECT.md` at the project root (confirm path before writing).

---

```markdown
# <Project Name>

> <one-line description>

---

## Problem & Motivation

<problem / motivation>

---

## Definition of Done

<what "finished" looks like — capabilities unlocked, user outcomes, success state>

---

## Key Components

<bulleted list of major system components with a one-line description each>

---

## Tech Stack

<language, frameworks, infra, must-use tools>

---

## Non-Goals

<what is explicitly out of scope>

---

## Constraints

<hard constraints, or "None">

---

## Team

| Role | Focus |
|------|-------|
| <role> | <focus> |
...

---

## Usage in Phase Planning

This file is the source of truth for all planning phases.
List it as the first dependency in every phase config:

\`\`\`yaml
dependencies:
  - PROJECT.md
  - <other docs>
\`\`\`

Every teammate spawned by `/team-kickoff` should read this file before writing their plan.
```

---

## Step 3 — Confirm and save

Ask: "Should I save this to `PROJECT.md`?" (suggest project root as default, but let user override).

If yes, write the file. Then print:

```
PROJECT.md saved. Next steps:

  1. Run /new-team-phase to scaffold your first planning phase.
     (PROJECT.md will be pre-suggested as a dependency.)

  2. Or if you already have a phase config:
       /team-kickoff path/to/config.yaml

Tip: list PROJECT.md as the first dependency in every phase config
so all teammates share the same project foundation.
```
