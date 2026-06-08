# v0.8.15 Design Alignment

## 问题修复

- Restores interactive confirmation for single literal `feedback --pattern <id>` runs, while keeping batch feedback gated by `--force` or `--dry-run`.
- Resolves registered project roots when project commands are run from subdirectories, so project state and `.agents/skills` updates stay anchored to the project root.
- Allows `repo sync --all` to sync disabled repositories as documented.
- Keeps `repo sync --json` output machine-readable by suppressing local git progress output.
- Updates outdated feedback guard guidance to use `skill-hub apply --pattern <id>`.

## 测试与验证

- Adds e2e coverage for literal feedback confirmation, subdirectory project-root routing, and disabled repository sync.
- Validated with `make test`, `make build`, and the full Python e2e suite.
