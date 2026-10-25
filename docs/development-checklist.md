# Development checklist

Updated: 2026-10-25 (Asia/Bangkok)

This is the shared project roadmap from the repository audit and its approved
recommendations. It records application work and developer documentation, not
personal AI memory or tool state. Recheck the source before implementing an item;
the audit was static and did not verify production behavior.

## Current status

- [x] Read the external proposed development plan and audit the existing repository.
- [x] Present the architecture, quality baseline, risks, and recommendations.
- [x] Receive approval to continue and record the work in project documentation.
- [x] Add this checklist and concise root `AGENTS.md` instructions.
- [x] Establish a reproducible build, lint, typecheck, and test baseline.
- [ ] Implement and verify the fixes below in small, reviewable changes.

The baseline is recorded in [quality-baseline.md](quality-baseline.md). Typecheck,
frontend build, Go compilation, Go vet, and both Go binary builds passed. F4 cleared
all 39 frontend lint errors and removed five warnings: lint now passes with 74
warnings and no new warning fingerprints. Frontend tests are not configured. R1
adds backend submission authorization tests that require an explicit disposable
PostgreSQL test URL; without it, tests skip. F4 added focused frontend typing/hook/ref
fixes; R1 scopes nested submission routes. R2 isolates new storage uploads and adds
storage regression tests using local fixtures. Completed fixes must be committed locally
with their tests/documentation under AGENTS.md. Infrastructure remains unchanged.
F2 setup documentation describes
the verified host workflow and the blocked Compose integration path. F3 source/API
and domain references are available in [architecture.md](architecture.md) and
[domain-rules.md](domain-rules.md). Continue with scoped fixes; do not combine all
pending work into one change.

## 1. Foundation

- [x] **F1: Establish the baseline.** Confirm the supported local environment and
  dependency versions. Run frontend lint, typecheck, and build, plus Go test/vet.
  Record existing failures separately from regressions and distinguish commands
  that compile code from tests that exercise behavior.
  Acceptance: exact commands, environment versions, outcomes, and blockers are
  reproducible without production credentials or student data.
  Evidence: [quality-baseline.md](quality-baseline.md), measured 2026-10-24 in a
  disposable external workspace. Completion records the baseline, not a green gate.
- [x] **F2: Correct setup documentation.** Update the root README and replace the
  generic frontend setup notes. Identify the supported Compose configuration,
  required environment variables, startup side effects, and safe local workflow.
  Acceptance: a fresh disposable environment can follow the documented steps;
  configuration defects are tracked separately from documentation corrections.
  Evidence: root/frontend READMEs now cover the F1-verified compile commands,
  runtime environment and side effects, the primary Compose file, and O1-O3
  blockers. The documented frontend dev command started in the external disposable
  workspace; `/login` returned HTTP 200 and `/` redirected to `/login`. Backend
  `go build -mod=readonly ./...` passed. Full-stack startup is explicitly unverified
  and remains integration follow-up, not a passing result of this documentation task.
- [x] **F3: Add lean architecture and domain references.** Create
  `docs/architecture.md` and `docs/domain-rules.md` with source pointers, a compact
  API contract map, and current versus intended behavior. Add ADRs only for real
  decisions, not as an initial documentation framework.
  Evidence: both references describe current source, model relationships, API/data
  flows, known gaps, and explicitly unresolved policies. No ADR or policy decision
  was invented. Documentation/source-link checks are recorded in the handoff below.
- [x] **F4: Repair existing frontend lint errors in focused changes.** Start with
  the seven files and rule counts in [quality-baseline.md](quality-baseline.md).
  Preserve behavior; do not disable rules or upgrade dependencies just to pass.
  Acceptance: no lint errors, no added warnings, and typecheck/build still pass;
  validate affected upload, tag-input, and assignment/template behavior.
  Evidence: lint 0 errors/74 warnings with no added warnings; typecheck and build
  passed. Ten isolated browser scenarios passed for tags, file uploads, and
  assignment/template AI setup/save payloads. See the F4 quality-baseline section.

