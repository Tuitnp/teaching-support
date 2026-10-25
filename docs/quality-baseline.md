# Build, lint, typecheck, and test baseline

Measured: 2026-10-24 (Asia/Bangkok).
Source commit: `ef15269d9c231af5913106353da445efa89a440b`.

The F1 tables below record results before application fixes. F4 subsequently cleared
frontend lint errors: current lint passes with 0 errors and 74 warnings; see the F4
section below. R8 adds focused Node 24 date/time tests; broader frontend component
tests are not configured. R1 subsequently added
PostgreSQL-backed submission authorization tests; see the R1 section for execution
requirements and current results. R2 added storage regression tests that run without
external services or credentials; see its section below.
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
$env:JWT_SECRET = 'local-regression-tests-only-32-byte-secret'
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

## R2 storage object isolation checks

Measured: 2026-10-25 (Asia/Bangkok), on the previously measured Windows/Go host.
New uploads in local, S3, and MinIO use the shared `newObjectKey` helper with the
existing `google/uuid` dependency's cryptographically random UUID v4 generation.
No new dependencies or schema changes were needed. Local writes use exclusive file
creation to avoid truncating an existing object. Display filenames remain in
`Attachment.FileName`; all existing read/delete paths and persisted keys are retained.

Added [storage_test.go](../backend/internal/storage/storage_test.go). The suite
exercises the actual adapters and SDK uploader with local temporary files and a
loopback HTTP object fixture. It uses anonymous test clients, no cloud credentials,
and no S3/MinIO server, bucket provisioning, database, API, or worker startup.

Eleven provider scenarios passed:

- Thirty-two concurrent uploads per provider, across two independent adapter
  instances sharing the store, have distinct keys and retain their own contents.
  Each key is downloaded and read; deleting one leaves every other payload intact.
- S3 and MinIO each upload two concurrent same-name objects larger than 6 MiB through
  the SDK multipart path, then pass the same retention/read/delete checks.
- Every provider handles six filename edge cases (empty, repeated dots, both path
  separator forms, Thai, and overlong names) and zero-byte uploads without deriving
  storage paths from those names.
- Every provider reads/downloads/deletes a seeded ordinary timestamp-prefixed legacy
  object by its existing key.

The pre-patch run failed with reused keys in every adapter, including multipart
uploads; ordinary legacy-key controls passed. The post-patch suite passed. A fresh
independent read-only candidate review found no confirmed bypass or regression.

| Check | Actual command | Result |
| --- | --- | --- |
| Focused storage suite | `go test -mod=readonly -count=1 -v -run 'TestStorage' ./internal/storage` | Pass; all 11 provider scenarios |
| Full Go tests | `go test -mod=readonly -count=1 ./...` | Pass; storage tests executed; R1 tests skipped without their explicit database URL |
| Static analysis | `go vet -mod=readonly ./...` | Pass |
| API binary | `go build -mod=readonly -o <external-results>/r2-server.exe ./cmd/server` | Pass; not executed |
| Worker binary | `go build -mod=readonly -o <external-results>/r2-worker.exe ./cmd/worker` | Pass; not executed |
| Patch hygiene | `git diff --check` | Pass |

Commands ran from an external backend copy with `GOTOOLCHAIN=local`; process-local
`TMP`/`TEMP` kept test files outside the repository. Environment files were excluded
from the copy; no dependencies were installed. R1 database tests and frontend checks
were not repeated because R2 changes neither route authorization nor frontend code.
Only application source/tests and normal project documentation were added inside
the repository.

This prevents future same-name key collisions; it cannot reconstruct bytes already
overwritten or separate historical records that share one key. Existing local
read/delete rewriting of `..` in legacy keys is unchanged; new UUID keys contain no
such sequences. These tests do not verify live S3/MinIO infrastructure, credentials,
bucket initialization, or deployment behavior. Failure-safe submission replacement
remains separate R6 work.

## R3 JWT configuration and session checks

