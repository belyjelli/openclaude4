# Reference: partitioning and briefs

## Overlap matrix (quick)

For each candidate parallel task, list **primary paths** (files or dirs). If any path appears in two rows, those tasks are **not** parallel — reorder or merge.

| Track | Primary paths | Notes |
|-------|---------------|--------|
| A | e.g. `internal/mcpclient/` | |
| B | e.g. `internal/config/` | |

## Agent brief template

Copy and fill per delegated unit:

```markdown
## Goal
<one sentence>

## In scope
- Paths: `...`
- Behaviors: ...

## Out of scope (other tracks own these)
- Paths: `...`
- Do not: ...

## Definition of done
- [ ] `go test ./<pkg>/...` (or full `./...` if cross-cutting)
- [ ] TODO.md / docs updated if the item required it
```

## Anti-patterns

- Two agents editing the same **merge hotspot** (`go.mod`, root `README`, CI workflow) without a single owner.
- Parallel refactors of **one** type or function signature.
- Vague briefs (“improve MCP”) with no path boundaries — leads to duplicate edits.
- Marking TODO items done before **tests** pass for that slice.

## Dependency heuristics (Go repos)

- **Config / types first** — Changes to loaded config shape or public structs block consumers.
- **Generated or wire-up last** — `cmd/` registration often depends on `internal/` packages being stable.
- **Tests with the feature** — New behavior should land with tests in the same track when possible.
