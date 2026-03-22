---
name: new-team-phase
description: Interactively scaffold a new team-kickoff phase config YAML. Use when the user wants to plan a new phase but doesn't have a config file yet.
disable-model-invocation: false
---

# New Phase Scaffold

Help the user create a phase config YAML for `/team-kickoff`.

## Step 1 — Ask these questions in a single message

Ask all at once (don't ask one at a time):

1. **Phase name** — what is this planning phase called? (e.g., "Performance", "Auth", "Backend API")
2. **Goal** — one sentence: what should be defined/decided/designed by the end of this phase?
3. **System description** — briefly describe the system (language, key components, infra). A few sentences is fine.
4. **Existing docs** — are there any existing files in the repo teammates should read first? (e.g., README, architecture docs, existing API specs). List paths or say "none".
   **Tip:** If a `PROJECT.md` exists, suggest it as the first dependency so all teammates share the same project foundation.
5. **Teammates** — list the roles you want (3-5 per phase is the best practice). For each, give:
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

plans_dir: <base>/plans      # where /team-kickoff writes plan docs
content_dir: <base>/content  # where /team-execute writes final deliverables

teammates:
  - role: "<role>"
    doc: <filename>.md        # just the filename — dirs come from plans_dir / content_dir
    focus: |
      <focus bullets>
```

Rules for paths:
- `plans_dir` and `content_dir` are sibling directories under a common base (e.g., `docs/perf/plans` and `docs/perf/content`)
- Each teammate's `doc` is just a filename (e.g., `nutrition-plan.md`), NOT a full path
- `/team-kickoff` writes to `<plans_dir>/<doc>`
- `/team-execute` reads from `<plans_dir>/<doc>` and writes to `<content_dir>/<doc>`
- This keeps plans preserved as a record and deliverables cleanly separated

## Step 3 — Confirm and save

Ask: "Should I save this to `<suggested-path>`?" — suggest `docs/<phase-slug>/kickoff.yaml` as the default location.

If yes, write the file. Then proceed to Step 4.

## Step 4 — Create or update PROJECT.state.yaml

After saving the kickoff YAML, create or update `PROJECT.state.yaml` at the project root so `/where-are-we` can track this phase.

**If `PROJECT.state.yaml` does not exist**, create it:

```yaml
project: <project-name-slug>
updated: <today's date, YYYY-MM-DD>

tracks:
  main:
    description: "<inferred from PROJECT.md or system_context>"
    project_doc: PROJECT.md
    status: pending
    phases:
      <phase-slug>:
        config: <path to kickoff.yaml just written>
        plan_status: not_started
        execute_status: not_started
        depends_on: []
        notes: "<phase goal>"
```

**If `PROJECT.state.yaml` already exists**, add the new phase to the appropriate track:
- Set `plan_status: not_started` and `execute_status: not_started`
- Populate `depends_on` if it depends on existing phases
- Update the `updated:` date

Then print:

```
Config saved. Run this to kick off the team:

  /team-kickoff <path>
```