Measured: 2026-10-25. Approved policy: fixed eight-hour HS256 sessions, no refresh,
re-login after expiry, reject legacy tokens without expiry, and require a nonblank
JWT secret of at least 32 bytes. Both signing and parsing use a shared runtime
config accessor. API startup validates it immediately after dotenv, before DB
initialization; login handles signing failures through the normal error response.
The worker does not use JWTs and does not require this API-only setting.

Added config and middleware regression tests. The pre-patch run reproduced missing
expiry, acceptance of HS384/HS512, invalid identity claims, and stale-key behavior.
The final tests verify exact eight-hour issuance, valid/expired/boundary/missing-exp
tokens, signature/algorithm/claim rejection, invalid secrets, byte-length and exact
key preservation, dotenv-only configuration after package initialization, process
environment precedence, and rejection before middleware DB access. Existing
submission fixtures now set their own synthetic secret of sufficient length.

| Check | Result |
| --- | --- |
| `go test -mod=readonly -count=1 ./internal/config ./internal/middleware` | Pass |
| `go test -mod=readonly -count=1 ./...` | Pass; R1 tests initially skipped without DB URL |
| `go test -mod=readonly -count=1 ./internal/handler` with disposable test URL | Pass; all 82 R1 scenarios executed with the new JWT policy |
| `go vet -mod=readonly ./...` | Pass |
| API and worker `go build -mod=readonly -o <external-results>/r3-*.exe ./cmd/...` (separate commands) | Both pass |
| Built API with missing or short JWT secret and an invalid DB sentinel | Both exit nonzero with only the generic JWT configuration error, before DB initialization |
| `git diff --check` | Pass |

An independent source review found no confirmed backend bypass/regression. Tests,
binaries, synthetic dotenv fixtures, and a new loopback-only disposable PostgreSQL
cluster stayed outside the repository. No real authentication service, production
database, migration, or valid-config application startup was used. The test cluster
is retained temporarily for the immediately following R4 checks. No packages were
installed or dependencies upgraded.

Frontend files were unchanged and checks were not repeated. The existing browser
retains cached user state; expiry is enforced by API 401 responses, after which users
must log out and sign in again. Automatic browser navigation, refresh tokens, and
server-side revocation are outside this patch. Deploying it invalidates old tokens
without expiry; operators must provide a sufficiently long secret before startup.

## R4 human-grade protection and concurrent AI persistence

Measured: 2026-10-25. User-approved policy: teacher/TA grades always outrank AI,
including forced jobs and retries; if a human grade commits before AI persistence,
skip both AI score and feedback. Normal jobs fill only wholly ungraded rows; forced
jobs may additionally replace AI-attributed work. Unknown graders and scores without
provenance are preserved rather than treated as ungraded.

The worker now filters candidates consistently with a repository transaction that
conditionally updates the submission, checks affected rows, and upserts AI feedback
through the same transaction. PostgreSQL's submission row lock serializes manual
updates against both AI writes. If manual grading commits first, neither AI write
occurs; if AI locks first, manual grading waits, then its score wins. No manual-role
permissions changed. Removed the broad forced-reset helper, which could delete AI
feedback on human-graded work. Forced replacement now preserves prior AI output on
failure and updates an existing AI comment in place rather than deleting/recreating
it. These changes require no migrations or dependency changes.

Added [grading_policy_test.go](../backend/internal/worker/grading_policy_test.go).
Twenty scenarios run the actual Asynq handler directly (no Redis), real repositories
and PostgreSQL, synthetic encrypted credentials, fake file storage, and the existing
custom-provider/OpenAI SDK against a loopback HTTP fixture:

- Normal/forced jobs protect zero-point teacher, TA, and unknown-role grades, all
  existing manual/AI comments, authors, visibility, and timestamps in mixed batches.
- A manual grade committed during the fake provider call protects both writes under
  normal/forced execution and subsequent payload retries.
- Existing AI scores are skipped normally, replaced only when forced, and preserved
  on provider failure. Normal retries do not duplicate scores/comments; unattributed
  existing scores are left alone.
