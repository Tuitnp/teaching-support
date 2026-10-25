# Teaching Support development instructions

## Start here

- Read `docs/development-checklist.md` for the agreed priorities, current status,
  acceptance checks, and unresolved domain decisions. Read only the relevant items.
- Use `docs/architecture.md` for source/API pointers and `docs/domain-rules.md` for
  current behavior versus unresolved policy. Ask before implementing ambiguous rules.
- Inspect current source before relying on historical audit findings. Update the
  checklist when work is completed, priorities change, or checks reveal new facts.
- Keep work scoped to one observable outcome. Preserve existing user changes and
  avoid unrelated formatting, dependency upgrades, and architectural refactors.
- After completing a fix and passing the relevant checks, create a local Git commit
  containing the fix and its tests/documentation. Inspect the staged diff, exclude
  unrelated changes, and report the commit hash. Do not push unless requested.

## Architecture pointers

- Frontend: Next.js App Router in `frontend/src/app`; shared components in
  `frontend/src/components`; authentication in `frontend/src/lib/auth-context.tsx`.
- Backend: Fiber router -> handlers -> services -> repositories under
  `backend/internal`; API and worker entry points are under `backend/cmd`.
- Contracts and permissions: `backend/internal/models/{forms,response,models}.go`.
- Deadlines and response conversion: `backend/internal/models/convertor.go`;
  submission rules: `backend/internal/service/submission_service.go`.
- Storage: `backend/internal/storage`; grading: `backend/internal/worker`;
  queue: `backend/internal/queue`; deployment definitions: `docker/`.

## Safety and scope

- Scope nested resources to the requested course, assignment, owner, and active
  membership. Distinguish global user roles from course roles.
- Add behavior-focused regression tests for permissions, submission preservation,
  grading, deadline boundaries, and retry behavior when changing those paths.
- API and worker startup both run GORM AutoMigrate. Do not run either against an
  existing or production database without authorization for those side effects.
- Obtain explicit authorization for production operations, migrations, credential
  changes, or permissions/grading policy changes that are not already authorized.
  Do not ask again for work already covered by the user's approval.
- Never commit real credentials or student records. Keep AI-specific tools,
  configuration, graphs, indexes, memories, exports, and scratch notes physically
  outside this repository, including ignored directories. Stable human-readable
  project documentation and normal application tests belong in the repository.

## Commands and verification

Use npm with the existing lockfile. Once dependencies are available, run checks
from the relevant directory:

```text
frontend/: npm run lint
frontend/: npm exec -- tsc --noEmit --incremental false
frontend/: npm run build
backend/:  go test -mod=readonly ./...
backend/:  go vet -mod=readonly ./...
```

Measured results are in `docs/quality-baseline.md`. Builds, typecheck, frontend lint,
and Go vet passed; lint still has 74 warnings. Backend submission authorization and
AI grading regression tests require `TEACHING_SUPPORT_TEST_DATABASE_URL` naming a disposable PostgreSQL
database; see the baseline for setup. Without it, those tests skip. Frontend tests
are not configured. Storage regression tests run without external services or
credentials. A Go test command without executed tests proves compilation,
not coverage. Run
focused checks first, broader checks as appropriate, and `git diff --check` before
handoff. Report commands actually run, failures, skipped checks, and remaining
risks. Do not mark checklist items complete without their acceptance evidence.
