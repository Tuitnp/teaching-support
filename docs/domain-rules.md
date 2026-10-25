# Domain rules and open decisions

Source reviewed: 2026-10-24; storage entry updated 2026-10-25. **Current** below means verified in source, not an
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
| Due / close / start | Stored assignment timestamps; due is used in lateness reports, close in acceptance, start in student assignment reads |
| Override | A student's `ExtendedDueDate`; currently replaces both returned due and close dates and the submission cutoff |
| Submission / attachment | Submission record with optional file reference; attachment metadata and storage bytes are separate resources |
| Grade / comment | Nullable numeric score and grading role versus authored feedback with independent visibility |
| Grading job | Assignment/teacher database progress record, distinct from the Asynq task that performs work |

Canonical entities and permissions: [models.go](../backend/internal/models/models.go).
Do not interpret model role constants as evidence that every role has permissions.

## Current behavior and repair direction

| Area | Current, source-verified behavior | Repair direction / open policy |
| --- | --- | --- |
| Login/session | University `student` type creates a student; other types create teachers. User-approved R3 policy: HS256 sessions expire eight hours after login, with no refresh. Expiry is required, so legacy tokens are rejected. API validates a nonblank secret of at least 32 bytes after dotenv and before DB startup. Browser logout clears local state; expired requests return 401 and users must log out/sign in again. | R3 verified. Automatic browser expiry navigation and server-side session revocation are not implemented. Changing the configured secret invalidates existing signatures. |
| Authorization | Course middleware requires active membership. Global roles gate templates, AI settings, and Python; course roles gate course work. Nested submission read/update/grade/comment routes additionally bind the submission to both URL parents; reads use course role, and student reads/updates retain ownership checks. | R1 verified with PostgreSQL-backed HTTP tests. Preserve this route guard when adding callers; ID-only helpers do not independently authorize access. Standalone attachment download remains R10. |
| TA/admin | `ta` and `admin` constants exist, but neither has a permission-map entry. TA grading helper allows ungraded or AI-graded work, but handler gates deny TA grading. | R11: decide TA actions and human-grade precedence before making the permission map operational for them. |
| Visibility/scheduling | Student assignment queries require `visible=true` and `start_date <= now`. Submission creation/update checks visibility and effective close, but not start. DTOs serialize dates as `YYYY-MM-DD`. | R8/R9: preserve instants and enforce agreed boundaries consistently. |
| Extensions | Override must not precede assignment due time or be earlier than approximately now minus one minute. Existing override is deleted before replacement. Override lookup errors are ignored during submission cutoff calculation. | R8/R12: settle extension/cutoff semantics and failure behavior; avoid losing an existing override on failure. |
| Submission content | Backend permits creation without a file; a supplied file is capped at 5 MiB. The UI requires a file for a new submission. No text-answer column exists. Multiple records are permitted per student/assignment. | R9/R13: decide empty submissions and attempts. Text answers are an unconfirmed feature, not an existing requirement. |
| Replacement/storage | R2 uses random UUID keys. Approved R6 policy retains replaced bytes/attachment rows for separate cleanup; fileless updates detach the reference, and grades/comments stay unchanged. Upload precedes a transaction containing metadata insert, owner-scoped reference update, and response load. Only the reference column is updated. No storage deletion occurs on this path, even after DB errors. | R6 verified for failures, shared IDs/keys, and concurrency. Failed DB transactions can leave staged bytes without metadata; uncertain commit outcomes are not compensated by deleting storage. Cleanup must be designed separately, including existing explicit attachment/assignment deletion paths. Historical overwritten bytes are not repaired. |
| Reusable files | Assignment/template creation allows up to five combined existing/new attachments; general uploaded files have a 10 MiB cap. Existing attachment IDs must belong to the requester. | R9: consistently enforce intended limits across create/edit paths. Do not assume create checks cover updates. |
| Download access | Owner may download. Submission files allow non-student course memberships without an active-status filter. Assignment files allow active membership in a referencing course without visibility/start checks. | R10: settle pre-release attachment access and apply active membership rules consistently. |
| Manual grading | Teacher helper permits grading; TA helper has conditional behavior. Scores are `int16` without assignment-relative bounds. Grade+comment operations are separate writes. | R9/R12: agreed bounds and atomic behavior. Do not invent a grading scale beyond the assignment maximum. |
| Comments | Manual upsert keys on submission/author/role; AI upsert keys on submission/AI role. Visibility is stored separately; student detail preloads only visible comments. AI score and its immediately visible feedback commit together. | Decide feedback release and AI review requirements. Manual score+comment atomicity remains R12. |
| AI grading | Approved R4 policy: normal jobs fill only rows with point and grader null; forced jobs may also replace AI grades. Human/unknown/unattributed scores remain protected at write time. R5 requires one explicit, unique in-range index per input, with matching optional ID, before saving the batch. Missing/null/repeated fields and extra/missing results are rejected. | R4 concurrency and R5 mapping verified. Previous AI output survives failed replacement; existing AI comment IDs are retained. Normalization still clamps scores and supplies fallback comments. Broader R12 recovery remains open. |
| Job state/retries | Pending/processing jobs block another start for the same assignment/teacher. DB job is created/reset before enqueue; failures can leave stale state. Early worker exit with zero attached submissions does not complete/fail the job. | R12: recoverable states, truthful status, retry safety, and explicit timeout units. |
| Statistics | Active students drive summaries. A map keeps one encountered submission per student, without an explicit attempt-selection policy. Lateness uses `UpdatedAt` and base due date; grading can affect that timestamp. | R8/R13: stable submission timing, overrides, deterministic attempt selection, and meaningful incomplete/not-started counts. |
| Python execution | Global teacher permission only; Judge0 receives limits and network disabled. No student execution permission exists in the map. | Decide student/TA availability before changing execution permissions. |
| AI config updates | Global `ai` permission plus owned config/credential checks apply. The actual update predicates both owner and requested config ID and rejects zero matches; siblings and credential records are unchanged. Nonempty name/model fields update; temperature always updates, including zero. | R7 verified through HTTP and repository checks, including owner/deletion races. Existing omitted credential IDs still fail with generic 500; explicit ID validation remains R9. |

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

## Decisions to resolve with the user before behavior changes

1. **Attempts/content:** one mutable submission or multiple attempts? Which attempt
   supplies grades/statistics? Are empty new submissions or text answers allowed?
   R6 explicitly retains fileless removal and existing grades/comments on replacement.
2. **Time policy:** exact start/due/close inclusivity, authoritative display timezone,
   and extension effects on due/close; define the timestamp that measures lateness.
3. **Roles:** allowed TA/admin actions, privacy of member data, and whether students
   or TAs may execute Python. Do not infer policy from UI controls.
4. **Grades:** teacher versus TA precedence, explicit regrade authority, and whether
   changed submission content invalidates prior grades. Human-over-AI precedence and
   forced AI-only replacement were approved and verified under R4.
5. **Feedback:** when scores/comments become visible and whether AI output requires
   teacher approval before publication.
6. **Reuse/access:** whether files of hidden/upcoming assignments may be downloaded,
   and whether multi-section distribution is needed beyond per-course template use.
7. **Sessions/jobs:** server-side session revocation and intended grading timeout, retry,
   cancellation, and recovery behavior.

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
default test fixtures. No policy or runtime behavior was changed by this reference.
