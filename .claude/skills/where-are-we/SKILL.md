---
name: where-are-we
description: Show project progress — what's done, what's next, and what to kick off. Reads PROJECT.state.yaml to give a status report across all work streams.
argument-hint: "[optional: track name to focus on]"
disable-model-invocation: false
---

# Where Are We Skill

Read the project state file and present a clear status report showing what's been done and what's next.

## Step 1 — Find and read the state file

Look for `PROJECT.state.yaml` in the current working directory (project root). If it doesn't exist, tell the user:

```
No PROJECT.state.yaml found in this project.

To create one, either:
  - Run /team-code-project to scaffold a new project (generates it automatically)
  - Create one manually — see the format at ~/.claude/skills/where-are-we/SKILL.md
```

If it exists, read it.

## Step 2 — Parse and summarize

If `$ARGUMENTS` names a specific track (e.g., "compute-spawning"), focus on that track only. Otherwise, show all tracks.

### For each track, show:

```
## <Track Name> — <status emoji> <status>
<description>

| Phase | Plan | Execute | Config | Notes |
|-------|------|---------|--------|-------|
| <phase> | <plan_status> | <execute_status> | <config path> | <notes> |
...
```

**Plan status** tracks `/team-kickoff` (writing plan docs):
- not_started, in_progress, done

**Execute status** tracks `/team-execute` or `/team-code-execute` (writing deliverables/code):
- not_started, in_progress, done

**Overall phase status** is derived:
- `next` → plan not started, dependencies met (suggest `/team-kickoff`)
- `planning` → plan in progress
- `planned` → plans done, execute not started (suggest `/team-execute` or `/team-code-execute`)
- `executing` → execute in progress
- `done` → both plan and execute done
- `pending` → dependencies not met
- `blocked` → explicitly blocked

Status emoji mapping:
- done → checkmark
- planned → star (plans ready, execute is the next action)
- next → circle-arrow (ready to kick off planning)
- planning / executing → arrow (in progress)
- pending → circle
- blocked → x

### Then show the "What's Next" section:

Collect ALL phases across ALL tracks that are actionable. Two types of actionable:
1. **Ready to plan** — `plan_status: not_started` and dependencies met
2. **Ready to execute** — `plan_status: done` and `execute_status: not_started`

```
## What's Next

Ready to execute (plans approved):

1. **<track> / <phase>** — <notes>
   Execute: `/team-execute <config path>` or `/team-code-execute <config path>`

Ready to plan (dependencies met):

1. **<track> / <phase>** — <notes>
   Kick off: `/team-kickoff <config path>`
```

Prioritize "ready to execute" over "ready to plan" — finish what's already planned first.

### Then show "Coming After":

Collect phases where `status: pending` and show what they're blocked on.

```
## Coming After

- **<track> / <phase>** — blocked on: <depends_on list>
```

## Step 3 — Suggest next action

End with a direct suggestion:

```
Suggested next action: /team-kickoff <path to highest-priority next phase config>
```

Priority order for suggestions:
1. Phases that unblock the most downstream work
2. Phases in tracks that are `in_progress` (finish what you started)
3. Standalone tasks

## State file format reference

```yaml
project: <name>
updated: <date>

tracks:
  <track-slug>:
    description: "<what this work stream is>"
    project_doc: <path to PROJECT.md>  # optional
    status: done | in_progress | pending
    phases:
      <phase-slug>:
        config: <path to kickoff.yaml>   # for team phases
        plan_status: not_started | in_progress | done
        execute_status: not_started | in_progress | done
        completed: <date>                # when both plan + execute are done
        commit: <short sha>              # when done
        depends_on: [<phase-slugs>]      # within same track
        notes: "<what this phase does>"
```

**Status lifecycle per phase:**
1. `plan_status: not_started` → `/team-kickoff` starts → `plan_status: in_progress`
2. Plans written + approved → `plan_status: done`
3. `/team-execute` or `/team-code-execute` starts → `execute_status: in_progress`
4. Deliverables written → `execute_status: done`, set `completed` date
