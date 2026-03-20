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

| Phase | Status | Config | Notes |
|-------|--------|--------|-------|
| <phase> | <status> | <config path> | <notes> |
...
```

Status emoji mapping:
- done → checkmark
- in_progress → arrow
- next → star  (this is the actionable one)
- pending → circle
- blocked → x

### Then show the "What's Next" section:

Collect ALL phases across ALL tracks where `status: next`. These are the actionable items — their dependencies are met and they're ready to start.

```
## What's Next

Ready to start (all dependencies met):

1. **<track> / <phase>** — <notes>
   Kick off: `/team-kickoff <config path>`

2. **<track> / <phase>** — <notes>
   This is a standalone task — build directly, no kickoff needed.
   Plan: <plan path>
```

For phases with a `config` field, suggest `/team-kickoff <config>`.
For phases with a `plan` field (standalone), suggest reading the plan and building directly.

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
        plan: <path to plan doc>         # for standalone phases
        status: done | in_progress | next | pending | blocked
        completed: <date>                # when done
        commit: <short sha>              # when done
        depends_on: [<phase-slugs>]      # within same track
        notes: "<what this phase does>"
```