- Real separate PostgreSQL connections contend in both human-first and AI-first
  orders. Tests observe `pg_blocking_pids`, using only fixture-local SQL, triggers,
  and a transaction advisory lock to control ordering; no timing-only assumption.
- Comment-write failure rolls back the AI score. Concurrent normal saves apply once;
  concurrent forced saves and forced repeats keep one AI comment. Wrong-assignment
  saves leave the submission untouched. Successful mixed jobs finish with complete
  progress and processed counts.

The pre-patch run reproduced human overwrites in normal/forced jobs, provider-window
races, and destructive failure resets. Final cases pass. Independent source review
found no confirmed functional bypass/regression; its requested contention, rollback,
and concurrent-save checks were implemented and executed. A reported trailing blank
line was removed before final diff validation.

| Check | Result |
| --- | --- |
| `go test -mod=readonly -count=1 -timeout 90s -v ./internal/worker` | Pass; initial 19 scenarios, then expanded with concurrent forced saves |
| `go test -mod=readonly -count=1 -timeout 90s ./...` with disposable test URL | Pass; all 20 R4 scenarios, 82 R1 HTTP cases, JWT/config and storage suites executed |
| `go vet -mod=readonly ./...` | Pass |
| `go build -mod=readonly -o <external-results>/r4-server.exe ./cmd/server` | Pass; not executed |
| `go build -mod=readonly -o <external-results>/r4-worker.exe ./cmd/worker` | Pass; not executed |
| `git diff --check` | Pass |

Checks used an external source copy, `GOTOOLCHAIN=local`, and external `TMP`/`TEMP`.
Only the explicit disposable test database was used, in random schemas dropped by
test cleanup. No AutoMigrate, application startup, paid AI calls, live Redis/provider,
production/student data, package installation, or infrastructure changes occurred.
Frontend files were unchanged and their checks were not repeated. Broader R12 job
recovery/manual multi-write atomicity and R5 result attribution remain unverified.
Final cleanup found zero R1/R4 fixture schemas, stopped the disposable PostgreSQL
cluster, and confirmed that its port had no listener. All 79 checked backend files
matched the external test copy; all 75 relative links in the changed project
documents resolved.

## R5 strict AI result attribution

Measured: 2026-10-25. Added explicit JSON attribution validation and removed both
unused-result reassignment and response-order fallback. A batch requires exactly
one result per input index, with unique in-range integer indexes. Missing/null
indexes/items and repeated JSON fields (including case aliases) are rejected.
An optional ID must be a nonempty string matching the indexed submission. The
entire batch is validated before any score/comment persistence. R4's transaction,
score clamping, comment normalization, and existing code-fence handling are retained.

Added `backend/internal/worker/grading_result_test.go` and extended the existing
fake-provider fixture. Before patching, duplicate/index-field responses succeeded
through the worker and pure tests reproduced reassignment/defaulting bugs. After:

- All 23 pure cases pass, including shuffled distinct scores/comments and legitimate
  index zero, plus score/comment normalization controls.
- All 40 PostgreSQL cases (20 malformed classes under normal and forced jobs) fail
  safely with unchanged submission/comment snapshots and a failed job status.
- Existing 20 R4 grading cases and 82 R1 HTTP cases pass alongside JWT/config and
  storage tests. Independent review found no confirmed bypass/regression.

Commands from the external backend copy: `go test -mod=readonly -count=1 -timeout
120s ./internal/worker`, then `go test -mod=readonly -count=1 -timeout 120s ./...`
with the disposable test URL, `go vet -mod=readonly ./...`, and separate API/worker
`go build -mod=readonly -o <external-results>/r5-server.exe ./cmd/server` /
`go build -mod=readonly -o <external-results>/r5-worker.exe ./cmd/worker`; all passed.
`git diff --check` passed. Binaries were not executed. Tests used the existing
installed tools and a new external loopback-only PostgreSQL cluster, retained for
the following R6/R7 checks. No packages, real credentials/student data, production
services, AutoMigrate, or frontend changes were involved; frontend checks were not
repeated. Earlier completed batches in a multi-batch job are not rolled back when a
later batch is rejected; broader job atomicity/recovery remains R12.

