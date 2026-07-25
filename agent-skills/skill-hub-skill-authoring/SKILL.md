---
name: skill-hub-skill-authoring
description: "Primary workflow when creating or updating skills in an environment where skill-hub is installed, available, or used to manage skills. Use this before the generic skill-creator for skill-hub managed skills. Covers creating, repairing, validating, registering, importing, archiving, and otherwise managing reusable local skills with skill-hub; authoring Chinese-by-default SKILL.md files; declaring the matching formatter; maintaining .agents/skills content; syncing local project skill edits back to the default skill repository; and previewing remote push without publishing unless explicitly requested."
metadata:
  author: skill-hub Team
  tags: skill-hub,skill-authoring,skills,validation,feedback
  version: 1.0.6
---

# Skill Hub Skill Authoring

Use this skill when the task is to create or manage reusable skills themselves.

This is an authoring and maintenance workflow. If the user only wants to consume an existing managed skill in an application project, use `skill-hub-project-usage` instead.

## Scheduling Priority

When `skill-hub` is installed, available on `PATH`, or the current project/repository uses skill-hub conventions such as `.agents/skills`, this skill is the primary workflow for creating or updating skills.

Use the generic `skill-creator` only as secondary design guidance. If `skill-creator` and this skill differ, follow this skill for storage location, lifecycle commands, validation, feedback, default language, formatter declaration, and remote-push rules.

For skill-hub managed skills, always anchor creation on:

```bash
skill-hub create <skill-id>
skill-hub validate --pattern <skill-id> --links
skill-hub feedback --pattern <skill-id> --dry-run
```

## Authoring Model

- Local project skill content lives under `.agents/skills/<skill-id>/`.
- The required entry point is `.agents/skills/<skill-id>/SKILL.md`.
- The default skill repository under `~/.skill-hub/repositories/<default>/` is the archive source for reusable skills.
- `feedback` copies project-local skill content back to the local default skill repository.
- `import <skills-dir> --archive --archive-only --force` is the archive path for existing skill directories outside the standard project workspace, including release-bundled `agent-skills/*`, when they should not be registered in the current project state.
- `repo rebuild-index [repo]` repairs stale `registry.json` indexes; do not use manual directory copies as an archive workflow.
- `push` publishes local repository changes to a remote and must only run after explicit user approval.
- Do not create target-specific branches or write `preferred_target`.

## Start Or Inspect A Skill Workspace

Check the current project state:

```bash
skill-hub status --json
```

Initialize the project if needed:

```bash
skill-hub init
```

Before creating a skill, check whether the ID already exists:

```bash
skill-hub list
skill-hub search <keyword>
```

Prefer stable lowercase IDs such as `go-runtime-patterns` or `skill-hub-project-usage`.

## Resolve Authoritative Content Before Updating

When a managed skill copy differs from the local repository, do not choose a source by file modification time alone.

1. Treat `~/.skill-hub/repositories/<default>/skills/<skill-id>/` and its Git history as the reusable-skill source of truth.
2. Compare project, global-agent, and repository copies with directory hashes; use `status --global` for managed global copies.
3. Inspect `git log --follow -- skills/<skill-id>/SKILL.md` and the matching `registry.json` entry before repairing a discrepancy.
4. Keep an explicit skill version (`version` or `metadata.version`) once it exists. A missing version is an integrity defect, not a reason to fall back to `1.0.0`.
5. Preserve the repository copy's current content when refreshing project or global copies; do not feed a stale copy back through `feedback`.

If a repository commit changes the skill body but accidentally drops metadata, restore the metadata from the last consistent commit/index, then refresh the derived copies with `apply` or `apply --global`.

## Create A New Skill

Create the project-local skill directory:

```bash
skill-hub create <skill-id>
```

Then edit `.agents/skills/<skill-id>/SKILL.md`.

Keep `SKILL.md` focused:

- Frontmatter must include `name` and `description`.
- Make `description` clear enough to trigger the skill for the right task.
- New skills must use Chinese by default for human-facing definitions, headings, workflow instructions, variables, examples, and final guidance unless the user explicitly requests another language.
- Keep technical identifiers, command names, code symbols, API names, file paths, and upstream product names in their canonical form.
- Put only essential workflow instructions in the body.
- Use `references/` for detailed docs that should be loaded only when needed.
- Use `scripts/` only for deterministic repeated operations.
- Use `assets/` for reusable output files or templates.

If the skill needs user-provided values at `use` time, add `variables` in frontmatter. If not, leaving variables absent is valid and `use` will print that the skill has no configurable variables.

