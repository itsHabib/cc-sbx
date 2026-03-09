---
name: hackathon-init
description: Scaffold a complete agent hackathon — generates PROJECT.md, phase kickoff YAMLs, and output directories from a few questions. Run this to spin up a new hackathon in seconds.
disable-model-invocation: false
---

# Hackathon Init Skill

Scaffold a full agent hackathon from scratch. Asks the user a few questions, then generates all the files needed to run the hackathon using `/team-kickoff`.

## Step 1 — Ask all questions in a single message

Use `AskUserQuestion` to ask all of these at once (do NOT ask one at a time):

1. **Hackathon name** — what is this hackathon called?
2. **Theme prompt** — the creative brief for teams (e.g., "Build a CLI tool that solves a daily developer frustration")
3. **Language constraint** — Go, Python, any, etc. (default: any)
4. **Number of teams and team size** — how many teams, and how many engineers per team? (default: 2 teams of 2)
5. **Judging criteria and weights** — what are teams scored on? (default: Creativity 25%, Usefulness 25%, Code Quality 25%, Completeness 25%)
6. **Special rules or constraints** — any additional rules? (default: none)
7. **Include commentator?** — a neutral observer who documents the hackathon (default: yes)

## Step 2 — Compute derived values

From the user's answers, derive:

- **Team names**: use Greek letters in order — Alpha, Beta, Gamma, Delta, Epsilon, Zeta, Eta, Theta, Iota, Kappa, Lambda, Mu — for however many teams.
- **Team slugs**: lowercase team names — `team-alpha`, `team-beta`, etc.
- **Engineer roles**: `eng-1` through `eng-N` where N = (num_teams × team_size). Engineers are assigned to teams sequentially: eng-1 and eng-2 → Team Alpha, eng-3 and eng-4 → Team Beta, etc.
- **Partner assignments**: within each team, all engineers are partners. For team size 2, each engineer has one partner. For team size 3+, list all teammates.

## Step 3 — Generate all files

Generate all files below in one shot. Confirm the output path with the user before writing (default: current directory).

### 3a. `PROJECT.md`

```markdown
# <Hackathon Name>

## What This Is

A mini hackathon run entirely by AI agents. <num_teams> teams of <team_size> engineers compete to build the best project. <If commentator: "A commentator observes and documents everything."> Judges score the results.

## Theme

> <theme prompt — the user's creative brief, verbatim>

## Format

- <total engineers> software engineers, paired into <num_teams> teams of <team_size>
<If commentator: "- 1 commentator who observes and documents progress">
- 3 phases: Ideation → Build → Judging
- All communication between partners happens via DM and must be logged

## Teams

| Team | Members | Output Directory |
|------|---------|-----------------|
<For each team: | Team <Name> | <eng-X, eng-Y, ...> | output/team-<slug>/ |>

## Rules

1. Only communicate with your teammates. Do not share ideas with other teams.
2. All code goes in your team's output directory.
3. The tool must actually run — no vaporware.
4. Maintain a `log.md` in your team's output directory. Log every significant decision, disagreement, pivot, and idea you discuss. This is your team's story.
5. Include a README.md with install and usage instructions.
<Any special rules the user specified, numbered continuing from 6>

## Judging Criteria

| Criterion | Weight | Description |
|-----------|--------|-------------|
<For each criterion from user input or defaults>

## Communication Trail

Every phase produces persistent artifacts:
<For each team: "- **Team logs** (`output/team-<slug>/log.md`) — the team logs their decisions, debates, and pivots as they go">
<If commentator: "- **Commentator recaps** (`output/recap.md`) — the commentator writes periodic updates covering all teams">
- **Final scorecard** (`output/scorecard.md`) — judges produce a detailed scoring breakdown

## Constraints

<Language constraint, if any>
<Any special constraints from user input>
- Must include install/run instructions

## Phases

| Phase | Config | What Happens |
|-------|--------|-------------|
| 1. Ideation | docs/phase-1-ideation/kickoff.yaml | Teams brainstorm and produce a pitch |
| 2. Build | docs/phase-2-build/kickoff.yaml | Teams build their project |
| 3. Judging | docs/phase-3-judging/kickoff.yaml | Judges score all teams |
```

### 3b. `docs/phase-1-ideation/kickoff.yaml`