## R6 failure-safe submission replacement

Measured: 2026-10-25. User approved preserving fileless removal and existing
grades/comments, while retaining replaced files and attachment records for a
separate cleanup task. Replacement now stages incoming bytes, then atomically
inserts attachment metadata, updates one owner-scoped submission reference, and
loads the response. Missing-at-write rows are rejected. Only attachment reference
and its existing update timestamp change, preserving concurrent grades. Upload
preparation is shared with creation without changing creation's existing writes.

Added `backend/internal/handler/submission_replacement_test.go` using the existing
disposable PostgreSQL fixture and a storage fake that records deletion calls. The
pre-patch run reproduced deletion before success and multiple failure paths. Final
18 cases pass: success, fileless detach, first upload, shared attachment ID, shared
legacy key, repeated replacement, concurrent grading, open/upload/metadata-insert/
reference-update/response-reload failures, missing-at-write rollback, oversized/
hidden/closed rejection, two concurrent uploads, and concurrent upload/detach.
Snapshots verify original references, attachment rows, grade/grader, comments and
bytes on known failure paths; every case requires zero deletion calls. Successful
replacement retains shared and prior bytes, and the final reference resolves to
stored content. Failure injection uses removable GORM callbacks in test fixtures.

Focused command: `go test -mod=readonly -count=1 -timeout 90s -run
'Test(SubmissionReplacement|ConcurrentSubmissionReplacements)' ./internal/handler`.
Then `go test -mod=readonly -count=1 -timeout 120s ./...` with the disposable test
URL, `go vet -mod=readonly ./...`, and separate API/worker builds to external
`r6-server.exe`/`r6-worker.exe` all passed. Existing R1/R4/R5, JWT/config and storage
tests executed. `git diff --check` passed. Independent source review found no
confirmed bypass/regression. No frontend changes/checks, installed packages,
production data/services, migrations, or valid application startup were involved.

The deliberate tradeoff is retained storage: old objects/rows and newly staged
objects after DB failure are not deleted in the request. In an uncertain commit
outcome, retaining bytes avoids deleting the now-current file; it does not claim
to distinguish committed from uncommitted acknowledgments. No garbage collector
was added. Future cleanup must inventory all references and shared keys before
deletion; existing explicit `DeleteAttachment`/`DeleteAssignment` behavior is not
hardened by this fix. The external test cluster remains available for R7 checks.

## R7 owner-and-ID-scoped AI config updates

Measured: 2026-10-25. The service now forwards the requested config ID to the
repository, whose UPDATE requires both that ID and the authenticated owner. Zero
matched rows return `ErrAIConfigNotFound`; DB errors retain existing handling. No
permission, credential, field-selection, schema, or dependency behavior changed.

Added `backend/internal/handler/ai_config_scope_test.go`, using the actual router,
JWT middleware, services/repository, and disposable PostgreSQL. The pre-patch run
reproduced sibling mutations on ordinary edits, zero/empty field updates, and
owner/deletion races. Final 18 cases pass:

- Fifteen HTTP cases check exact target and response fields, same-owner siblings
  sharing/differing credentials, foreign configs, missing/foreign credentials,
  omitted credentials, student/TA/admin global roles, unauthenticated/malformed
  requests, DB errors, and target owner changes/deletion before the UPDATE.
- Three repository cases independently verify the owned target, wrong owner, and
  missing ID. Non-target configs and all credential rows, including timestamps,
  remain identical in snapshots. Write-time race/failure callbacks are removed.

The omitted-credential test initially exposed an incorrect test expectation: the
existing empty-UUID lookup returns generic 500, not 404. That behavior is preserved
and recorded under R9; the fixture uses a savepoint to isolate the failed UUID query
from its outer transaction. Missing and foreign valid UUID targets return 404.

