# Build, lint, typecheck, and test baseline

Measured: 2026-10-24 (Asia/Bangkok).
Source commit: `ef15269d9c231af5913106353da445efa89a440b`.

The F1 tables below record results before application fixes. F4 subsequently cleared
frontend lint errors: current lint passes with 0 errors and 74 warnings; see the F4
section below. Frontend tests are still not configured. R1 subsequently added
PostgreSQL-backed submission authorization tests; see the R1 section for execution
requirements and current results.
Use the historical snapshot to distinguish existing debt from regressions; do not
treat existing failures as permission to add new ones.

## Environment and isolation

| Component | Observed version |
| --- | --- |
| OS / architecture | Windows / amd64 |
| Go | 1.25.3, `GOTOOLCHAIN=local` |
| Node.js | 24.13.0 |
| npm | 11.9.0 |
| Next.js (lockfile) | 16.2.6 |
| React / React DOM (lockfile) | 19.2.0 |
| TypeScript (lockfile) | 5.9.3 |
| ESLint (lockfile) | 9.39.1 |
| eslint-config-next (lockfile) | 16.0.1 |

Checks ran against a disposable copy of the current tracked `backend/` and
`frontend/` files outside the repository. Environment files were excluded.
SHA-256 comparison confirmed all 193 copied source/configuration files still
matched the working repository after the checks, including dependency manifests
and lockfiles. Next's generated files, installed dependencies, raw logs, and Go
binaries stayed outside the repository.

No API, worker, database, container, or deployed service was started. Go's build and
test commands did not invoke application `main()`. No production credentials or
student data were supplied. Frontend build used a localhost API URL.

## Executed checks

Run frontend commands from `frontend/` and Go commands from `backend/`.
`<results>` below denotes an external directory, not a repository directory.

| Check | Actual command | Exit | Result |
| --- | --- | --- | --- |
| Dependency restore | `npm ci --no-audit --no-fund` | 0 | Installed 939 packages using the existing lockfile |
| Frontend lint | `npm run lint -- --format json --output-file <results>/eslint.json` | 1 | 39 errors, 79 warnings, 0 fatal errors across 109 linted files |
| Frontend typecheck | `npm exec -- tsc --noEmit --incremental false` | 0 | Passed before Next-generated declarations existed |
| Frontend production build | `npm run build` | 0 | Turbopack compilation, Next TypeScript validation, and page generation passed |
| Frontend tests | `npm test` | 1 | Missing `test` script; no tracked frontend test files found |
| Go tests / compilation | `go test -mod=readonly ./...` | 0 | All 17 packages reported `[no test files]`; compilation passed |
| Go vet | `go vet -mod=readonly ./...` | 0 | Passed with no diagnostics |
| API binary build | `go build -mod=readonly -o <results>/server.exe ./cmd/server` | 0 | Passed; binary was not executed |
| Worker binary build | `go build -mod=readonly -o <results>/worker.exe ./cmd/worker` | 0 | Passed; binary was not executed |

Frontend checks used `CI=true` and `NEXT_TELEMETRY_DISABLED=1`; the build also used
`NEXT_PUBLIC_API_URL=http://127.0.0.1:8080/api`. These settings were process-local,
not committed application configuration. Go checks used `GOTOOLCHAIN=local` and
`-mod=readonly` to avoid automatic toolchain switching or module-file edits.

Dependency restore reported a `node-domexception@1.0.0` deprecation warning. The
build reported an empty build cache. Neither prevented completion. npm dependency
vulnerability auditing was disabled for this restore and was not performed.

## F1 lint debt (historical)

| Severity | Rule | Count |
| --- | --- | --- |
| Error | `@typescript-eslint/no-explicit-any` | 25 |
| Error | `react-hooks/preserve-manual-memoization` | 10 |
| Error | `react-hooks/refs` | 3 |
| Error | `react/no-unescaped-entities` | 1 |
| Warning | `@typescript-eslint/no-unused-vars` | 58 |
| Warning | `react-hooks/exhaustive-deps` | 19 |
| Warning | `@next/next/no-img-element` | 1 |
| Warning | `react-hooks/incompatible-library` | 1 |

All 39 errors occur in these files, relative to `frontend/`:

| File | Errors | Warnings in that file |
| --- | --- | --- |
| `src/components/assignment-ai-setup-dialog.tsx` | 5 | 2 |
| `src/components/create-template-dialog.tsx` | 3 | 5 |
| `src/components/edit-assignment-dialog.tsx` | 10 | 4 |
| `src/components/edit-template-dialog.tsx` | 6 | 4 |
| `src/components/extend-due-date-dialog.tsx` | 1 | 0 |
| `src/components/ui/file-upload.tsx` | 3 | 2 |
| `src/components/ui/tags-input.tsx` | 11 | 6 |

