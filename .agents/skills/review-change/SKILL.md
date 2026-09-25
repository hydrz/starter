---
name: review-change
description: Review a working tree, commit, or pull request for correctness, security, compatibility, architecture, tests, and repository standards. Use when asked for code review or pre-merge validation; do not modify code unless the user also asks for fixes.
---

# Review a change

Read `docs/standards/code-review.md` plus standards for the touched areas. Review the requested diff, not merely the final files.

## Workflow

1. Establish the review base and scope. Inspect `git status --short`, `git diff --stat`, and the relevant diff without discarding local changes.
2. Trace changed behavior across contracts, generated artifacts, application logic, persistence, UI, deployment, and documentation as applicable.
3. Prioritize functional defects, security or data-loss risks, compatibility breaks, races, resource leaks, and missing tests. Verify generated files against their SSOT rather than reviewing generated style.
4. Run the narrowest useful tests first. Run `pnpm check`, `pnpm test`, and `pnpm build` when scope and environment allow; add `pnpm test:database` or `pnpm test:embed` when relevant.
5. Report findings first, ordered by severity. For every finding, cite a precise file and line range, explain the failure scenario and impact, and propose a concrete direction.
6. Then list assumptions or questions and a brief change summary. State “no findings” when appropriate, while noting untested risks and environment limitations.

## Review boundaries

Do not edit files, commit, approve, merge, publish, or dismiss failed checks unless explicitly asked. Do not report formatting preferences as defects when repository standards permit them. Never infer a passing result for a command not run.

## Success criteria

Every finding is actionable and grounded in changed code, severity reflects impact, validation is reproducible, and the report distinguishes defects from questions and environmental limitations.
