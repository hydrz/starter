---
name: build-vertical-slice
description: Implement an end-to-end product feature across TypeSpec, PostgreSQL/sqlc, Go application and HTTP layers, Orval, and React. Use when a request spans multiple architecture layers; use narrower skills for contract-only or schema-only changes.
---

# Build a vertical slice

Read `docs/development/vertical-slice.md` and the relevant standards. Use `$change-api-contract` and `$change-database-schema` for their respective boundaries rather than duplicating their rules.

## Workflow

1. Inspect repository state and translate the request into acceptance criteria, authorization rules, invariants, public errors, and observable states. Ask when domain behavior is ambiguous.
2. Identify the smallest coherent slice. Keep dependencies pointing from transport and persistence adapters toward the application boundary.
3. If persistence changes, follow `$change-database-schema`; define application/domain behavior behind a minimal repository interface.
4. If HTTP shape changes, follow `$change-api-contract`; implement the generated server interface and map domain failures to stable public errors.
5. Build the React feature from generated Orval functions/hooks. Use TanStack Query for server state, React Hook Form plus Zod for forms, and Zustand only for genuine client-global state.
6. Cover domain rules with unit tests, HTTP mapping with handler tests, and user-visible behavior with web tests. Include loading, empty, error, success, and authorization states when relevant.
7. Run targeted checks throughout, then `pnpm check`, `pnpm test`, and `pnpm build`. Run `pnpm test:database` for persistence behavior when Docker is available.
8. Update durable developer or operator documentation only where behavior or procedures changed. Summarize layer-by-layer impact and residual risks.

## Stop conditions

Stop before destructive data operations, breaking public contracts, invented authorization policy, or production actions without explicit approval. Do not weaken architecture boundaries merely to make generated interfaces compile.

## Success criteria

The requested user outcome works through every affected layer, SSOT and generated artifacts agree, failure states are intentional, tests exercise behavior, and the standard quality gates pass.
