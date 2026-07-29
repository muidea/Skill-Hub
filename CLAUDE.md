# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`skill-hub` is a Go CLI/server for managing the lifecycle of AI agent skills (Prompts, scripts, workflows). It uses Git repositories to distribute skills to project workspaces (`.agents/skills/`) and to the host's global agent skills directory. See `README.md` for product behavior and `DEVELOPMENT.md` for the long-form architecture reference. Repository-level conventions (commit style, hooks, release process) live in `AGENTS.md` — read it first and follow it.

## Commands

Primary build / test / lint entry points (all routed through the Makefile):

```bash
make build           # build bin/skill-hub with version ldflags
make test            # go test ./... --count 1
make test-pkg PKG=./internal/cli    # one Go package, verbose
make test-coverage   # coverage.out, then `make coverage-html` for HTML
make lint            # gofmt -d, go vet, staticcheck (if installed)
make clean           # remove bin/ and dist/
make install         # copy bin/skill-hub into ~/.local/bin/
make release-all VERSION=1.0.0     # cross-platform tar.gz + sha256 into dist/
```

Useful direct invocations:

```bash
go test -v -run TestApply ./internal/cli           # single test
go test ./internal/modules/kernel/hub/... -v       # one kernel module
go vet ./...
gofmt -d .                                          # format diff only
~/codespace/venv/bin/python3 -m pytest -p no:rerunfailures tests/e2e -c tests/e2e/pytest.ini
```

If the host has read-only `$HOME` caches, redirect Go/XDG caches to `/tmp` (see `AGENTS.md`).

## High-level architecture

The product has two runtime modes that share state. The CLI is the primary user entry point; when a local `serve` instance is reachable, CLI commands delegate to the running service via the bridge in `internal/cli/service_bridge.go`, otherwise they fall back to in-process execution. Both modes read/write the same on-disk state:

- `~/.skill-hub/` — global domain: repo config, local repo clones, default repo, global state/index/cache.
- `<project>/.agents/skills/` — project workspace: the skill content a project actually uses.

Source layout:

- `application/skill-hub/cmd/main.go` — binary entry, just calls `cli.Execute()`.
- `application/skill-validate/` — standalone validator binary entry.
- `internal/cli/` — cobra command set (one file per command group: `apply.go`, `use.go`, `feedback.go`, `repo.go`, etc.) and the `service_bridge.go` that prefers the running service over local execution.
- `internal/modules/kernel/` — service-oriented kernel, one subdirectory per bounded domain. Each kernel module follows the same `biz/`, `pkg/`, `service/` layering (application services expose business operations; `pkg/` holds domain primitives). Submodules: `global`, `hub`, `repository`, `runtime`, `server`, `skill`, `upgrade`, plus the per-project groups `project_apply`, `project_feedback`, `project_inventory`, `project_lifecycle`, `project_state`, `project_status`, `project_use`.
- `internal/modules/blocks/` — cross-cutting infrastructure blocks consumed by kernel modules: `adapter` (target agents: cursor, claude, opencode), `git` (wraps go-git), `httpapi` (HTTP DTOs/business types used by both server and bridge), `hubclient` (talks to the local serve endpoint), `webui` (embedded admin UI).
- `internal/multirepo/`, `internal/state/`, `internal/config/`, `internal/template/`, `internal/engine/`, `internal/utils/`, `internal/testutils/` — supporting subsystems (multi-repo management, project state persistence, viper config, template rendering, skill loading, shared helpers, test fixtures).
- `pkg/spec/` — public data structures (`Skill`, `SkillMetadata`, search types). Treat this as the contract surface for any cross-package skill data.
- `pkg/errors/` — `AppError` with stable `ErrorCode` constants. Use these codes everywhere instead of ad-hoc `errors.New`; CLI rendering and tests rely on them.
- `agent-skills/` — bundled agent workflow skills (not Go code): `skill-hub-workflow`, `skill-hub-project-usage`, `skill-hub-skill-authoring`. They are also shipped inside release tarballs (see `make release-all`).
- `tests/e2e/` — Python pytest scenarios. Each `test_*.py` exercises a CLI workflow end-to-end against a fresh temp workspace; see `tests/e2e/pytest.ini` and `conftest.py` for fixtures.
- `scripts/` — release script (`create-release.sh`) and Go tests for it, plus install/upgrade shell scripts.

### Boundaries to respect

- `create` / `remove` / `validate` only operate on the local project workspace; they must not be service-backed.
- `feedback` only archives into the **default** repository; the outdated-repo guard in `internal/cli/feedback.go` blocks feedback when the workspace skill is older than the repo copy, prompting the user to `apply` first.
- `use` records the source repository; `apply` / `status` always resolve skills through that source.
- `repo *` manages the multi-repo catalog; it is not a substitute for project workspace commands.
- `prune` only cleans dead project entries from `~/.skill-hub/state.json`. It never creates or restores project state.
- New CLI flags use kebab-case (`--dry-run`, `--skip-agent-skills`); exported Go names stay exported-PascalCase, unexported helpers stay camelCase.

### Build-time version injection

`internal/cli/root.go` reads `version`, `commit`, `date` package vars. The Makefile injects them via `-ldflags` from the `LDFLAGS` variable; `make build` defaults to `VERSION=dev` and the current short SHA. To bake a real release version, pass `VERSION=x.y.z` to `make build` or `make release-all`.

## Commit and release rules (from `AGENTS.md`)

- Conventional Commit prefixes only: `feat:`, `fix:`, `docs:`, `test:`, `ci:` (and similar). Keep commits scoped; include docs/tests with behavior changes.
- Never add `Co-Authored-By: Claude ...` (or any AI co-author trailer) to commit messages. The repo's `commit-msg` hook (`.githooks/commit-msg`) strips them — but do not rely on the hook, just don't write them.
- Never commit secrets or any local `~/.skill-hub` state.
- Never create release tags by hand. The only supported release path is `./scripts/create-release.sh --version X.Y.Z --from vA.B.C --yes`, which runs tests, builds, writes release notes, and pushes both the branch and tag consistently.
- User-facing release changes go under `docs/release-notes-vX.Y.Z-*.md` before tagging.
- PR descriptions should state the user-visible change, list the validation commands you ran, link related issues, and include screenshots only for Web UI changes.

## Adding things

- **New adapter** (e.g., a new agent target): create `internal/adapter/<name>/adapter.go` + `adapter_test.go`, implement the `adapter.Adapter` interface (`Supports`, `Apply`, `Remove`, `Extract`), then register the factory in `internal/adapter/adapter.go`. The `httpapi` and `webui` blocks pick up new targets automatically once registered.
- **New cobra command**: add `internal/cli/<name>.go` + `<name>_test.go`, wire it into `root.go`'s `init()` (see existing pattern in `internal/cli/root.go:46-70`). Cover both direct CLI rendering and the underlying kernel/service call when the command is service-eligible.
- **New kernel domain**: follow the `biz/` + `pkg/` + `service/` split used by `internal/modules/kernel/hub`; expose the service through the `server` module so the bridge in `internal/cli/service_bridge.go` can reach it.
- **New test**: prefer table-driven tests in `*_test.go` next to the code under test, use `t.TempDir()` for FS isolation, and avoid network unless the scenario explicitly requires it. E2E Python tests go under `tests/e2e/test_<scenario>.py`.