Every new skill must include a concise formatter section in the body:

```markdown
## Formatter

- Markdown/YAML: run `skill-hub validate --pattern <skill-id> --links` before feedback.
- Scripts/code: use the formatter already configured by the target project or repository. If none exists, state the expected formatter explicitly.
- Run formatting before `skill-hub feedback --pattern <skill-id> --force`.
```

Choose formatter commands by content:

- Markdown/YAML-only skills: use stable Markdown formatting plus `skill-hub validate --pattern <skill-id> --links`.
- Go scripts or examples: use `gofmt -w <files>`.
- Python scripts: use the repository formatter such as `ruff format <files>` or `black <files>`.
- JavaScript or TypeScript examples: use the repository script such as `npm run format` or the configured `prettier`.
- Shell scripts: use the repository formatter if present; otherwise keep POSIX/Bash style consistent and run the relevant shell syntax check when available.

Do not invent a formatter that the project cannot run. If the repository has no formatter for a file type, record the manual formatting expectation in the skill.

## Register Or Import Existing Skills

Register an existing project-local skill without overwriting files:

```bash
skill-hub register <skill-id>
```

Import skills from an existing directory when migrating content:

```bash
skill-hub import <skills-dir>
skill-hub import <skills-dir> --archive --force
skill-hub import <skills-dir> --archive --archive-only --force
```

Use `--skip-validate` only when intentionally staging invalid content for later repair.

## Validate And Repair

Validate frontmatter and local links:

```bash
skill-hub validate --pattern <skill-id>
skill-hub validate --pattern <skill-id> --links
```

Preview project status:

```bash
skill-hub status --pattern <skill-id>
skill-hub status --pattern <skill-id> --json
```

Use automatic frontmatter repair only when the user accepts file edits:

```bash
skill-hub validate --pattern <skill-id> --fix
```

Run path and duplicate checks when reorganizing skills:

```bash
skill-hub lint --paths --json
skill-hub dedupe --json
```

Repair duplicate non-canonical copies only after a dry run:

```bash
skill-hub sync-copies --canonical .agents/skills --scope . --dry-run
skill-hub sync-copies --canonical .agents/skills --scope .
```

## Archive To The Local Skill Repository

Use the correct archive entry point for the source location:

- Project workspace `.agents/skills/<skill-id>`: use `feedback`.
- Existing or release-bundled skill directory such as `agent-skills/<skill-id>`: use `import <skills-dir> --archive --archive-only --force`.
- Stale repository index after abnormal local edits: use `repo rebuild-index [repo]`.

Preview the archive diff:

```bash
skill-hub feedback --pattern <skill-id> --dry-run
```

Archive the skill to the local default repository:

```bash
skill-hub feedback --pattern <skill-id> --force
```

For many skills:

```bash
skill-hub feedback --all --force --json
```

For bundled or batch directories:

```bash
skill-hub import agent-skills --archive --archive-only --force
skill-hub repo rebuild-index
```

After feedback, confirm the project and repository copies are synced:

```bash
skill-hub status --pattern <skill-id> --json
```

The archive guard rejects updates that drop an existing explicit version, required frontmatter, a first- or second-level section, or a resource under `references/`, `scripts/`, `assets/`, or `agents/`. `--force` does not bypass this integrity protection. Retired metadata with no consumer may be removed through a normal update. If an intentional removal is required, first record and review the canonical repository change rather than using an incomplete project copy as the source.

`repo rebuild-index` can rewrite metadata for many skills. Review `git -C ~/.skill-hub/repositories/<default> diff -- registry.json` after rebuilding; keep the repair scoped to the intended skills instead of accepting unrelated index churn.

## Publish Only When Explicitly Requested

Preview local repository changes:

```bash
skill-hub push --dry-run --json
```

Only publish when the user explicitly requests remote publication:

```bash
skill-hub push --message "update skills"
```

In `serve` mode, remote push requires the service-side `secretKey`:

```bash
SKILL_HUB_SERVICE_SECRET_KEY=<secretKey> skill-hub push
```

Do not treat `feedback`, `pull`, or `repo sync` as remote publication.

## Safety Rules

- Never run `push` automatically.
- Preserve existing user-authored skill content.
- Validate before feedback and after repair.
- Preserve explicit versions and core resources during every update; use Git history and the registry entry to repair drift.
- Keep reusable skill content under `.agents/skills/<skill-id>/`.
- Do not write target-specific state or rely on compatibility filtering.
- If non-push commands return an old read-only serve error, update or restart the running `serve` instance instead of changing the workflow.
