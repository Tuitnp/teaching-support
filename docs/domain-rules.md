# Domain rules and open decisions

Source reviewed: 2026-10-24; R1-R9 entries updated 2026-10-25. **Current** below means verified in source, not an
approved product rule or runtime-tested guarantee. **Repair direction** refers to
the accepted [checklist](development-checklist.md). **Decision needed** is an open
question; these documents do not choose a policy or authorize its implementation.
Use the [architecture map](architecture.md) for entry points and API contracts.

## Vocabulary

| Term | Current meaning |
| --- | --- |
| User role | Global `User.Role`; loaded by auth middleware, independent of course membership |
| Course role / status | `CourseMember.Role` plus `active,inactive,withdraw,drop`; active membership gates nested course routes |
| Course / section | Each `Course` has section/semester metadata; assignments belong to a specific course |
| Assignment / template | Scheduled course work versus creator-owned reusable content; reuse creates a separate assignment |
| Due / close / start | Stored assignment timestamps; due is used in lateness reports; inclusive start/close bound submission acceptance and start gates student assignment reads |
| Override | A student's `ExtendedDueDate`; sets effective due, with effective close equal to the later of original close and extended due |
| Submission / attachment | Submission record with optional file reference; attachment metadata and storage bytes are separate resources |
| Grade / comment | Nullable numeric score and grading role versus authored feedback with independent visibility |
| Grading job | Assignment/teacher database progress record, distinct from the Asynq task that performs work |

Canonical entities and permissions: [models.go](../backend/internal/models/models.go).
Do not interpret model role constants as evidence that every role has permissions.

## Current behavior and repair direction

