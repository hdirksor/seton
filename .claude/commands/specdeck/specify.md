Before doing anything else: read `specdeck.yml`. If the `skills_version` field is missing or less than 2, stop and tell the user: "Your specdeck skills are out of date — run `specdeck link` to update them, then retry."

Read `specdeck.yml` in the project root to find the `specs_repo` path.

## Step 1 — Orient on the branch

Run `git -C <specs_repo> branch --show-current` and show the user the current branch. Ask: "Continue adding specs on this branch, or create a new branch?"

If the user wants a new branch:
- Propose a kebab-case branch name based on $ARGUMENTS or what the user describes.
- Ask the user to confirm or adjust the name before creating it.
- Check if the working tree is dirty (`git -C <specs_repo> status --porcelain`). If it is, warn the user: "The working tree has uncommitted changes. Create the new branch from this state, or stop and clean up first?" If they want to proceed, continue on the current dirty branch — do not create a new branch.
- If the tree is clean, create and switch to the new branch (`git -C <specs_repo> checkout -b <branch-name>`).
- Create `INTENT.md` in `specs_repo` with the following stub, then ask the user to describe the goal of this branch. Fill in their answer and commit the file.

```markdown
# Intent

## Goal
<!-- What is this branch trying to achieve? -->

## Scope
<!-- What is in scope? What is explicitly out of scope? -->

## Context
<!-- Links to tickets, designs, Slack threads, or other relevant references -->
```

If the user wants to continue on the current branch:
- Check whether `INTENT.md` exists in `specs_repo`. If it does, read it — it describes the goal and scope of this branch's work.
- If it does not exist, ask the user if they want to create one before continuing.

## Step 2 — Collect input

Before writing any specs, you must have something concrete to work from. Ask the user:

"What should I base these specs on? Point me to existing code, a design doc, a ticket, or describe the feature — I won't infer details you haven't provided."

Do not proceed until the user provides at least one of: a file or directory path, a document, or a written description. If $ARGUMENTS already supplies this, confirm with the user that it covers what they want before continuing.

## Step 3 — Read existing specs on this branch

Before drafting anything, read the existing spec files in `specs_repo` that are relevant to this feature. If `INTENT.md` exists, use it to orient what this branch is trying to achieve and what is in or out of scope. Do not duplicate or contradict specs that already exist. Note any gaps in existing coverage that this run should fill.

## Step 4 — Generate specs

Analyse the input provided. For existing code, read the relevant files. For design docs or descriptions, work only from what is written.

Write specs that cover:
- What the feature does (behaviour, not implementation)
- Who uses it and when
- Key states and transitions
- Edge cases and constraints

**Conservative rule**: only write a spec if the input provides explicit evidence for it. If the user has not described the visual appearance, do not write visual specs. If error handling is not mentioned, call it out rather than invent it. When in doubt, omit.

After drafting, explicitly list what you are *not* speccing and why — e.g. "Skipped: error states (not described), visual layout (no design provided)." Ask the user if any of those gaps should be filled before committing.

## Step 5 — Write and commit

Write specs as `.yml` files inside `specs_repo/containers/`, in an appropriate subdirectory matching the feature area (e.g. `containers/app/user-login/index.yml`). Each distinct container — screen, panel, or reusable component — gets its own `.yml` file. A parent container references its children via `$ref` entries in its `containers` list rather than inlining them.

Use the container schema. Only populate fields for which the input provides explicit evidence:

```yaml
title: <human-readable name>
description: >
  <optional: one-paragraph summary of purpose>
behavior:
  - <observable behaviour statement>
specs:
  <property>: <value>   # visual or structural constraints
states:
  - title: <state name>
    description: <optional>
    specs:
      <property>: <value>
events:
  - title: <event name>
    description: <optional>
    effects:
      - title: <effect name>
        description: <what changes>
containers:
  - $ref: '<relative path to child container yml>'
```

Stage and commit the new or updated spec files inside `specs_repo` with a short message describing what was specced. All `git add` and `git commit` commands must use `git -C <specs_repo>`.

Then ask the user: "Are you done with spec changes on this branch, or do you plan to add more?"

If done: delete `INTENT.md` from `specs_repo`, stage the deletion, and commit it (using `git -C <specs_repo>`) with the message `chore: remove INTENT.md before merge`. The branch is now ready to merge.

If not done: leave `INTENT.md` in place.