Useful starting points are the render-time ref access in `file-upload.tsx:349`,
memoization diagnostics beginning at `tags-input.tsx:65`, and the unescaped
apostrophe in `extend-due-date-dialog.tsx:95`. Type annotations and hook behavior
should be repaired in focused changes; do not suppress rules or upgrade packages
merely to make the baseline green. Rerun lint before using these counts or line
numbers to assess a later change.

## Practical limits and follow-up

- A passing build does not imply passing lint; both commands are necessary here.
- At F1 there was no automated business-rule coverage. Add focused regression tests
  with the permission, storage, grading, and deadline fixes in the checklist.
- F1 tested no live authentication, file storage, Judge0, queue, database, or
  AI-provider behavior. R1 later exercised local PostgreSQL and API middleware with
  fake storage; full integration and E2E validation remain outstanding.
- Docker was not available on PATH. Linux/container builds, Node 25 from the
  frontend Dockerfile, and production deployment behavior remain unverified.
- Document one supported development environment before treating these Windows
  results as a cross-platform guarantee. No dependency upgrades were made to
  reconcile the existing version differences.
- F1 is complete because the baseline and blockers are reproducible. The lint
  cleanup, test coverage, setup repairs, and CI gates remain separate work.

For another clean measurement, copy the current tracked application files into a
fresh external workspace, exclude environment files, run the commands above, and
save raw outputs externally. No database or provider credentials are required for
these compilation/static checks. Dependency restore and the build's Google Font
download may require network access.

## F2 setup-documentation verification

On 2026-10-24, the documented frontend development command was exercised in the
same external workspace with PowerShell 7.6.5 and the F1 locked dependencies:

```powershell
$env:NEXT_PUBLIC_API_URL = 'http://127.0.0.1:8080/api'
$env:NEXT_TELEMETRY_DISABLED = '1'
npm run dev -- --hostname 127.0.0.1 --port 3300
```

Next.js became ready; `GET /login` returned HTTP 200, and `GET /` returned a 307
redirect followed by HTTP 200 at `/login`. This was an HTTP smoke check, not browser
interaction, authenticated login, or E2E verification. The frontend server was
stopped with Ctrl+C, and no listener remained on port 3300. The API/worker and
their services were not started. The separately documented
`go build -mod=readonly ./...` command also passed without executing either entry
point. All nine relative Markdown links resolved, all ten documented PowerShell
blocks parsed without syntax errors, and `git diff --check` passed. Syntax checks
do not establish that the unexecuted Docker/runtime commands work.

The updated root README selects `docker/docker-compose.yml` as the primary
integration configuration while explicitly documenting its unresolved MinIO,
Judge0, readiness, and persistence issues. Docker startup and integration steps
remain unexecuted and must be verified after O1-O3 repairs. F2 completion records
the corrected documentation and verified host workflow, not a working full stack.

## F4 lint repair and behavioral checks

Measured: 2026-10-24, using the same external workspace, locked dependencies, and
host tools as F1. Seven components were changed and one type-only module added.
The credential/config types match the backend response fields; existing typed
assignment/template properties replace redundant `any` casts. Tag callbacks/effects
now declare their reactive dependencies. File input relies on Dropzone's prop getter
for its ref. The due-date dialog renders the same apostrophe with a JSX entity.
No new lint suppressions, dependency upgrades, or backend/policy changes were made.

| Check | Result |
| --- | --- |
| `npm run lint -- --format json --output-file <external-results>/f4-eslint.json` | Exit 0; 110 files, 0 errors, 74 warnings |
| Warning comparison against F1 | Five warnings removed; no added warning fingerprints (file/rule/message, ignoring shifted line numbers) |
| `npm exec -- tsc --noEmit --incremental false` | Exit 0, both before browser checks and after the final production build |
| `npm run build` | Exit 0; compilation, TypeScript validation, and page generation passed |
| Headless Chrome behavioral checks | Ten scenarios passed, zero browser runtime errors |

Behavior was checked with actual components in a temporary external Next.js page,
using existing bundled Playwright and installed Chrome. Fixtures used a fake teacher
session and intercepted every API request to the localhost fixture endpoint; no
database, university service, Judge0, or paid provider was contacted. Scenarios covered:

- Latest tag callback, changed min/max props, paste deduplication/limits, RTL
  navigation, and keyboard deletion.