| Area | Current, source-verified behavior | Repair direction / open policy |
| --- | --- | --- |
| Login/session | University `student` type creates a student; other types create teachers. User-approved R3 policy: HS256 sessions expire eight hours after login, with no refresh. Expiry is required, so legacy tokens are rejected. API validates a nonblank secret of at least 32 bytes after dotenv and before DB startup. Browser logout clears local state; expired requests return 401 and users must log out/sign in again. | R3 verified. Automatic browser expiry navigation and server-side session revocation are not implemented. Changing the configured secret invalidates existing signatures. |
| Authorization | Course middleware requires active membership. Global roles gate templates, AI settings, and Python; course roles gate course work. Nested submission read/update/grade/comment routes additionally bind the submission to both URL parents; reads use course role, and student reads/updates retain ownership checks. | R1 verified with PostgreSQL-backed HTTP tests. Preserve this route guard when adding callers; ID-only helpers do not independently authorize access. Standalone attachment downloads use the R10 reference policy below. |
| TA/admin | R11 adds explicit TA grants for active-course reads, individual files, authored feedback and restricted initial/AI grading. TAs cannot alter any existing manual/unknown/unattributed grade or another author's feedback. Teacher/student grants are preserved; admin remains without grants. | The user delegated these choices. [Full permission matrix and reasons](permissions.md); HTTP and real concurrent-write regressions verify scope and authority under lock. Global roles remain independent of course roles. Frontend permission UI alignment is separately tracked. |
| Visibility/scheduling | Approved R8 policy: assignment schedule DTOs use full UTC RFC3339 timestamps. Display/edit uses Asia/Bangkok (UTC+07:00). Student reads require visibility/start; submission create/update includes exact start and effective close. Exact due is on time. Untouched UI edits retain the original timestamp, including sub-millisecond fractions. | R8/R9 regression coverage: creation requires all dates and start < due <= close; edits validate merged date/point values before side effects and again under the assignment lock. |
| Extensions | Override must not precede assignment due time or be earlier than approximately now minus one minute. R8 sets due to extended due and close to `max(original close, extended due)`, without shortening access. Submission override lookup errors fail before upload/write. R12b updates an existing override in place under the assignment lock. | R8 settles timing semantics. R12b failure tests preserve the prior extension; R13 uses descending UUID to choose the canonical legacy row; no timestamps or schema constraints are added. |
| Submission content | Approved R9 policy: creation requires one nonempty file; supplied replacement files must also be nonempty and <= 5 MiB. Fileless updates retain the approved R6 detach behavior. No text-answer column exists. R13 permits one current mutable record per student/assignment, with earlier records retained as student-read-only history; repeated POST uploads replace the current file without resetting grades/comments. | R9 verified for file presence, byte/size boundaries and malformed uploads. R13 selects one mutable current record plus retained history. Text answers are an unconfirmed feature, not an existing requirement. |
| Replacement/storage | R2 uses random UUID keys. Approved R6 policy retains replaced bytes/attachment rows for separate cleanup; fileless updates detach the reference, and grades/comments stay unchanged. Upload precedes a transaction containing metadata insert, owner-scoped reference update, and response load. Only the reference column is updated. No storage deletion occurs on this path, even after DB errors. | R6 verified for failures, shared IDs/keys, and concurrency. Failed DB transactions can leave staged bytes without metadata; uncertain commit outcomes are not compensated by deleting storage. Cleanup must be designed separately, including existing explicit attachment/assignment deletion paths. Historical overwritten bytes are not repaired. |
| Reusable files | R9 validates all assignment/template create/edit attachments before uploads/writes: <= five combined references/files, <= 10 MiB per uploaded attachment, distinct valid UUID references that exist and belong to the requester. General uploads check the complete batch before uploading its first file. Optional AI configs must exist and be owned. | R9 verified. General attachments may still be empty files; the nonempty rule applies to submission files. R12b makes assignment create/edit metadata, scalars, tags, joins and response load transactional after staging uploads; staged bytes may remain after failure. Template/library multi-step writes remain separate follow-up. |
| Download access | Approved R10: uploaders retain their own files, including detached/template-only files, regardless of course state. Nonowners need active membership in the reference's course. Staff with `file:read_all` may read handouts/submissions regardless of assignment release; students need a handout reference to a visible assignment with a known start at or before now. Due/close do not revoke read access. Any independently authorized reference grants access. | PostgreSQL-backed download regression tests cover ownership, release, active status, role scope, reuse and fail-closed errors. Nested submission reads and bulk ZIP rules are unchanged by R10. Shared-file deletion/cleanup remains separate. |
| Manual grading | Approved R9 policy: assignment/template maxima are positive int16 integers; manual grades explicitly supply an integer in [0, assignment maximum]. Reducing an assignment maximum below any existing submission grade is rejected; grades are never rescaled. Grade writes and assignment edits serialize on the assignment row, rechecking current bounds. R12a commits manual grade plus authored feedback in one transaction; concurrent authored upserts serialize on the assignment. | R9/R11 bounds and precedence remain enforced under locks. PostgreSQL failure/retry/concurrency checks verify R12a rollback and feedback identity. |
| Comments | Manual upsert keys on submission/author/role; AI upsert keys on submission/AI role. Visibility is stored separately; student detail preloads only visible comments. AI score and its immediately visible feedback commit together. | Decide feedback release and AI review requirements. R12a makes manual score+authored feedback transactional. |
| AI grading | Approved R4 policy: normal jobs fill only rows with point and grader null; forced jobs may also replace AI grades. Human/unknown/unattributed scores remain protected at write time. R5 requires one explicit, unique in-range index per input, with matching optional ID, before saving the batch. Missing/null/repeated fields and extra/missing results are rejected. | R4 concurrency and R5 mapping verified. Previous AI output survives failed replacement; existing AI comment IDs are retained. Normalization still clamps scores and supplies fallback comments. R12 now guards job identity, retries and cancellation; see [job lifecycle](grading-jobs.md). |
| Job state/retries | Approved R12: explicit 30-minute attempt, three retries with existing backoff, pending between retries, failed after exhaustion, zero work completed. Tasks bind to committed job UUIDs and current attempt tokens. Forced retries may regenerate committed AI results; normal retries retain saved grades. | [Recovery and rollout rules](grading-jobs.md). Enqueue/commit failures are recoverable; reads reconcile Redis crash/archive state without treating outages as failure. DB/provider/queue-adapter regression tests pass; real Redis restart testing remains Q1. |
| Statistics | R13 counts each unique active student once. Submitted has a current file; Incomplete has a current record without a file; Not Started has no record. Current identity is latest record creation, then descending UUID. History is excluded from reports/ZIP/AI, even after detaching the current file. Lateness still uses current upload time and effective due; unknown upload times remain explicit. | [Approved R13 policy](attempt-policy.md). Identical memberships collapse; conflicts deny affected access and make reports return 409. Graded zero remains distinct from missing grades. Schema changes and historical cleanup require separate review. |
| Python execution | Global teacher permission only; Judge0 receives limits and network disabled. R11 deliberately grants neither global TAs nor students execution. Course roles do not change independent global execution permissions. | Student execution remains an open policy. |
| AI config updates | Global `ai` permission plus owned config/credential checks apply. The actual update predicates both owner and requested config ID and rejects zero matches; siblings and credential records are unchanged. Nonempty name/model fields update; temperature always updates, including zero. | R7 verified through HTTP and repository checks, including owner/deletion races. R9 rejects missing/malformed credential IDs with 400 before querying; UUID validation also covers affected assignment/submission/template/attachment/config lookups. |

