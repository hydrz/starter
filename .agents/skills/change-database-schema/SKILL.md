---
name: change-database-schema
description: Add or evolve PostgreSQL schema, Goose migrations, SQL queries, and sqlc-generated pgx access code. Use for persistent data-model or query changes; do not use for API-only or transient in-memory changes.
---

# Change the database schema

Read `docs/development/database.md`, `docs/standards/contracts-and-data.md`, and ADR 0002 before editing. Treat committed migrations and `db/queries/` as sources; treat `internal/store/` as generated output.

## Workflow

1. Inspect `git status --short`, existing migrations, affected queries, and application usage. Preserve unrelated work.
2. Choose the next zero-padded migration number. Never edit an already released migration; add a forward repair instead.
3. Design production changes with expand/contract when old and new application versions may overlap. Make locks, backfills, defaults, nullability, and indexes explicit.
4. Write Goose `Up` SQL. Add a useful local `Down` section, but do not present production rollback as safe when data loss or incompatibility is possible.
5. Update explicit SQL in `db/queries/`; use bound parameters and deterministic ordering for paginated queries.
6. Run `pnpm generate:db`. Do not hand-edit `internal/store/`. Adapt repository mapping and application tests to generated types.
7. Run `pnpm check:sql`, targeted Go tests, and `pnpm check:generated`. Validate the migration locally with `pnpm db:migrate` if local database is available.
8. Run `pnpm check` and document rollout, backfill, observability, recovery, and contract-phase requirements.

## Confirmation boundaries

Obtain explicit confirmation before destructive SQL, irreversible data conversion, a table rewrite with material operational risk, production execution, or deletion of persistent data. Never print secrets or production rows.

## Success criteria

Migration ordering is valid, SQL is generated reproducibly, application mappings compile, automated checks pass, and the rollout plan is safe for mixed versions.