Audit baseline: no tracked tests or CI workflows were found; frontend dependencies
and required local environment files were absent. Builds, lint, typecheck, tests,
and service startup were not executed. `gofmt -l` reported 65 files would change;
do not fold repository-wide formatting into functional fixes. The unchanged tree
passed `git diff --check`. Revalidate these observations as development proceeds.

## 2. Highest-priority fixes

Each checkbox should normally be a separate change with focused regression tests.

- [x] **R1: Scope submission access to the full resource hierarchy.** Previously,
  reads used global role and submission ID without binding both URL parents;
  updates omitted parent checks, and grading/comments omitted assignment checks. See
  `backend/internal/handler/submission_handler.go`,
  `backend/internal/service/submission_service.go`, and
  `backend/internal/repository/submission_repository.go`.
  Acceptance: unrelated course/assignment IDs, unauthorized readers, and inactive
  memberships cannot access or update the resource; legitimate access still works.
  Evidence: shared submission-route middleware binds all six nested actions to the
  requested course and assignment; reads use course role. All 82 PostgreSQL-backed
  HTTP regression scenarios passed, including no-write/no-storage-access assertions
  for denied requests and legitimate create/download controls. Go test/vet and both
  binary builds passed; independent source review found no confirmed bypass or
  regression. See the R1 section in [quality-baseline.md](quality-baseline.md).
- [x] **R2: Make storage keys collision-resistant.** Local, S3, and MinIO uploads
  previously used a second-resolution timestamp plus filename. See
  `backend/internal/storage/{local,aws,minio}.go`.
  Acceptance: same-name concurrent uploads have distinct keys and retain their own
  bytes; downloads/deletion affect only the intended object.
  Evidence: all providers share random UUID keys independent of filenames; local
  file creation is exclusive. Eleven provider scenarios passed for concurrent
  same-name uploads, multipart uploads, filename edge cases, empty files, and
  ordinary legacy keys. Downloads/reads retain each payload and deletion preserves
  other objects. Full Go test/vet and both binary builds passed; R1 database tests
  skipped without their explicit test URL. Independent review found no confirmed
  bypass or regression. See [quality-baseline.md](quality-baseline.md). Existing
  collided objects/overwritten bytes are not repaired by new-key generation.
- [x] **R3: Validate JWT configuration and define expiry.** Previously, secrets were
  captured before dotenv and expiry was commented out. See
  `backend/internal/service/jwt.go`, `backend/internal/middleware/middleware.go`,
  and `backend/internal/config/config.go`.
  Acceptance: missing secrets fail safely, dotenv/system environment paths behave
  consistently, and valid/expired/invalid tokens follow the agreed session policy.
  Evidence: user approved fixed eight-hour sessions, re-login after expiry, rejection
  of legacy tokens without expiry, and a minimum 32-byte nonblank secret. The API
  validates after dotenv and before DB startup; signing/parsing share the validated
  runtime secret. Configuration/token tests, both invalid-config startup probes,
  all 82 submission HTTP scenarios, full Go test/vet, and both binary builds passed.
  Independent review found no confirmed backend bypass/regression. Browser expiry
  returns 401; automatic login navigation is not implemented. See the R3 baseline.
- [x] **R4: Protect human grades from AI overwrites.** Previously, the worker included
  human grades in pending/regrade sets and wrote AI grades unconditionally. See
  `backend/internal/worker/grading_handler.go`.
  Acceptance: agreed grade precedence is enforced at write time, including when a
  teacher grades while a job is running; retries do not overwrite protected scores.
  Evidence: approved human-over-AI precedence now applies to both selection and a
  conditional score+feedback transaction. Normal jobs fill entirely ungraded rows;
  forced jobs additionally replace AI grades. Removed destructive pre-reset so
  protected feedback and prior AI results survive failed regrading. Twenty real
  PostgreSQL/fake-provider scenarios passed, including actual contention in both
  write orders, concurrent normal/forced saves, rollback, and retries. Full Go
  tests (including all 82 R1 scenarios), vet, and both binary builds passed.
  Independent review found no confirmed functional bypass/regression. TA permissions
  remain unchanged; R5 attribution and broader R12 recovery remain separate.