`go test -mod=readonly -count=1 -timeout 90s -run
'TestAIConfig(UpdateScope|RepositoryScope)' ./internal/handler` and
`go test -mod=readonly -count=1 -timeout 120s ./...` passed with the disposable URL.
All prior R1–R6 regression suites executed; `go vet -mod=readonly ./...`, separate
API/worker builds to external `r7-server.exe`/`r7-worker.exe`, and `git diff --check`
passed. Independent source review found no confirmed bypass/regression. No live
provider or university calls, real credentials/student records, migrations,
application startup, package installs, or frontend changes/checks occurred.

Final isolation checks matched all 83 backend source/config files to the external
tested copy. Database cleanup found zero fixture schemas; the external PostgreSQL
cluster was stopped. All personal test workspaces and outputs stayed outside the
repository. Retained-file cleanup and explicit deletion safety remain the separate
R6 follow-up, and the R9 ID-validation gap remains open.

## R8 deadline precision and stable lateness reporting

Measured: 2026-10-25 (Asia/Bangkok). The user approved full UTC schedule
responses, Bangkok display/edit, inclusive start/close and on-time exact due,
extensions that never shorten original close, and reporting based on the latest
current file upload/replacement. No schema or dependency change was required.

`models/deadlines.go` shares effective deadlines and submission admission rules.
Assignment list/detail/write responses now use UTC RFC3339Nano strings rather
than date-only strings. Submission create/update checks start and effective close,
and returns override lookup errors before storage or DB writes. Summaries preload
attachments, fetch overrides in one assignment-scoped query, and use attachment
`CreatedAt` instead of submission `UpdatedAt`. Summary and both CSV exports select
the latest current file-bearing attempt. Missing historical timestamps produce
`timing unavailable`; known timestamps win over unknown ones, with submission ID
as a deterministic tie-break. This is not a historical timestamp reconstruction.

The shared frontend formatter/parser/picker uses Bangkok time. The existing
DayPicker timezone capability is reused. Direct local/TZDate time setters exposed
an unrelated host-DST normalization in a regression check, so time changes use UTC
arithmetic for the approved fixed UTC+07:00 offset. Untouched text fields are not
reparsed on blur; untouched assignment edits retain the original API timestamp,
including fractions below JavaScript's millisecond precision. Deliberate date/time
changes use millisecond precision. Picker buttons have explicit button types and
zone labels; the summary filter includes overdue and timing-unavailable statuses.

Added normal application tests:

- `backend/internal/models/deadlines_test.go`: offset/subsecond DTO round-trips,
  exact start/due/close and adjacent nanosecond boundaries, extension close
  preservation/extension, hidden assignments, deterministic current-file selection.
- `backend/internal/service/deadlines_test.go`: exact due/personal due, grading
  timestamp independence, unknown historical timing, selected scores and fileless
  attempts.
- `backend/internal/handler/deadlines_test.go`: ten real-router admission cases,
  two fail-closed override lookup cases with unchanged data/storage, summary with
  manual grading and simulated AI score/timestamp mutations, current-file
  replacement, student list/detail extension responses, DB-precision edit
  round-trip, and single-assignment/course CSV reporting.
- `frontend/tests/datetime.test.mjs`: three Node `node:test` cases (21 assertions),
  covering UTC/Bangkok/Los Angeles hosts, Bangkok midnight, natural-language
  parsing, untouched fractional precision, and the host DST gap. Run with
  `npm run test:datetime` on the verified Node 24 environment; no test dependency
  was installed. Node emits an informational module-type detection warning for
  importing the existing TypeScript module; app module configuration is unchanged.

The initial check run completed Go tests/vet, both binary builds and frontend
lint/typecheck/build before the last tests and UI precision refinements. A device
freeze interrupted later verification; no completion/commit was claimed from
those partial logs. Resumed checks run sequentially, with Go `-p 1` and
`GOMAXPROCS=2`. Frontend build uses process-local `CIRCLE_NODE_TOTAL=3`, which the
locked Next.js configuration maps to two page-generation workers; no application
or deployment configuration was edited for resource limits.