## Source pointers for changes

- Identity/access: [auth service](../backend/internal/service/auth_service.go),
  [JWT generation](../backend/internal/service/jwt.go),
  [middleware](../backend/internal/middleware/middleware.go),
  [submission handler](../backend/internal/handler/submission_handler.go).
- Assignment visibility/overrides: [assignment service](../backend/internal/service/assignment_service.go),
  [assignment queries](../backend/internal/repository/assignment_repository.go),
  [DTO conversion](../backend/internal/models/convertor.go).
- Files/submissions: [submission service](../backend/internal/service/submission_service.go),
  [attachment service](../backend/internal/service/attachment_service.go),
  [membership queries](../backend/internal/repository/member_repository.go),
  [shared helpers](../backend/internal/service/service.go),
  [storage adapters](../backend/internal/storage).
- Grades/comments/reporting: [submission queries](../backend/internal/repository/submission_repository.go),
  [comment service](../backend/internal/service/comment_service.go),
  [summary helpers](../backend/internal/service/service.go),
  [worker](../backend/internal/worker/grading_handler.go),
  [AI repository](../backend/internal/repository/ai_repository.go),
  [queue client](../backend/internal/queue/client.go).
- Execution: [run handler](../backend/internal/handler/run_handler.go),
  [Judge0 adapter](../backend/internal/service/run_service.go).

## Approved R8 timing and reporting policy

- API assignment start/due/close timestamps retain their instants and available
  fractional precision in UTC (`Z`). The database retains its native timestamp
  precision. Bangkok is the fixed display/edit zone; date-only values are no longer
  emitted for assignment scheduling. Explicit UI changes use JavaScript millisecond
  precision; leaving a date untouched preserves the original API string.
- Visibility plus `start <= now <= effective close` admits submission create/update.
  `upload time <= effective due` is on time. Extension due replaces base due;
  effective close is the later of original close and extension due.
- The attachment metadata's `CreatedAt` represents the successful current upload,
  including replacement. Grading and comments never supply an upload timestamp.
  R8 originally selected the latest current file across attempts. R13 supersedes
  that selection with one latest-created record, then descending UUID for creation
  ties/unknowns. A detached current record remains current and is Incomplete; old
  files do not replace it in reports. Upload time still determines lateness.
- Missing historical attachment/timestamp data stays unknown: `timing unavailable`
  counts as submitted and can still be graded. Known creation timestamps precede unknown ones for current-record identity;
  creation ties use the stable ID tie-break. This cannot
  reconstruct historical upload order. No backfill, schema change, or migration is
  part of R8. R13 settles duplicate membership/extension and current-record selection rules.