- [ ] **R5: Reject ambiguous AI result attribution.** The normalizer can reuse
  duplicate/invalid results to fill missing indexes. See
  `normalizeAIGradingResults` in `backend/internal/worker/grading_handler.go`.
  Acceptance: duplicate, missing, invalid, and extra results are rejected without
  misattributing any score/comment; valid results map exactly to the batch.
- [ ] **R6: Preserve submissions during file replacement.** The old object is
  deleted before replacement succeeds. See `handleAttachmentUpdate` in
  `backend/internal/service/submission_service.go`.
  Acceptance: upload/database failures preserve the original content/reference;
  successful replacement and later cleanup do not delete shared or current files.
- [ ] **R7: Limit AI-config updates to one owned record.** Repository updates filter
  only by user ID. See `UpdateAIConfig` in
  `backend/internal/repository/ai_repository.go` and
  `backend/internal/service/ai_service.go`.
  Acceptance: editing one config leaves the user's other configs and other users'
  configs unchanged; nonexistent/unauthorized IDs are rejected.

## 3. Domain correctness and recovery

- [ ] **R8: Preserve deadline precision and correct lateness reporting.** Assignment
  DTOs drop time/timezone; statistics use `UpdatedAt` and omit student extensions.
  See `backend/internal/models/convertor.go` and
  `backend/internal/service/service.go`.
  Acceptance: timestamp round-trips preserve instants; grading does not change
  submission lateness; start/due/close and override boundaries agree with the UI.
- [ ] **R9: Enforce server-side input rules.** Cover assignment date ordering,
  positive points, manual score bounds, submission start time, and file requirements.
  See `backend/internal/models/forms.go` and assignment/submission services.
  Acceptance: invalid direct API requests fail without partial writes; legitimate
  boundary values follow the documented domain policy.
- [ ] **R10: Make attachment permissions consistent.** Downloads omit assignment
  visibility/start checks; non-student membership checks omit active status. See
  `backend/internal/service/attachment_service.go` and
  `backend/internal/repository/member_repository.go`.
  Acceptance: hidden/upcoming assignments and inactive memberships obey the agreed
  access policy, while owners and authorized course staff retain appropriate access.
- [ ] **R11: Define and implement TA permissions.** The role exists but has no
  permission-map entry. See `backend/internal/models/models.go` and handler gates.
  Acceptance: approved teacher/TA/student actions have explicit allow/deny tests.
- [ ] **R12: Make partial writes and job recovery reliable.** Review grade/comment
  atomicity, assignment edits, queue-enqueue failures, early worker failures, and
  retries. See assignment/submission services, `backend/internal/queue/`, and
  `backend/internal/worker/`.
  Acceptance: failed steps leave recoverable state; jobs reach honest terminal
  states; retries do not duplicate feedback or corrupt grades.
  F3 also identified `asynq.Timeout(60)` in the queue client: the locked API accepts
  a `time.Duration`, so this is not a declared 60-second limit. Verify the intended
  units and actual task deadline behavior before choosing a fix; runtime impact
  has not been tested.
- [ ] **R13: Define attempt and statistics semantics.** Multiple submissions are
  permitted and summary selection is not an explicit latest-attempt policy. Review
  the `NotStarted` calculation and intended membership/override uniqueness.
  Acceptance: deterministic summaries and documented constraints; any schema change
  is reviewed separately with migration and rollback considerations.

Decisions to settle alongside the relevant item: allowed attempts; upload versus
submission timestamps; extension effects on due/close dates; manual-role precedence
and regrade authority (R4 settles human-over-AI precedence); TA rights; student Python execution; empty submissions;
and whether text answers or multi-section distribution are actually required.
Do not infer these requirements from the historical plan.

## 4. Configuration and security follow-up

Infrastructure, migration, and credential operations need a scoped review and
authorization for their side effects; existing user authorization still applies.

