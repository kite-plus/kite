# Domain Docs

This repo is **multi-context**: there is no single glossary at the root. A root `GLOSSARY-MAP.md` points at one `GLOSSARY.md` per context, and each context keeps its own decisions.

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`GLOSSARY-MAP.md`** at the repo root. It points at one `GLOSSARY.md` per context — read the ones relevant to the topic you're about to work in.
- **`docs/adr/`**: system-wide decisions.
- **`<context>/docs/adr/`**: decisions that only bind one context.

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest creating them upfront. The `/domain-modeling` skill (reached via `/grill-with-docs` and `/improve-codebase-architecture`) creates them lazily when terms or decisions actually get resolved.

## File structure

```
/
├── GLOSSARY-MAP.md                    ← points at the contexts below
├── docs/adr/                          ← system-wide decisions
├── internal/
│   ├── content/
│   │   ├── GLOSSARY.md
│   │   └── docs/adr/                  ← context-specific decisions
│   ├── api/
│   │   ├── GLOSSARY.md
│   │   └── docs/adr/
│   └── …
├── web/
│   ├── GLOSSARY.md
│   └── docs/adr/
└── …
```

Contexts here follow the package boundaries the code already has: the Go packages under `internal/` each own their vocabulary, and `web/` (the admin UI) is its own context. `cmd/` is entry-point glue and does not need one unless it grows vocabulary of its own.

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in that context's `GLOSSARY.md`. Don't drift to synonyms the glossary explicitly avoids.

If the concept you need isn't in the glossary yet, that's a signal: either you're inventing language the project doesn't use (reconsider) or there's a real gap (note it for `/domain-modeling`).

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-0007 (event-sourced orders), but worth reopening because…_
