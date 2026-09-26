---
name: change-api-contract
description: Update TypeSpec API operations, models, or errors and regenerate the OpenAPI, Go server, and Orval TypeScript artifacts. Use for endpoint or API-shape changes; do not use for database-only or implementation-only work.
---

# Change an API contract

Treat `packages/contracts/main.tsp` as the HTTP contract SSOT. Read `docs/development/contracts.md` and `docs/standards/contracts-and-data.md` before editing.

## Workflow

1. Inspect `git status --short`, the request, and affected consumers. Do not overwrite unrelated changes.
2. Clarify ambiguous compatibility, authorization, pagination, or error semantics before implementation. Preserve existing operation IDs unless a deliberate breaking change is approved.
3. Edit TypeSpec only. Do not hand-edit `spec/generated/`, `internal/api/`, or `apps/web/src/api/generated/`.
4. Run `pnpm generate:contract`, inspect the OpenAPI diff, then run `pnpm generate:server` and `pnpm generate:web`.
5. Adapt the HTTP implementation and frontend callers to the generated interfaces. Keep domain logic out of generated and transport code.
6. Add or update transport and consumer tests for success, validation, authorization, and stable public errors as applicable.
7. Run `pnpm check:generated`, targeted tests, `pnpm check`, and `pnpm build`. Report commands and results exactly.
8. Summarize compatibility impact, generated files, documentation impact, and any follow-up deployment coordination.

## Stop conditions

Stop and ask before removing or renaming a public field or operation, changing established semantics, exposing sensitive data, or resolving an unclear backward-compatibility decision. Never bypass generated-drift checks or conceal an unavailable tool as a pass.

## Success criteria

The TypeSpec source expresses the requested behavior, all generated artifacts match it, implementations compile, relevant tests pass, and compatibility consequences are explicit.