```yaml
phase: "ideation"
goal: >
  Each team brainstorms and agrees on a project idea matching the theme,
  then produces a pitch document. <If commentator: "The commentator observes and recaps.">

system_context: |
  HACKATHON CONTEXT:
  You are participating in a mini hackathon. <num_teams> teams of <team_size>
  engineers are competing. <If commentator: "A commentator is observing everything.">

  THEME: <theme prompt>

  RULES:
  - Only communicate with your teammates. Do not look at or coordinate
    with other teams.
  <language constraint if any>
  - It must actually work when built — scope accordingly.
  - Log every significant decision, disagreement, and pivot to your
    team's log.md file. This is your team's story — be honest and
    detailed about your thought process.
  <any special rules>

  JUDGING CRITERIA (so you know what matters):
  <For each criterion: "- <Name> (<Weight>) — <Description>">

  See PROJECT.md for full rules and structure.

dependencies:
  - PROJECT.md

plans_dir: docs/phase-1-ideation/plans
content_dir: output

teammates:
  <For each engineer, generate a teammate block:>
  - role: "eng-<N>"
    doc: output/team-<slug>/PITCH.md
    focus: |
      You are eng-<N> on Team <Name>. Your <partner/teammates>: <list other eng roles on this team>.
      You are competing against <list other team names>.

      Your goal this phase: agree on a project idea with your
      <partner/teammates> and produce a pitch document.

      DM <partner names> to brainstorm. Riff on ideas, debate, push back
      on each other. When you've agreed, write PITCH.md covering:
      what the project does, why it's useful, and your rough approach.

      LOGGING (CRITICAL):
      Maintain output/team-<slug>/log.md as a running transcript.
      After EVERY DM exchange with your <partner/teammates>, immediately append
      an entry capturing: what was discussed, what was proposed,
      what was decided, any pushback or disagreements. Do this
      BEFORE moving on. The log should read like a conversation
      history, not a summary written after the fact. Include your
      own reasoning and reactions, not just decisions.

  <If commentator, add:>
  - role: "commentator"
    doc: output/recap.md
    focus: |
      You are the hackathon commentator. You observe, you do not
      participate.

      Periodically check the task list to see what all teams are
      working on. Read their log.md files when they update them.
      Write your observations to output/recap.md.

      Cover things like:
      - What ideas each team is exploring
      - How the pairs are collaborating (or not)
      - Any interesting dynamics — who's leading, who's pushing back
      - How the teams' approaches compare
      - The overall energy and momentum

      Do NOT DM the engineers. Do NOT suggest ideas. You are a
      neutral observer documenting what happens.
```

### 3c. `docs/phase-2-build/kickoff.yaml`

```yaml
phase: "build"
goal: >
  Each team builds the project they pitched in Phase 1.
  <If commentator: "The commentator tracks progress and documents the build process.">

system_context: |
  HACKATHON CONTEXT:
  You are in the BUILD phase of a mini hackathon. <num_teams> teams of
  <team_size> engineers are competing. Each team has already pitched
  their idea in Phase 1.

  RULES:
  - Only communicate with your teammates. Do not coordinate with
    other teams.
  - All code goes in your team's output directory.
  - The project must actually run.
  - Keep logging decisions and progress to your team's log.md.
  - Include a README.md with install and usage instructions.
  <any special rules>

  JUDGING CRITERIA:
  <For each criterion: "- <Name> (<Weight>) — <Description>">

  See PROJECT.md for full rules and structure.

dependencies:
  - PROJECT.md
  <For each team: "- output/team-<slug>/PITCH.md">

plans_dir: docs/phase-2-build/plans
content_dir: output

teammates:
  <For each engineer, generate a teammate block:>
  - role: "eng-<N>"
    doc: output/team-<slug>/README.md
    focus: |
      You are eng-<N> on Team <Name>. Your <partner/teammates>: <list other eng roles on this team>.
      You are competing against <list other team names>.

      Read your team's PITCH.md from Phase 1 — that's what
      you're building. Coordinate with <partner names> on how to divide
      the work. Build the project in output/team-<slug>/.

      Write working code. Include a README.md with clear install
      and usage instructions. Make sure it actually runs.

      LOGGING (CRITICAL):
      Continue the running transcript in output/team-<slug>/log.md.
      After EVERY DM exchange with your <partner/teammates>, immediately append
      an entry capturing: what was discussed, what was decided,
      any technical debates or tradeoffs. Log problems you hit and
      how you solved them in real time, not after the fact. Include
      your own reasoning, not just outcomes.

  <If commentator, add:>
  - role: "commentator"
    doc: output/recap.md
    focus: |
      You are the hackathon commentator. You observe, you do not
      participate. This is the BUILD phase.

      Periodically check the task list and read the teams' log.md
      files. Append your build-phase observations to output/recap.md.

      Cover things like:
      - How each team divided the work
      - What technical approaches they chose
      - Who's making faster progress
      - Any struggles, pivots, or breakthroughs
      - How the pairs are communicating and coordinating

      Do NOT DM the engineers. Do NOT help with code.
      You are a neutral observer.
```