Resumed `npm run test:datetime`, `npm run lint` (0 errors, existing 74 warnings),
`npm exec -- tsc --noEmit --incremental false`, and `npm run build` passed against
current frontend source in the existing external workspace with locked installed
dependencies. A temporary harness outside the repository used installed headless
Chrome and synthetic intercepted API data. All three browser timezone scenarios
passed Bangkok detail display, untouched blur/save with original microseconds,
and hour/minute/calendar-day changes with the expected UTC multipart payloads.
No live application API, university service, or AI provider was used.

Final `go test -mod=readonly -p 1 -count=1 -timeout 120s ./...` and
`go vet -mod=readonly -p 1 ./...` passed with the explicit disposable PostgreSQL
URL, executing the new R8 and prior R1-R7 database/worker regressions. The focused
HTTP run also passed. Entry points compiled as part of the full test run; both
standalone binaries had built successfully before the interruption with the same
runtime code. `git diff --check` passed. All 112 current frontend source/manifest/
lock/test files matched the external checked copy byte-for-byte. The disposable
PostgreSQL database had zero fixture schemas, then the cluster was stopped; neither
test DB nor frontend dev port remained listening. The test browser was closed.
Personal harnesses/profiles/results remained outside the repository. No packages,
tools, migrations, production operations, or deployment changes were performed.

Remaining limits: historical file timestamps cannot be inferred from grades;
duplicate memberships/overrides, attempt limits, detail-page default attempt,
and incomplete/not-started statistics remain R13. Input ordering/score validation
remains R9; atomic override replacement remains R12. The API/worker application
entry points were not started and no existing application database was migrated.

## R9 server-side input validation

Measured: 2026-10-25 (Asia/Bangkok). The user approved positive integer maximum
points, `start < due <= close`, explicit integer grades in [0, maximum], existing
five-file/10 MiB attachment and 5 MiB submission limits, nonempty submission uploads,
and rejection of maximum reductions below existing grades without rescaling.

Transport date/point fields now distinguish omission from explicit zero. Create
validates required values; edits validate the merged date/point values. Assignment
and template attachment validation runs before scalar writes/uploads, checks the
entire batch, and validates distinct owned attachment IDs and optional owned AI
configs. General uploads reject invalid later files before uploading the first.
Submission creation requires one nonempty file; supplied replacements must also
be nonempty. A fileless update still detaches and retains grades/comments (R6).
UUID checks reject malformed/empty identifiers and URN syntax before PostgreSQL;
canonical uppercase UUIDs remain supported. The R7 omitted-credential 500 is now
400 and its regression expectation was updated without changing ownership rules.

Assignment edits and manual/AI grade writes acquire the assignment row lock before
submission writes. Repository checks use current dates/points and existing grades
under that lock, preventing races between partial edits, reductions and grades.
Storage/provider calls are outside these short transactions. Ineligible/missing
AI saves retain R4's no-op behavior; eligible stale scores exceeding a newly
reduced maximum fail without changing scores/comments. Existing historical grades
are not normalized by unrelated edits or by increasing/keeping a maximum.

Added `backend/internal/handler/input_validation_test.go`, reusing the actual
router and explicitly disposable PostgreSQL fixture. Its six groups cover:

- Assignment/template create/edit: missing, zero, negative, fractional/overflow
  points; missing/bad/ordered dates; partial fields; equality at close/due;
  32767 maximum; count/mixed-reference/size limits; malformed, foreign, duplicate
  and missing references; owned AI configs; maximum reductions and legacy scores.
- Both manual grade routes: omitted/null, negative, above-max, fractional/overflow
  rejection, and accepted zero/full points with persisted grades.
- Submission create/update: missing, empty, duplicate, valid, exact 5 MiB and
  oversized files. Broken multipart is rejected by fasthttp before routing;
  its transport error is checked separately from handler HTTP responses.
- General attachment uploads: five files, exact 10 MiB, oversized later file and
  six-file rejection. Invalid cases require unchanged DB/storage snapshots.
- Malformed assignment/submission/template/attachment/config/credential IDs,
  including missing and URN credentials, return 400 without mutation.
