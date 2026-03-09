---
name: new-team-phase
description: Interactively scaffold a new team-kickoff phase config YAML. Use when the user wants to plan a new phase but doesn't have a config file yet.
disable-model-invocation: true
---

# New Phase Scaffold

Help the user create a phase config YAML for `/team-kickoff`.

## Step 1 — Ask these questions in a single message

Ask all at once (don't ask one at a time):

1. **Phase name** — what is this planning phase called? (e.g., "Performance", "Auth", "Backend API")
2. **Goal** — one sentence: what should be defined/decided/designed by the end of this phase?
3. **System description** — briefly describe the system (language, key components, infra). A few sentences is fine.
4. **Existing docs** — are there any existing files in the repo teammates should read first? (e.g., README, architecture docs, existing API specs). List paths or say "none".
5. **Teammates** — list the roles you want. For each, give:
   - role name
   - what they focus on (a sentence or a few bullets)
   - where their output doc should go (or leave blank to suggest a path)

## Step 2 — Generate the YAML

Once the user answers, produce a ready-to-use YAML file:

```yaml
phase: "<phase name>"

goal: >
  <goal>

system_context: |
  <system description>

dependencies:
  - <path>   # or omit section if none

teammates:
  - role: "<role>"
    doc: <suggested output path based on phase + role>
    focus: |
      <focus bullets>
```

Rules for suggesting doc paths: use `docs/<phase-slug>/<role-slug>.md` lowercased with hyphens (e.g., `docs/perf/load-test-engineer.md`), unless the user specified a path.

## Step 3 — Confirm and save

Ask: "Should I save this to `<suggested-path>`?" — suggest `docs/<phase-slug>/kickoff.yaml` as the default location.

If yes, write the file. Then print:

```
Config saved. Run this to kick off the team:

  /team-kickoff <path>
```