### 3d. `docs/phase-3-judging/kickoff.yaml`

```yaml
phase: "judging"
goal: >
  Judges review all teams' projects and score them against the
  hackathon criteria. <If commentator: "The commentator writes the final recap.">

system_context: |
  HACKATHON CONTEXT:
  You are judging a mini hackathon. <num_teams> teams of <team_size>
  engineers each built a project. Your job is to evaluate their work
  fairly and thoroughly.

  Each team's output is in their directory:
  <For each team: "- Team <Name>: output/team-<slug>/">

  Each directory contains:
  - PITCH.md — what they said they'd build
  - README.md — install and usage instructions
  - log.md — their decision trail and process
  - Source code files

  JUDGING CRITERIA:
  <For each criterion: "- <Name> (<Weight>) — <Description>">

  See PROJECT.md for full rules and structure.

dependencies:
  - PROJECT.md
  <For each team:>
  - output/team-<slug>/PITCH.md
  - output/team-<slug>/README.md
  - output/team-<slug>/log.md

plans_dir: docs/phase-3-judging/plans
content_dir: output

teammates:
  - role: "judge-technical"
    doc: output/scorecard.md
    focus: |
      You are the technical judge. Focus on <technical criteria from the judging list — e.g., Code Quality and Completeness>.

      For each team:
      1. Read their PITCH.md to understand what they promised
      2. Read their source code thoroughly
      3. Read their README.md — are the instructions clear?
      4. Assess: does the code actually do what the pitch claims?
      5. Evaluate code quality — structure, readability, error
         handling, edge cases

      Score each team 1-10 on your assigned criteria.
      Write detailed justifications. Be specific — cite actual
      code, actual gaps, actual strengths.

  - role: "judge-product"
    doc: output/scorecard.md
    focus: |
      You are the product judge. Focus on <product criteria from the judging list — e.g., Creativity and Usefulness>.

      For each team:
      1. Read their PITCH.md — is the idea interesting?
      2. Read their README.md — does the UX make sense?
      3. Read their log.md — how did they arrive at this idea?
         Did they consider alternatives?
      4. Evaluate against each of your assigned criteria.

      Score each team 1-10 on your assigned criteria.
      Write detailed justifications. Compare the teams directly
      where it helps illustrate differences.

  <If commentator, add:>
  - role: "commentator"
    doc: output/recap.md
    focus: |
      You are the hackathon commentator. This is the final phase.

      Read everything — all teams' pitches, code, logs, and the
      judges' scores. Write the final section of output/recap.md.

      Tell the full story:
      - Recap what each team built
      - Highlight the best moments from the logs
      - Summarize the judges' verdicts
      - Declare the winner
      - Reflect on how the teams differed in approach,
        collaboration style, and execution

      This is the definitive record of the hackathon. Make it
      worth reading.
```

### 3e. Output directories

Create empty directories for each team:

```
output/team-alpha/
output/team-beta/
... (one per team)
```

Use `mkdir -p` via Bash to create them.

## Step 4 — Write all files and print next steps

After writing all files, print:

```
Hackathon scaffolded! Here's what was created:

  PROJECT.md                           — hackathon source of truth
  docs/phase-1-ideation/kickoff.yaml   — brainstorm + pitch phase
  docs/phase-2-build/kickoff.yaml      — build phase
  docs/phase-3-judging/kickoff.yaml    — judging phase
  output/team-<slug>/                  — output directory for each team

Next steps:

  1. Review PROJECT.md to make sure everything looks right.

  2. Start the hackathon:
       /team-kickoff docs/phase-1-ideation/kickoff.yaml

  3. When Phase 1 completes, run Phase 2:
       /team-kickoff docs/phase-2-build/kickoff.yaml

  4. When Phase 2 completes, run judging:
       /team-kickoff docs/phase-3-judging/kickoff.yaml
```