- Four concurrent-write cases observe real PostgreSQL blocking, then prove bounds
  are rechecked after the competing transaction commits: maximum versus manual/AI
  score, existing grade versus reduction, and overlapping partial date edits.

The test app's body limit now matches the existing production 60 MiB setting so
exact 5/10 MiB application boundaries execute. The R1 allowed-create control now
supplies a file; R8/worker direct callers use presence-aware forms. Permission,
replacement, human-grade protection and strict AI mapping assertions remain intact.

An initial compile check caught and resolved a UUID variable-name error. Initial
multipart test expectations were corrected to account for transport rejection.
The first full run then exposed a missing-assignment AI no-op regression caused
by locking before the conditional save; no-op/eligibility handling was restored.
Final `go test -mod=readonly -p 1 -count=1 -timeout 120s ./...` and
`go vet -mod=readonly -p 1 ./...` passed with the disposable PostgreSQL URL and
`GOMAXPROCS=2`. All prior R1-R8 suites executed, including worker concurrency tests;
entry points compiled. The final focused rerun also passed all six groups and
131 cases with no skips, including exact persisted boundary values and preserved
historical grades. `git diff --check` passed. No frontend source changed, so
frontend checks were not repeated; its last measured lint remains 74 warnings.

The disposable test database had zero fixture schemas and was stopped afterward;
its port no longer listened. Personal scratch/logs stayed outside the repository.
No packages/tools, application startup, production services, real credentials or
student data, migrations, infrastructure or deployment changes were involved.
Full edit and manual score/comment atomicity after runtime I/O/DB failures, and
broader job retry/recovery remain R12. Assignment-level grade serialization adds
short DB contention within one assignment; it holds no lock during AI requests.


## R10 attachment download authorization

Measured: 2026-10-25 (Asia/Bangkok). The user approved uploader access,
active memberships for nonowners, visible/started student handouts with no due/close
cutoff, staff file permissions, and access through any independently authorized
reference. Replaced first-submission precedence and course-ID intersection with
one scoped reference lookup, pairing assignment state and active course role.
Unreferenced/template-only files remain uploader-only; unknown start timestamps
fail closed for nonowner students. Removed unused broad non-student lookup helpers.

`go test -mod=readonly -p 1 -count=1 -timeout 120s ./internal/handler ./internal/service
-run TestAttachment -v` passed, including 34 new HTTP scenarios, authorization DB
failure, and four exact-release unit cases. HTTP cases verify unchanged records
and zero storage calls on denial. The existing upload-validation tests also ran.
Full `go test -mod=readonly -p 1 -count=1 -timeout 120s ./...` and
`go vet -mod=readonly -p 1 ./...` passed with an explicit disposable PostgreSQL URL;
DB regressions executed, including prior R1-R9 and worker tests. `git diff --check`
passed. Checks ran sequentially with GOMAXPROCS=2 and offline Go module settings.

No frontend source changed; frontend checks were not repeated. Nested submission
read and ZIP permission gates were preserved. No packages, tools, real services,
application startup, migrations, schema/deployment configuration or production
operations were introduced. Test cluster/logs are outside the repository and are
reused for the separately authorized R11 checks before shutdown. Shared-file
deletion/cleanup and legacy duplicate membership policy remain separate work.


## R11 delegated TA permissions and manual authority

Measured: 2026-10-25 (Asia/Bangkok). The user authorized choosing teacher-like TA
powers with fewer critical permissions and requested the reason for each choice.
`docs/permissions.md` records every permission, global/course scope, the eight
TA grants, and the teacher-only critical operations. Admin/student/teacher grants
and existing global role assignments are unchanged. No schema change is needed.

TA course membership now permits unreleased assignment/submission reads, summaries,
roster access, individual files, own feedback and initial/AI-grade review. Teachers
retain correction authority for all manual grades. TA writes protect teacher/TA,
unknown, empty-grader and unattributed grades. A shared model predicate is checked
both before writing and under assignment then submission locks. Comment-own
visibility now checks author ID and role, protecting other TAs' feedback. A denied
course delete used Fiber's error with an application-only translator and returned
500; its gate now returns the application's 403 error.

