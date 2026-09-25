---
name: prepare-release
description: Prepare and validate a semantic-version release, embedded binary, container image inputs, checksums, notes, and draft-release readiness. Use for release candidates or version tags; never deploy or publish without explicit human confirmation.
---

# Prepare a release

Read `docs/deployment.md`, `.github/workflows/release.yml`, and `docs/templates/release-notes.md`. This Skill prepares evidence; the workflow remains the release authority.

## Workflow

1. Inspect `git status --short`, current branch, recent tags, and the requested version. Require a clean, reviewed release commit and a `vMAJOR.MINOR.PATCH` version unless repository policy explicitly changes.
2. Determine changes since the previous release. Highlight breaking API/schema changes, migrations, configuration changes, security fixes, and operator actions.
3. Run `pnpm check`, `pnpm test`, `pnpm test:race`, `pnpm test:embed`, and `pnpm build`. Run `pnpm test:database` and Docker image validation when the environment supports them; report skipped checks plainly.
4. Inspect the built binary and compute SHA-256 using the same artifact naming and linker metadata as `.github/workflows/release.yml`. Do not substitute an unverified local convention for the workflow.
5. Draft release notes from `docs/templates/release-notes.md`, including upgrade order, migration impact, rollback/recovery constraints, known risks, and exact verification evidence.
6. Confirm the tag target and remote state immediately before proposing a tag. Stop after presenting the release plan unless the user explicitly authorizes external writes.
7. If authorized, create only the requested tag or draft release, then verify the remote result and report immutable identifiers. Production deployment remains a separate confirmation.

## Confirmation boundaries

Require explicit confirmation before creating or pushing a tag, publishing a GitHub Release or package, pushing an image, changing `latest`, deploying, rolling back, or writing to any external system. Never expose credentials or bypass required checks and approvals.

## Success criteria

The release candidate is reproducible, checks and limitations are recorded, artifacts and checksums correspond, upgrade guidance is actionable, and no external publication occurred without explicit approval.