- [ ] **O1: Repair verified setup inconsistencies.** Review MinIO endpoint URL and
  ignored SSL argument, missing documented `JUDGE0_API_KEY`, conflicting Compose
  files, and local/container Node versions. Check Next/ESLint compatibility without
  assuming an upgrade is necessary. See Dockerfiles, Compose, storage, runner, and
  frontend manifests/lockfile.
  After repairs, validate the README's full-stack steps in a disposable Docker
  environment; Docker was unavailable on PATH during F2. Compose overrides the
  MinIO endpoint, and mounted Judge0 credentials are not interpolated from `.env`.
- [ ] **O2: Review build contexts and credential defaults.** Add appropriate
  `.dockerignore` coverage and replace tracked credential defaults with safe
  examples where appropriate. `backend/.env copy` contains default credentials;
  their use in production is unverified. Never print or commit real credentials.
- [ ] **O3: Review operational boundaries.** Plan startup migration ownership,
  readiness checks, queue persistence, Judge0 isolation/image reproducibility, and
  backup/restore verification. Both entry points currently run AutoMigrate.
  Do not start the stack against an existing database just to test setup.
- [ ] **O4: Investigate unverified security questions.** Trace AI-config ownership
  when assigning configurations, custom provider URL restrictions, and log access/
  retention. Confirm actual deployment topology and backup practices. Record
  findings as confirmed, disproved, or unresolved before proposing fixes.

## 5. CI, feature work, and optional tools

- [ ] **Q1: Add minimal CI after baseline checks are understood.** Frontend lint,
  typecheck, and build; Go test/vet; focused integration tests using disposable
  services. Acceptance: checks need no production credentials or real student data.
- [ ] **Q2: Add a small critical-flow E2E suite when setup is deterministic.** Cover
  teacher assignment creation -> student submission -> teacher grading, plus denied
  access. Mock university authentication and AI providers. Start with Chromium;
  add frontend test dependencies only for a demonstrated behavior-testing need.
- [ ] **Q3: Resume one bounded feature.** Define its acceptance criteria, implement
  within the existing architecture, run relevant checks, and review the diff.
- [ ] **Q4: Extract large components only when a task benefits.** AI settings,
  assignment dialogs, and grading are hotspots. Prefer focused extraction of tested
  behavior over a broad architecture rewrite or speculative dependency cleanup.
- [ ] **T1: Decide whether navigation tooling is needed.** Start with `rg`, Git,
  focused source reads, existing editor navigation, and short docs. If repeated
  discovery is expensive, trial only one of Graphify, Serena, or Repomap on a UI
  bug, permission change, and grading-impact task. Measure discovery time, reads,
  tool calls, correctness, usage when available, and setup/maintenance overhead.
  Acceptance: repeatable net benefit; verify Go/TS support and external placement
  of all configuration/output/state. Otherwise retain the existing toolset.

No AI-specific tools, generated graphs, indexes, memories, or source exports belong
inside the application repository, even if ignored. Do not install overlapping MCP
servers or add AI tooling to application infrastructure. Repomix is only an optional
scoped export tool, with output outside the repository and reviewed before sharing.

## Completion and handoff

Mark an item complete only when its acceptance checks are met. Add a concise entry
below with the item ID, commit/reference when available, commands actually run,
outcomes, and remaining risks. Keep private session notes and logs outside the repo.

- 2026-10-24: Added project instructions and this checklist; linked from README.
  Documentation only. `git diff --check` passed; both README documentation targets
  exist; the two new documents have no trailing whitespace. Application tests were
  not run for this documentation change. No application fixes, dependency
  installations, or infrastructure changes made.
- 2026-10-24 (F1): Restored locked frontend dependencies and ran lint, typecheck,
  build, and test commands in an external disposable copy. Go test/vet and API/
  worker builds passed; no Go tests exist. Frontend typecheck/build passed; lint
  failed with 39 errors and 79 warnings; `npm test` reported a missing script.
  Full environment, commands, limits, and follow-up are in `quality-baseline.md`.
  Hash comparison of 193 copied files found no source/manifest changes. No services
  were started, dependencies upgraded, or application fixes applied.
