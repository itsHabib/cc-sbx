---
name: team-code-project
description: Scaffold a complete code project — generates PROJECT.md and phase kickoff YAMLs from a few questions. Analyzes project complexity to decide phases, teammate roles, and focus areas automatically. Run this to go from idea to ready-to-kickoff in seconds.
argument-hint: [optional project description]
disable-model-invocation: false
---

# Team Code Project Skill

Generate a `PROJECT.md` and all phase `kickoff.yaml` files for a code project from a short conversation.

This combines `/init-team-project` + `/new-team-phase` into one intelligent step — the AI decides the phase breakdown and team composition based on the project's complexity.

## Step 1 — Gather project info

If `$ARGUMENTS` contains a project description, use it as context. Otherwise, ask all questions in a **single message**:

1. **Project name** — what is this project called?
2. **What does it do?** — describe the project in 2-3 sentences. What problem does it solve?
3. **Tech stack** — language(s), frameworks, key dependencies.
4. **Constraints** — any hard constraints? (existing systems to integrate with, specific APIs, etc.). Say "none" if not applicable.

That's it — keep it lightweight. You'll infer the rest.

## Step 2 — Analyze and design

From the user's answers, determine:

### Project structure
- Key components / packages / modules
- Non-goals (infer reasonable scope boundaries)

### Phases
Decide how many phases the project needs. Guidelines:

| Project size | Example | Phases |
|-------------|---------|--------|
| Small (1-3 files, simple logic) | CLI utility, single-package library | 1 phase: "build" |
| Medium (4-10 files, 2-3 packages) | CLI tool with config + engine + output | 1-2 phases: "build" or "core" + "integration" |
| Large (10+ files, multiple packages, complex deps) | Full-stack app, multi-service system | 2-4 phases: e.g., "foundation" → "core" → "integration" → "polish" |

Phase design rules:
- Each phase should be independently valuable (produces working, testable code)
- Later phases can `depends_on` earlier phases
- Prefer fewer phases — only split when there's a genuine dependency barrier
- Every phase gets its own `kickoff.yaml`

### Teammates per phase
For each phase, determine teammates (3-5 per phase). Guidelines:
- Split by **ownership boundary** (who owns which files/packages)
- Each teammate should own 2-5 files
- Teammates within a phase work in parallel, so minimize cross-teammate dependencies
- Give each teammate a clear, specific focus with exact file paths

## Step 3 — Present the plan

Print a summary for user review:

```
## Project: <name>

### Phases

**Phase 1: <name>** — <goal>
| # | Role | Focus | Key Files |
|---|------|-------|-----------|
| 1 | <role> | <one-line focus> | <file1>, <file2> |
...

**Phase 2: <name>** (depends on: Phase 1) — <goal>
| # | Role | Focus | Key Files |
|---|------|-------|-----------|
...

### Files to generate
- PROJECT.md
- PROJECT.state.yaml
- docs/<phase-1>/kickoff.yaml
- docs/<phase-2>/kickoff.yaml
...
```

Ask: "Does this look right? I'll generate all files."

## Step 4 — Generate all files

### 4a. Generate PROJECT.md

Write to `PROJECT.md` at the project root:

```markdown
# <Project Name>

> <one-line description>

---

## Problem & Motivation

<inferred from user's description>

---

## Definition of Done

<inferred: what "finished" looks like>

---

## Key Components

<bulleted list of major components with one-line descriptions>

---

## Tech Stack

<from user input>

---

## Non-Goals

<inferred reasonable scope boundaries>

---

## Constraints

<from user input, or "None">

---

## Team

| Role | Focus |
|------|-------|
<all roles across all phases>

---

## Phases

| Phase | Config | Goal |
|-------|--------|------|
| <phase> | docs/<phase>/kickoff.yaml | <goal> |
...

---

## Usage in Phase Planning

This file is the source of truth for all planning phases.
List it as the first dependency in every phase config:

\`\`\`yaml
dependencies:
  - PROJECT.md
\`\`\`
```

### 4b. Generate kickoff.yaml for each phase

For each phase, create `docs/<phase-slug>/kickoff.yaml`:

```yaml
phase: "<phase name>"

goal: >
  <phase goal — one sentence>

system_context: |
  <rich context about the project, architecture, conventions, and any
   information teammates need to write good code. Include:
   - what the project does
   - key architectural decisions
   - file/package layout
   - coding conventions (error handling, testing patterns, etc.)
   - integration points between teammates' code
   - any schemas, APIs, or protocols teammates need to know>

dependencies:
  - PROJECT.md
  <plus any outputs from earlier phases>

plans_dir: docs/<phase-slug>/plans
content_dir: docs/<phase-slug>/content

teammates:
  - role: "<role name>"
    doc: <role-slug>.md
    focus: |
      <detailed focus areas with exact file paths to create,
       what each file should contain, and how it connects
       to other teammates' code. Be specific enough that
       the teammate can write their plan without ambiguity.>

  - role: "<role name>"
    doc: <role-slug>.md
    focus: |
      <...>
```

**System context quality matters.** This is what every teammate reads to understand the project. Include:
- The full picture of what's being built
- Schemas, data models, or API contracts that teammates share
- Coding conventions and patterns to follow
- How the pieces fit together

Create the directory structure for each phase:
```
docs/<phase-slug>/plans/    # created empty, filled by /team-kickoff
docs/<phase-slug>/content/  # created empty, filled by /team-execute
```

### 4c. Generate PROJECT.state.yaml

Write `PROJECT.state.yaml` at the project root. This file powers the `/where-are-we` skill for tracking progress across phases.

```yaml
project: <project-name-slug>
updated: <today's date, YYYY-MM-DD>

tracks:
  main:
    description: "<one-line project description>"
    project_doc: PROJECT.md
    status: pending
    phases:
      <phase-1-slug>:
        config: docs/<phase-1-slug>/kickoff.yaml
        status: next
        depends_on: []
        notes: "<phase 1 goal>"
      <phase-2-slug>:
        config: docs/<phase-2-slug>/kickoff.yaml
        status: pending
        depends_on: [<phase-1-slug>]
        notes: "<phase 2 goal>"
      # ... repeat for all phases
```

Rules:
- The first phase (no dependencies) gets `status: next`
- All other phases get `status: pending`
- `depends_on` mirrors the dependency chain from the kickoff YAMLs
- Use a single `main` track unless the project has clearly independent work streams

## Step 5 — Print next steps

```
All files generated:

  PROJECT.md
  PROJECT.state.yaml
  docs/<phase-1>/kickoff.yaml
  docs/<phase-2>/kickoff.yaml
  ...

Next steps:

  1. Review the generated files and edit anything you want to adjust.

  2. Kick off the first phase:
       /team-kickoff docs/<phase-1>/kickoff.yaml

  3. After plans are approved, execute:
       /team-execute docs/<phase-1>/kickoff.yaml

  4. Then move to the next phase:
       /team-kickoff docs/<phase-2>/kickoff.yaml

Tip: Each phase's kickoff.yaml is fully self-contained.
     Edit teammates, focus areas, or system_context as needed.
```