Focused command passed:
`go test -mod=readonly -p 1 -count=1 -timeout 120s ./internal/handler -run
'TestTA|TestManualGradeRolePrecedence|TestSubmissionAccessControls' -v`.
Five new groups cover 100 cases: 17 scoped read/global-role/status controls,
48 teacher/TA/student cases across both grade routes and prior grader states,
seven authored-feedback cases, 26 critical-operation denials, and two real
PostgreSQL concurrent teacher/TA writes. The existing R1 TA read expectation now
permits authorized staff access. Denied operations preserve data and avoid storage.
The initial focused runs caught the course-delete status bug above plus test
fixture omissions (full course columns and the correct AI-config URL); those were
corrected and the final focused run passed without skips.

Full `go test -mod=readonly -p 1 -count=1 -timeout 120s ./...` and
`go vet -mod=readonly -p 1 ./...` passed with the explicit disposable PostgreSQL URL,
including R1-R10, AI worker and storage regressions. `git diff --check` passed.
Checks ran sequentially with GOMAXPROCS=2, offline cached modules, and all scratch,
DB and logs outside the repository. Zero fixture schemas remained; PostgreSQL was
stopped and its port confirmed closed. No existing database, application startup,
real credentials/student data, production services, packages/tools, migrations or
deployment configuration were involved.

No frontend source changed, so frontend checks were not repeated. Existing
frontend controls do not fully express TA permissions or individual feedback
authorship; Q5 tracks this UI alignment. Backend scope/authority are enforced
regardless of controls. Manual score/comment failure atomicity remains R12;
duplicate memberships, feedback release and job recovery remain separate work.


## R12a manual score and feedback atomicity

Measured: 2026-10-25 (Asia/Bangkok). Manual grade-plus-comment now shares one
transaction, retaining R9 bounds and R11 human authority under assignment then
submission locks. Authored upserts key on submission/author/role and serialize
on the assignment, preserving existing comment IDs and avoiding concurrent inserts.
AI's separate role-scoped persistence is unchanged.

PostgreSQL-backed `TestManualGradeCommentAtomicity` covers grade failure, comment
insert/update failures with full snapshot rollback, and retry without duplicate
feedback. `TestConcurrentAuthoredFeedback` verifies two real concurrent writes
leave one record. Both passed with an explicit disposable PostgreSQL URL. The
focused R1 submission, R9 concurrent-bound and R11 TA/grade suites also passed.
`go vet -mod=readonly -p 1 ./internal/repository ./internal/service ./internal/handler`
and `git diff --check` passed. Checks used GOMAXPROCS=2, offline cached modules,
and an external disposable cluster/log directory. Assignment writes and queue/worker
recovery are the next R12 parts; R12 is not yet marked complete. No frontend,
packages, live services, schema or deployment configuration changed.


## R12b assignment content and extension atomicity

Measured: 2026-10-25 (Asia/Bangkok). Assignment creation/edit now stages uploads,
then commits attachment metadata, scalar fields, normalized tags and both join
sets together. Edit response loading is inside the transaction; current date/max
checks still run under the assignment lock. Tag resolution tolerates existing
names and races through the existing unique name constraint. Existing extensions
are updated in place under the same assignment lock, rechecking the current due.
No new schema constraints or migration were added. Duplicate override policy is R13.

Ten `TestAssignmentContentAtomicity` create/edit scenarios cover failures in
attachment metadata, tag creation and both joins, plus successful full content.
Failed writes preserve DB snapshots and prior bytes, while retaining the staged
upload as required by R6. `TestOverrideFailurePreservesExtension` verifies rollback
of an injected update failure. These passed, as did the existing R9 assignment/
template input matrix and real concurrent bound checks.
`go vet -mod=readonly -p 1 ./internal/repository ./internal/service ./internal/handler`
and `git diff --check` passed. PostgreSQL/logs remain disposable and external.
Template/library multi-step writes are outside this assignment fix; no frontend,
packages, application startup, live database, or deployment changes occurred.
Queue/worker recovery remains the final R12 part.
