# AGENTS.md

Notes for agents (and people) working in this repository.

- Build and verification commands live in `Makefile`; `make check` is the pre-PR gate.
- Commit messages follow Conventional Commits, lowercase, scoped, no trailing
  full stop — the rule is spelled out in `docs/reference.md`.
- Domain documentation is multi-context; see `docs/agents/domain.md`.
- When a rule in this file cannot be kept — a verification step that needs a
  different environment, a deliberate divergence, a piece still owed — say so
  before making the change, and leave the reason where the next reader will
  find it. A deviation nobody was told about is a bug.

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`Amazon2025/kite`), driven with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical triage roles, each label string equal to its role name, plus the `wayfinder:` label family. See `docs/agents/triage-labels.md`.

### Domain docs

Layout: multi-context. See `docs/agents/domain.md`.