- 2026-10-24 (F2): Replaced root/frontend setup docs with source-backed native and
  conditional Compose instructions, complete runtime-variable coverage, startup
  side effects, and separately tracked configuration blockers. Frontend dev HTTP
  smoke passed in the external workspace; the server was stopped afterward. No
  backend, worker, database, or container was started. Full-stack validation is
  still pending O1-O3; application/configuration fixes remain unchecked. Nine
  relative links resolved, ten PowerShell blocks parsed, and `git diff --check`
  passed. Verification details are in `quality-baseline.md`.
- 2026-10-24 (F3): Added `architecture.md` and `domain-rules.md`; linked them from
  README and AGENTS.md. Rechecked routes, models, middleware, services, worker, and
  relevant locked dependency signatures. Recorded queue timeout-unit verification
  under R12. All 67 relative links resolved; code fences were balanced; changed
  documents had no trailing whitespace; `git diff --check` passed. Application tests
  were not rerun for this documentation-only change. No runtime services, package
  installations, application fixes, policy decisions, or ADRs were introduced.
- 2026-10-24 (F4): Replaced redundant `any` casts with existing/shared response
  types, corrected tag callback/effect dependencies, removed the duplicate upload
  ref, and escaped the due-date dialog apostrophe. Final lint passed (0 errors,
  74 warnings; five removed, none added); production build and post-build typecheck
  passed. Ten headless Chrome scenarios passed with synthetic data and intercepted
  requests; no runtime errors. Harness/logs stayed outside the repository and its
  temporary route was removed before final checks. No dependencies, rule settings,
  backend, database, infrastructure, or grading/permission policies changed.
  - 2026-10-24 (R1): Scoped all six nested submission actions to course, assignment,
    and submission before handler execution; reads now use active course role and
    application error statuses. Reproduced the vulnerable requests before patching;
    all 82 final PostgreSQL-backed HTTP scenarios passed, as did full Go test/vet,
    both binary builds, and diff checks. Independent source review found no confirmed
    bypass or regression. Fixture schemas were removed and the disposable external
    PostgreSQL cluster stopped. No existing database, API/worker startup, dependencies,
    infrastructure, or TA/admin/grading policy changed. Standalone downloads remain R10.
- 2026-10-25 (R2): Added the requirement to commit completed fixes with their
  tests/documentation. Replaced timestamp/filename storage keys with shared random
  UUIDs across all providers; local writes cannot truncate an existing object.
  Reproduced collisions before patching, then passed eleven provider regression
  scenarios, full Go test/vet, both binary builds, and patch checks. Independent
  review found no confirmed bypass or regression. R1 database tests skipped, and
  frontend checks were not repeated. No dependencies, production services, databases,
  or infrastructure were changed. Legacy normal keys remain supported; historical overwritten bytes
  and failure-safe replacement are outside R2.
- 2026-10-25 (R3): Recorded the approved eight-hour/re-login policy and minimum
  32-byte secret. Removed package-init key capture, enforced HS256 and expiry,
  validated identity claims, handled signing errors, and checked API config before
  DB startup. Config/middleware tests, invalid-startup probes, all 82 R1 HTTP cases,
  full Go test/vet, and both binary builds passed. Independent review found no
  confirmed backend bypass/regression. Browser auto-navigation and refresh/revocation
  are not implemented; expired requests return 401. Only disposable test data used.
- 2026-10-25 (R4): Enforced the approved human-over-AI rule in selection and a
  write-time conditional score+feedback transaction. Retired destructive forced
  resets; failed regrades preserve old AI output and successful replacements update
  AI feedback in place. Twenty PostgreSQL/fake-provider cases passed, including real
  contention, rollback, concurrent saves, and retries. Full Go tests including R1,
  vet, both builds, and diff checks passed. Independent review found no confirmed
  functional issue; verification gaps it identified were covered. TA permissions,
  R5 mapping, and broader R12 recovery were not changed.