## Approved R9 input policy

- Assignment creation requires positive integer maximum points and all nonzero
  dates, ordered `start < due <= close`. Templates also require positive maximum
  points. The existing `int16` representation caps points at 32767. For edits,
  omitted date/point fields retain existing values; explicit zero/invalid values
  are rejected. Other existing PUT field/replacement semantics are unchanged.
- Both manual grade endpoints require an explicit integer score, including zero;
  absent/null, negative, fractional, overflow and above-maximum scores are rejected
  before score/comment changes. Reductions of maximum points check all existing
  grades, including detached attempts; equality is allowed. Historical invalid
  scores are not backfilled or rescaled by unrelated edits or maximum increases.
- Submission creation requires exactly one file containing at least one byte and
  at most 5 MiB. A supplied replacement follows the same limits; no-file updates
  still detach only, preserving grades/comments. No extension/type restriction
  or text-answer feature is introduced.
- Assignment/template attachment batches allow at most five combined owned
  references/uploads and 10 MiB per uploaded file, inclusively. General attachment
  batches retain the same five-file/10 MiB limits. Validation checks the complete
  batch and references before any upload/write. Malformed UUIDs return 400;
  canonical uppercase UUIDs remain valid, while URNs are rejected before SQL.
- Assignment edits and manual/AI grade persistence lock the assignment before
  submission rows. Bounds are rechecked under that short DB transaction, outside
  provider/storage calls. Eligible AI results above a newly reduced maximum fail
  without changing score/feedback; missing/ineligible/protected rows retain R4's
  no-op behavior. R12 now makes full assignment edits and manual score/comment
  operations transactional and guards job recovery; see [grading jobs](grading-jobs.md).

## Decisions to resolve with the user before behavior changes

1. **Content:** R13 settles one mutable submission plus retained student-read-only
   history; see [attempt policy](attempt-policy.md). R6 retains fileless removal and
   existing grades/comments on replacement; R9 requires nonempty creation/replacement
   uploads. Text answers are unconfirmed and no text-answer field is added.
2. **Historical timing:** R8 time policy is approved above. Recovering unknown upload
   timestamps requires trustworthy historical evidence, not grading timestamps.
3. **Roles:** R11 settles TA actions, roster access using existing fields, and no
   global-TA Python grant; see [permission decisions](permissions.md). Admin actions,
   further roster privacy restrictions and student execution remain open.
4. **Grades:** R11 settles teacher-over-TA authority and teacher correction of all
   manual grades. Whether changed submission content invalidates prior grades
   remains open. Human-over-AI precedence and forced AI-only replacement were
   approved and verified under R4.
5. **Feedback:** when scores/comments become visible and whether AI output requires
   teacher approval before publication.
6. **Reuse:** R10 settles hidden/upcoming attachment access as above. Whether
   multi-section distribution is needed beyond per-course template use remains open.
7. **Sessions/jobs:** R12 approves 30 minutes per attempt and three queue-backoff
   retries: pending between attempts, failed only after exhaustion, zero work a
   completed no-op. Forced retries may regenerate AI results; normal retries retain
   saved grades. Session revocation and job cancellation remain open.

Open decisions do not prevent documenting current behavior. Before implementing a
dependent change, ask for the missing policy instead of turning an observed bug,
historical proposal, or UI assumption into a requirement. Record an ADR only when
an actual consequential decision is made.

## Regression evidence expected with fixes

Use each checklist item's acceptance checks. Prioritize allow/deny tests for the
full resource hierarchy, same-name upload isolation, preservation on failure, exact
AI-result mapping, human-grade protection, score/date boundaries, and retry safety.
These are future verification requirements; F1 found no business-rule tests.
Do not use real student records, production services, or paid provider calls as
default test fixtures. Approved fixes and their measured evidence are recorded in the checklist and quality baseline.
