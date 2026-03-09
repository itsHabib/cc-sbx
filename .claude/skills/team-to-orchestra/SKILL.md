---
name: team-to-orchestra
description: Convert PROJECT.md + kickoff YAMLs into an orchestra.yaml for the agent-orchestra CLI
argument-hint: [PROJECT.md path] [kickoff.yaml paths...]
disable-model-invocation: false
---

# Team-to-Orchestra Converter

Convert a set of `/team-kickoff` phase configs (PROJECT.md + kickoff YAMLs) into an `orchestra.yaml`
file compatible with the `agent-orchestra` Go CLI (`orchestra run`).

## Mapping Rules

The team-kickoff format and orchestra.yaml format are structurally similar but use different schemas.
Here is the mapping:

| Team Kickoff | Orchestra.yaml |
|-------------|---------------|
| Project name (from PROJECT.md `# title`) | `name:` |
| Each kickoff.yaml = 1 phase | Each phase = 1 `team` |
| `phase:` field | `team.name:` (slugified) |
| `goal:` field | First task `summary:` |
| `system_context:` field | `team.context:` |
| `teammates[].role` | `team.members[].role` |
| `teammates[].focus` | `team.members[].focus` |
| `dependencies:` (kickoff yaml paths) | `team.depends_on:` (mapped to team names) |

### Team Lead Role

Each team's lead role is derived from the phase goal. Use a descriptive role like:
`"Phase lead — <phase name>"`. Set model to `opus` for planning-heavy phases (Phase 1, 2)
and `sonnet` for execution-heavy phases (3+), unless the user overrides.

### Task Generation

For each phase, generate tasks from the teammates' focus areas:
- Each teammate's focus becomes a task
- `summary`: teammate role + one-line description
- `details`: the full `focus` field content
- `deliverables`: extract file paths mentioned in the focus (look for backtick-quoted paths)
- `verify`: `go build ./... && go test ./...` for Go phases, `npm run build` for frontend phases,
  or combine both for full-stack phases

### Dependency Mapping

Kickoff YAMLs reference dependencies as file paths (e.g., `docs/data-engine/kickoff.yaml`).
Map these to orchestra team names by:
1. Extract the directory name from the path (e.g., `data-engine` from `docs/data-engine/kickoff.yaml`)
2. That directory name IS the team name in orchestra.yaml
3. Skip `PROJECT.md` from dependencies (it's context, not a team)

## Step 1 — Identify inputs

If `<user_argument>` contains file paths, use those. Otherwise, search for:
1. `PROJECT.md` in the current directory or common locations
2. All `kickoff.yaml` files under `docs/*/kickoff.yaml`

Read all identified files. If no files are found, ask the user for paths.

## Step 2 — Parse and validate

For each kickoff.yaml, extract:
- `phase` (name)
- `goal`
- `system_context`
- `dependencies` (list of paths)
- `teammates` (list of role + focus)

Validate:
- All dependency paths reference existing kickoff files (or PROJECT.md)
- No circular dependencies
- Each phase has at least one teammate

## Step 3 — Present the mapping

Show the user a summary:

```
## Orchestra Mapping

Project: <name>
Teams: <count>

| # | Team (from phase) | Members | Depends On | Tasks |
|---|-------------------|---------|------------|-------|
| 1 | data-engine       | 4       | —          | 4     |
| 2 | full-stack        | 3       | data-engine| 3     |
...

Output: <project-root>/orchestra.yaml
```

Ask: "Does this mapping look right? I'll generate the orchestra.yaml."

## Step 4 — Generate orchestra.yaml

Write the orchestra.yaml file to the project root (same directory as PROJECT.md).

Use this template structure:

```yaml
name: "<project-name>"

defaults:
  model: "sonnet"
  max_turns: 200
  permission_mode: "acceptEdits"
  timeout_minutes: 45

teams:
  - name: "<phase-slug>"
    lead:
      role: "Phase lead — <phase name>"
      model: "opus"  # or sonnet for later phases
    members:
      - role: "<teammate role>"
        focus: "<teammate focus summary — first line only>"
      # ... more members
    context: |
      <system_context from kickoff.yaml — included verbatim>
    tasks:
      - summary: "<teammate role>: <brief description>"
        details: |
          <full teammate focus content>
        deliverables:
          - "<extracted file paths>"
        verify: "go build ./... && go test ./..."
      # ... more tasks (one per teammate)
    depends_on:
      - "<upstream team name>"
      # ... more dependencies

  # ... more teams
```

### Generation rules:
- Team names are the kickoff directory slugs (e.g., `data-engine`, `full-stack`, `security-caching`)
- Include the full `system_context` in each team's `context` field (teams need it for autonomous work)
- Each teammate becomes BOTH a member AND a task
- Member `focus` is truncated to first line (summary). Task `details` gets the full content.
- Extract file paths from focus text: scan for backtick-quoted paths like \`internal/db/schema.sql\`
- Verify commands:
  - Phases with only Go code: `go build ./... && go test ./...`
  - Phases with only frontend: `cd frontend && npm run build`
  - Phases with both: `go build ./... && go test ./... && cd frontend && npm run build`
  - Phases with Docker: `docker build -t test .`
  - Add phase-specific verify if the kickoff mentions specific test commands
- For the first 2 phases, use model `opus` for the lead (architectural decisions matter more).
  For phases 3+, use `sonnet` (more execution-focused).

## Step 5 — Print next steps

```
Generated: <path>/orchestra.yaml

Validate:   cd <project-root> && orchestra validate
Dry run:    orchestra run --dry-run
Execute:    orchestra run
Phase-only: orchestra run --phase data-engine
```