- Upload keyboard file chooser/ref wiring, append, removal, and oversized-file
  rejection without losing the accepted file.
- Template creation and editing, assignment editing with dates/visibility, and AI
  setup saving before the grading callback; selected config and entered prompts
  were checked in multipart request bodies.

The fixture respects existing prompt-reset behavior when selecting a config; it
does not define a new prompt policy. The temporary route was removed before final
lint/build checks, and the dev server/browser were stopped. Harness code, generated
files, logs, and screenshots remain outside the application repository. These are
focused browser checks, not a new committed test suite or live integration coverage.

Ref handling follows the [official Dropzone guidance](https://github.com/react-dropzone/react-dropzone#refs);
callback dependencies follow [React's useCallback reference](https://react.dev/reference/react/useCallback).

## R1 submission authorization regression checks

Measured: 2026-10-24. Added
[submission_scope_test.go](../backend/internal/handler/submission_scope_test.go),
using the real Fiber router, bearer-token authentication, course membership checks,
services, repositories, and PostgreSQL driver. Storage is an in-memory fake. No
dependencies were added. Tests run serially because current packages use globals.

Before the patch, the suite reproduced mismatched-parent file reads/replacements,
assignment-mismatched grading/comments, and global/course-role confusion. Legitimate
owner/teacher controls passed. After the patch, all 82 subtests passed:

- Sixty cases exercise all six nested actions against wrong/missing parents or
  submissions and inactive/withdrawn/dropped/absent membership. Denied requests must
  leave submission, attachment, comment, timestamp, and stored-byte snapshots intact,
  with zero storage calls.
- Twenty-two controls exercise legitimate owner/teacher behavior, course/global role
  differences, student ownership and grading denial, TA/admin read denial under the
  existing permission map, authentication, uppercase UUIDs, root submission creation,
  and assignment ZIP download.

The shared route guard binds course + assignment + submission before either
submission or comment handlers. Reads use the active course role and `SendError`.
The independent read-only candidate review found no confirmed bypass or regression.
Services retaining ID-only signatures rely on this HTTP boundary; future HTTP callers
must preserve it. Standalone attachment-download access is still tracked under R10.

| Check | Result |
| --- | --- |
| `go test -mod=readonly -count=1 -run 'TestSubmission(ResourceScope\|AccessControls)' ./internal/handler` | Pass; 82 scenarios executed |
| `go test -mod=readonly -count=1 ./...` | Pass; handler tests executed, remaining packages have no tests |
| Focused tests without `TEACHING_SUPPORT_TEST_DATABASE_URL` | Both tests explicitly skipped, as documented |
| `go vet -mod=readonly ./...` | Pass |
| `go build -mod=readonly -o <external-results>/r1-server.exe ./cmd/server` | Pass; not executed |
| `go build -mod=readonly -o <external-results>/r1-worker.exe ./cmd/worker` | Pass; not executed |
| `git diff --check` | Pass |

Checks used `GOTOOLCHAIN=local`, a synthetic process-local `JWT_SECRET`, and an
explicit test URL. PostgreSQL 17.4 was already installed: a new cluster was created
under external temporary storage, listening only on loopback on an unused port.
Tests used a dedicated `teaching_support_test` database and randomly named schemas,
then rolled back fixtures and dropped those schemas. A final namespace query found
zero remaining fixture schemas; the disposable cluster was stopped and its port
had no listener. API/worker startup, AutoMigrate,
existing databases, real student data, storage providers, and production services
were not used. Binaries and test workspace remained outside the repository.
Frontend checks were not repeated because R1 changes no frontend files.

To run against your own **disposable** PostgreSQL test instance, first create an empty
database named `teaching_support_test` (or with that prefix), using a test account
allowed to create/drop schemas. From `backend/`, set the URL to that instance; this
example assumes a loopback instance on port 5432 with suitable local authentication:

```powershell
$env:GOTOOLCHAIN = 'local'
$env:JWT_SECRET = 'local-regression-tests-only'
$env:TEACHING_SUPPORT_TEST_DATABASE_URL = 'postgres://teaching_support_test@127.0.0.1:5432/teaching_support_test?sslmode=disable'
go test -mod=readonly -count=1 ./...
go vet -mod=readonly ./...
```

Use only local disposable credentials and external PostgreSQL data directories;
never point this variable at production or reuse the application's `DATABASE_URL`.
The suite rejects URLs without the test database prefix, uses its own schema, and
never calls application startup or AutoMigrate. It skips when the explicit test URL
is absent: a default successful Go command then proves compilation, not that R1's
regression tests ran. Live storage, university login, Redis/worker, and deployment
integration remain unverified.
