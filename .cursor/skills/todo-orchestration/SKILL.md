---
name: todo-orchestration
description: Breaks roadmap items from TODO.md into non-overlapping workstreams and coordinates parallel agents or Task delegations without file or responsibility overlap. Use when the user wants to execute a backlog, parallelize work, avoid merge conflicts between agents, or orchestrate sub-agents on distinct TODO items.
---

# TODO orchestration and non-overlapping agents

## When to apply

- User points at **TODO.md**, **steps/**, or a roadmap and wants **progress** or **parallel** execution.
- User asks to **split work**, **delegate**, or run **multiple agents** without stepping on the same code.

## Principles

1. **Single source of truth** — Read `TODO.md` (and repo-specific `steps/*.md` if present) before planning. Prefer checked items and ordering already implied there.
2. **Partition before parallelizing** — No two concurrent agents may own the **same files** or the **same behavioral surface** (e.g. both changing `Agent` loop semantics). Split by **package**, **feature flag**, or **vertical slice** (e.g. “MCP only” vs “config only”).
3. **Explicit boundaries** — Each delegated unit gets a written **scope**: allowed paths (globs), **forbidden paths**, and **definition of done** (tests, docs touch if required).
4. **One owner per invariant** — Only one agent changes a given **API contract**, **config schema**, or **shared type** per batch; dependents run **after** that merges or completes.
5. **Verify per slice** — After each slice: run the **narrowest** meaningful check (`go test ./path/...`, targeted lint). Full `go test ./...` before calling a batch done.

## Workflow

1. **Inventory** — List open TODO items; note dependencies (A before B).
2. **Cluster** — Group into tracks where tracks are **disjoint** on files and concerns.
3. **Order** — Schedule tracks: **foundation** (types, config, shared libs) first; **parallel** only among disjoint tracks.
4. **Brief** — For each agent or `Task` invocation, output a short brief containing:
   - Goal (one sentence).
   - **In scope**: paths and behaviors.
   - **Out of scope**: paths and parallel items others own.
   - Done when: commands + any doc/TODO checkbox updates.
5. **Sync** — If two tasks might touch one file, **serialize** or **merge scopes** into one agent.

## Using Cursor sub-agents vs in-chat Task

- **Sub-agents (Task tool / parallel agents in Cursor)** — Use for **independent** slices with non-overlapping paths. Give each the same boundary block so they cannot “helpfully” edit neighbor files.
- **Single agent** — Use when overlap is unavoidable or the change is small; do not parallelize.

## After work

- Update **TODO.md** checkboxes only for work **actually** completed in this session.
- Do not mark items done for parallel work still in flight.

## More detail

- Overlap matrix template, example briefs, and anti-patterns: [reference.md](reference.md)
